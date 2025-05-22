# ZeroTier Extend

该扩展用户对[[xubiaolin/docker-zerotier-planet](https://github.com/xubiaolin/docker-zerotier-planet)]项目搭建的planet服务器进行扩展，使其支域名解析，并对其他相应系统客户端进行扩展，使其增加域名变动自动更新功能，因采用第三方实时监测域名变更而更新planet文件并重启服务，会存在短暂网络中断，在意着请勿使用，暂未支持Android。

## 脚本使用教程

### 一. server

该目录脚本文件用于对[[xubiaolin/docker-zerotier-planet](https://github.com/xubiaolin/docker-zerotier-planet)]docker容器进行扩展，使其支持域名变更后自动触发编译moon和planet文件

**使用方法 1：**

1. 创建zerotier-planet持久目录
2. 复制server文件到docker持久目录中
3. 请根据实际情况修改运行下面命令，必须设置DOMAIN变量，用于客户端使用

```
 docker run -d \
   --name zerotier-planet  \
   --restart always \
   -p 9994:9994 \
   -p 9994:9994/udp \
   -p 3443:3443 \
   -p 4000:4000 \
   -e ZT_PORT=9994 \
   -e API_PORT=3443 \
   -e DOMAIN=DNS域名 \
   -e FILE_SERVER_PORT=4000 \
   -e SECRET_KEY=请手动设置key或自动生成 \
   -v /持久目录/dist:/app/dist \
   -v /持久目录/ztncui:/app/ztncui \
   -v /持久目录/config:/app/config \
   -v /持久目录/zerotier-one:/var/lib/zerotier-one \
   -v /持久目录/entrypoint.sh:/app/entrypoint.sh \
   -v /持久目录/update_moon_planet.sh:/app/update_moon_planet.sh \
   xubiaolin/zerotier-planet:latest
```

**使用方法 2：**
直接使用第三方编译好的的容器

1. 创建zerotier-planet持久目录
2. 请根据实际情况修改运行下面命令，必须设置DOMAIN变量，用于客户端使用

```
 docker run -d \
   --name zerotier-planet  \
   --restart always \
   -p 9994:9994 \
   -p 9994:9994/udp \
   -p 3443:3443 \
   -p 4000:4000 \
   -e ZT_PORT=9994 \
   -e API_PORT=3443 \
   -e DOMAIN=DNS域名 \
   -e FILE_SERVER_PORT=4000 \
   -e SECRET_KEY=请手动设置key或自动生成 \
   -v /持久目录/dist:/app/dist \
   -v /持久目录/ztncui:/app/ztncui \
   -v /持久目录/config:/app/config \
   -v /持久目录/zerotier-one:/var/lib/zerotier-one \
   onlypeng/zerotier-planet:latest
```

### 二. linux

该目录脚本文件用于对常见linux、openwrt系统安装的zerotier客户端plante文件根据域名更改进行更新

**使用方法：**

1. 复制linux中脚本到任意目录，openwrt建议创建并放入/etc/config/zerotier_extend目录中
2. 修改脚本中配置变量:
| 变量              | 说 明                                                                                                           | 默认值 |
   | ----------------- | --------------------------------------------------------------------------------------------------------------- | ------ |
   | CHECK_INTERVAL    | 检测间隔时间                                                                                                    | 60秒   |
   | LOG_MAX_LINES     | 日志最大保留行数,最低300行                                                                                      | 3000   |
   | DOMAIN            | 检测的域名                                                                                                      | 必填   |
   | SERVER_IPS_URL    | 验证IP文件下载地址<br />http://域名/ips?key=服务端SECRET_KEY                                                    | 必填   |
   | SERVER_PLANET_URL | planet文件下载地址<br /> http://域名/planet?key=服务端SECRET_KEY                                                | 必填   |
   | PLANET_PATH       | planet文件路径<br />linux存放位置：/var/lib/zerotier-one/planet <br />openwrt存放位置： /etc/config/zero/planet | 必填   |
   | ZEROTIER_SERVER   | zerotier服务名称<br />linux一般为：zerotier-one <br />openwrt一般为：zerotier                                   | 必填   |
3. 确保脚本具有可执行权限，可以通过以下命令设置：chmod +x 脚本文件
4. 执行命令 ./update_planet.sh install 安装本服务
5. 执行命令 ./update_planet.sh start 启用服务
6. 执行命令 ./update_planet.sh stop停用服务
7. 卸载命令 ./update_planet.sh uninstall 卸载本服务

### 三. windows

该目录脚本文件用于对windows系统安装的zerotier客户端plante文件根据域名更改进行更新

**使用方法：**

1. 复制windows中script目录中zerotierextend.bat、update_planet.exe，configs中config.yaml文件到任意目录，建议创建并放入C:\ProgramData\ZeroTier\Extend目录中。注：ProgramData是隐藏目录
2. 修改config.yaml配置文件：
   | 变量                 | 说明                                                            | 默认值                             |
   | -------------------- | --------------------------------------------------------------- | ---------------------------------- |
   | app.checkInterval    | 检测间隔时间                                                    | 60秒                               |
   | app.logMaxLines      | 日志最大保留行数,最低300行                                      | 3000                               |
   | server.domain        | 检测域名                                                        | 必填                               |
   | server.ipsUrl        | 验证IP文件下载地址 <br />http://域名/ips?key=服务端SECRET_KEY    | 必填                               |
   | server.planetUrl     | planet文件下载地址 <br />http://域名/planet?key=服务端SECRET_KEY | 必填                               |
   | zerotier.serviceName | planet服务名称                                                  | ZeroTierOneService                 |
   | zerotier.planetPath  | planet文件路径                                                  | C:/ProgramData/ZeroTier/One/planet |
3. 使用管理员权限运行zerotierextend.bat进行安装、卸载、启动、停止等操作。
