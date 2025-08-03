package main

import (
	"fmt"
	"log"
	"os"
	"time"

	config "github.com/onlypeng/zerotier-extend/windows/internal/config"
	logger "github.com/onlypeng/zerotier-extend/windows/internal/logger"
	myservice "github.com/onlypeng/zerotier-extend/windows/internal/service"

	"github.com/kardianos/service"
)

func main() {

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
		return
	}

	logFile, err := logger.InitLog(cfg.AppConfig.LogFilePath, cfg.AppConfig.LogMaxLines)
	if err != nil {
		log.Fatalf("日志初始化失败: %v", err)
		return
	}
	defer logFile.Close()

	prg, err := myservice.NewProgram(cfg)
	if err != nil {
		log.Fatalf("创建服务失败: %v", err)
		return
	}

	svcConfig := cfg.ServiceConfig

	svcInstance, err := service.New(prg, &service.Config{
		Name:        svcConfig.Name,
		DisplayName: svcConfig.DisplayName,
		Description: svcConfig.Description,
		Option: service.KeyValue{
			"StartTimeout":           60 * time.Second,
			"OnFailure":              svcConfig.Options.OnFailure,
			"FailureResetPeriod":     svcConfig.Options.FailureResetPeriod,
			"FailureRestartInterval": svcConfig.Options.FailureRestartInterval,
		},
	})

	if err != nil {
		log.Fatalf("创建服务失败: %v", err)
		return
	}

	// 有命令行参数，处理服务命令
	if len(os.Args) > 1 {
		handleCommand(os.Args[1], svcInstance)
		return
	}

	// 作为系统服务运行
	if err := svcInstance.Run(); err != nil {
		log.Fatalf("服务运行失败: %v", err)
		return
	}
	log.Println("服务已停止")
}

func handleCommand(cmd string, svc service.Service) {
	status, err := svc.Status()
	if err != nil && err != service.ErrNotInstalled {
		log.Fatalf("获取服务状态失败: %v", err)
	}

	switch cmd {
	case "install":
		if status != service.StatusUnknown {
			log.Println("服务已安装")
			return
		}
		log.Println("开始安装服务...")
		if err := svc.Install(); err != nil {
			log.Fatalf("安装服务失败: %v", err)
		}
		log.Println("服务安装成功")

	case "uninstall":
		if status == service.StatusRunning {
			log.Println("服务正在运行，正在停止...")
			if err := svc.Stop(); err != nil {
				log.Fatalf("停止服务失败: %v", err)
			}
			log.Println("服务已停止")
		}
		log.Println("正在卸载服务...")
		if err := svc.Uninstall(); err != nil {
			log.Fatalf("卸载服务失败: %v", err)
		}
		log.Println("服务卸载成功")

	case "start":
		if status == service.StatusRunning {
			log.Println("服务已在运行中")
			return
		}
		log.Println("正在启动服务...")
		if err := svc.Start(); err != nil {
			log.Fatalf("启动服务失败: %v", err)
		}
		log.Println("服务启动成功")

	case "stop":
		if status == service.StatusStopped {
			log.Println("服务已停止")
			return
		}
		log.Println("正在停止服务...")
		if err := svc.Stop(); err != nil {
			log.Fatalf("停止服务失败: %v", err)
		}
		log.Println("服务停止成功")

	case "restart":
		if status == service.StatusStopped {
			log.Println("服务已停止，请直接启动服务")
			return
		}
		log.Println("正在重启服务...")
		if err := svc.Restart(); err != nil {
			log.Fatalf("重启服务失败: %v", err)
		}
		log.Println("服务重启成功")
	case "status":
		log.Println("服务状态:", status)
	default:
		log.Printf("未知命令: %s", cmd)
		fmt.Println("可用命令: install, uninstall, start, stop, restart,status")
	}
}
