#!/bin/sh
# Zerotier 自动更新守护进程（iStoreOS/OpenWrt 优化版 V5）
# 支持: OpenWrt(procd)、systemd、OpenRC、SysVinit

# === 配置 ===
CHECK_INTERVAL=60
LOG_MAX_LINES=3000
DOMAIN="域名"
SERVER_IPS_URL="https://域名/ips?key=SECRET_KEY"
SERVER_PLANET_URL="https://域名/planet?key=SECRET_KEY"
PLANET_PATH="planet文件位置"
ZEROTIER_SERVER="zerotier服务名"

EXTEND_SERVER="zerotier_extend"
LOG_FILE_PATH="log.txt"
LOCAL_IPS_PATH="ips.txt"
SERVER_IPS_PATH="server_ips.txt"

# 获取当前脚本的绝对路径
SCRIPT_DIR=$(dirname "$(readlink -f "$0")")

# 修正相对路径
paths="LOG_FILE_PATH LOCAL_IPS_PATH SERVER_IPS_PATH"
for var in $paths; do
    value=$(eval echo "\$$var")
    if [ "${value#/}" = "$value" ]; then
        eval "$var=\"${SCRIPT_DIR}/${value}\""
    fi
done

BACKUP_PLANET_PATH="${PLANET_PATH}.bak"

# === 日志 ===
log() {
  printf "%s - %s\n" "$(date +'%F %T')" "$*" | tee -a "$LOG_FILE_PATH"
  line_count=$(wc -l < "$LOG_FILE_PATH" 2>/dev/null || echo 0)
  if [ "$line_count" -gt "$LOG_MAX_LINES" ]; then
    tail -n $(($LOG_MAX_LINES - 100)) "$LOG_FILE_PATH" > "$LOG_FILE_PATH.tmp" && mv "$LOG_FILE_PATH.tmp" "$LOG_FILE_PATH"
  fi
}

# === 更新 Planet ===
update_planet() {
    log "检查 IP 变化"
    if command -v dig >/dev/null; then
        ipv4=$(dig +short A "$DOMAIN" | head -n1 || true)
        ipv6=$(dig +short AAAA "$DOMAIN" | head -n1 || true)
    elif command -v nslookup >/dev/null; then
        ipv4=$(nslookup "$DOMAIN" 2>/dev/null | awk '/^Address: / && $2 ~ /^[0-9.]+$/ {print $2; exit}' || true)
        ipv6=$(nslookup "$DOMAIN" 2>/dev/null | awk '/^Address: / && $2 ~ /:/ {print $2; exit}' || true)
    else
        log "错误：缺少 DNS 工具 (dig/nslookup)"
        return 1
    fi
    [ -z "$ipv4" ] && [ -z "$ipv6" ] && { log "获取 IP 失败"; return 1; }
    current="$ipv4"; [ -n "$ipv6" ] && current="$current,$ipv6"
    log "当前 IP: $current"

    old_local=
    [ -f "$LOCAL_IPS_PATH" ] && old_local=$(cat "$LOCAL_IPS_PATH")
    log "上次 IP: $old_local"

    [ "$current" = "$old_local" ] && { log "IP 未变，跳过本次更新"; return 0; }

    log "检测到 IP 变更，等待服务器端更新"
    old_server=
    [ -f "$SERVER_IPS_PATH" ] && old_server=$(cat "$SERVER_IPS_PATH")
    for _ in $(seq 1 60); do
        server_ips=$(curl --connect-timeout 5 --max-time 10 -fsSL "$SERVER_IPS_URL" || true)
        [ -n "$server_ips" ] && [ "$server_ips" != "$old_server" ] && break
        log "服务器文件未更新，等待${CHECK_INTERVAL}秒后重试"
        sleep $CHECK_INTERVAL
    done
    [ -z "$server_ips" ] && { log "获取服务器 IPS 失败"; return 1; }
    
    log "服务端已更新，开始更新 planet 文件"
    # 只有在还没备份的情况下才做备份
    if [ ! -f "$BACKUP_PLANET_PATH" ] && [ -f "$PLANET_PATH" ]; then
        cp "$PLANET_PATH" "$BACKUP_PLANET_PATH"
        log "已备份 planet 文件 $BACKUP_PLANET_PATH"
    fi
    tmp="${PLANET_PATH}.tmp"
    curl --connect-timeout 5 --max-time 30 -fsSL "$SERVER_PLANET_URL" -o "$tmp" || { log "planet 文件更新失败"; return 1; }
    mv "$tmp" "$PLANET_PATH" && log "已更新 planet 文件"
    
    manage_service restart "$ZEROTIER_SERVER" || return 1
    echo "$current" > "$LOCAL_IPS_PATH"
    echo "$server_ips" > "$SERVER_IPS_PATH"
    log "已更新IPS记录"
}

# === 守护 ===
run_daemon() {
    while true; do
        if update_planet; then
            log "${CHECK_INTERVAL}s 后继续检查"
        else
            log "更新失败，${CHECK_INTERVAL}s 后重试"
        fi
        sleep $CHECK_INTERVAL
    done
}
# 检测初始化系统类型
detect_init() {
  if [ -f /etc/openwrt_release ] || grep -qi openwrt /etc/os-release 2>/dev/null; then
    INIT=openwrt
  elif command -v systemctl >/dev/null 2>&1; then
    INIT=systemd
  elif command -v rc-update >/dev/null 2>&1; then
    INIT=openrc
  else
    INIT=sysv
  fi
}

detect_init

# 检查服务是否存在
check_service_exists() {
  svc="$1"
  case "$INIT" in
    systemd)
      systemctl list-unit-files --no-pager | grep -Eq "^${svc}\.service";
      ;;
    openrc)
      rc-status --all 2>/dev/null | grep -qw "$svc";
      ;;
    openwrt|sysv)
      [ -x "/etc/init.d/$svc" ];;
    *) return 1;;
  esac
}

# 获取服务状态 (running/stopped/unknown)
get_service_status() {
  svc="$1"
  case "$INIT" in
    systemd)
      status="$(systemctl is-active "$svc" 2>/dev/null)"
      case "$status" in
        active) echo "running" ;;
        inactive|failed|deactivating) echo "stopped" ;;
        *) echo "unknown" ;;
      esac
      ;;
    openrc)
      rc-service "$svc" status 2>/dev/null | grep -q started && echo "running" || echo "stopped"
      ;;
    openwrt|sysv)
      status_out=$("/etc/init.d/$svc" status 2>/dev/null)
      echo "$status_out" | grep -q running && echo "running" && return 0
      echo "$status_out" | grep -q inactive && echo "stopped" && return 0
      echo "unknown"
      ;;
    *)
      echo "unknown"
      ;;
  esac
}

# 服务管理: start/stop/restart/status
manage_service() {
  action="$1"; svc="$2"

  # 重启逻辑
  if [ "$action" = restart ]; then
    manage_service stop "$svc" || return 1
    sleep 5
    manage_service start "$svc" || return 1
    return 0
  fi
  # 查询状态
  if [ "$action" = status ]; then
    get_service_status "$svc"
    return 0
  fi

  # 开始/停止前检查状态
  status="$(get_service_status "$svc")"
  case "$action" in
    start)
      [ "$status" = running ] && { log "服务 $svc 已在运行"; return 0; } ;;
    stop)
      [ "$status" != running ] && { log "服务 $svc 未在运行"; return 0; } ;;
  esac

  # 执行动作，带重试
  retries=3; delay=5; i=1; success=1
  while [ $i -le $retries ]; do
    case "$INIT" in
      systemd)
        systemctl "$action" "$svc" && success=0 && break;;
      openrc)
        rc-service "$svc" "$action" && success=0 && break;;
      openwrt|sysv)
        "/etc/init.d/$svc" "$action" && success=0 && break;;
      *) log "错误：不支持的服务管理器"; return 1;;
    esac
    log "$action 第 $i 次失败，等待 ${delay}s 重试..."
    sleep $delay; i=$((i+1))
  done

  if [ $success -eq 0 ]; then
    log "服务 $svc $action 成功"
    return 0
  else
    log "错误：服务 $svc $action 失败"
    return 1
  fi
}

# 安装服务
install_service() {
  case "$INIT" in
    openwrt)
      SVC_SCRIPT="/etc/init.d/$EXTEND_SERVER"
      cat > "$SVC_SCRIPT" <<-EOF
#!/bin/sh /etc/rc.common
USE_PROCD=1
START=99
DEPENDS="network"
start_service() {
  procd_open_instance
  procd_set_param command /bin/sh "$SCRIPT_NAME"
  procd_set_param respawn
  procd_set_param stdout 1
  procd_set_param stderr 1
  procd_close_instance
}
EOF
      chmod +x "$SVC_SCRIPT"
      "$SVC_SCRIPT" enable
      ;;
    systemd)
      UNIT_FILE="/etc/systemd/system/${EXTEND_SERVER}.service"
      cat > "$UNIT_FILE" <<-EOF
[Unit]
Description=ZerotierExtend Daemon
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
ExecStart=/bin/sh "$SCRIPT_NAME"
Restart=always

[Install]
WantedBy=multi-user.target
EOF
      systemctl daemon-reload
      systemctl enable "$EXTEND_SERVER"
      manage_service start "$EXTEND_SERVER"
      ;;
    openrc)
      SVC_SCRIPT="/etc/init.d/$EXTEND_SERVER"
      cat > "$SVC_SCRIPT" <<-EOF
#!/sbin/openrc-run
description="ZerotierExtend Daemon"
command="/bin/sh"
command_args="$SCRIPT_NAME"
pidfile="/var/run/${EXTEND_SERVER}.pid"
EOF
      chmod +x "$SVC_SCRIPT"
      rc-update add "$EXTEND_SERVER" default
      ;;
    sysv)
      SVC_SCRIPT="/etc/init.d/$EXTEND_SERVER"
      cat > "$SVC_SCRIPT" <<-EOF
#!/bin/sh
### BEGIN INIT INFO
# Provides:          $EXTEND_SERVER
# Required-Start:    $remote_fs $syslog
# Required-Stop:     $remote_fs $syslog
# Default-Start:     2 3 4 5
# Default-Stop:      0 1 6
# Short-Description: ZerotierExtend
### END INIT INFO
case "$1" in
  start)
    /bin/sh "$SCRIPT_NAME" &
    ;;
  stop)
    pkill -f "$SCRIPT_NAME"
    ;;
  restart)
    \$0 stop; \$0 start
    ;;
  *)
    echo "Usage: \$0 {start|stop|restart}"; exit 1
    ;;
esac
exit 0
EOF
      chmod +x "$SVC_SCRIPT"
      ;;
  esac
  log "安装完成: $INIT"
}

# 卸载服务 + 恢复 Planet
uninstall_service() {
    restore_planet() {
        [ -f "$BACKUP_PLANET_PATH" ] && mv "$BACKUP_PLANET_PATH" "$PLANET_PATH" &&
        log "已恢复备份 Planet 文件"
        manage_service restart "$ZEROTIER_SERVER"
    }
    case "$INIT" in
        systemd)
        manage_service stop "$EXTEND_SERVER"
        systemctl disable "$EXTEND_SERVER" || true
        rm -f /etc/systemd/system/${EXTEND_SERVER}.service
        systemctl daemon-reload
        restore_planet
        ;;
        openrc)
        manage_service stop "$EXTEND_SERVER"
        rc-update del "$EXTEND_SERVER" default || true
        rm -f /etc/init.d/$EXTEND_SERVER
        restore_planet
        ;;
        openwrt)
        manage_service stop "$EXTEND_SERVER"
        /etc/init.d/$EXTEND_SERVER disable || true
        rm -f /etc/init.d/$EXTEND_SERVER
        restore_planet
        ;;
        sysv)
        manage_service stop "$EXTEND_SERVER"
        rm -f /etc/init.d/$EXTEND_SERVER
        restore_planet
        ;;
        *)
        log "未检测到已安装的服务，跳过卸载"
        ;;
    esac
    log "卸载完成: $INIT"
}

# === 入口 ===
SCRIPT_NAME=$(readlink -f "$0")
if [ $# -eq 0 ]; then
    run_daemon
else
    case "$1" in
    start|stop|restart|status) manage_service "$1" "$EXTEND_SERVER" ;;
    install)       install_service ;;
    uninstall)     uninstall_service ;;
    *) echo "只支持: $0 {start|stop|restart|status|install|uninstall}"; exit 1 ;;
    esac
fi
