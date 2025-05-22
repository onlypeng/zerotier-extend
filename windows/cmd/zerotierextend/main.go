package main

import (
	"fmt"
	"log"
	"os"
	"time"

	config "zerotierextend/internal/config"
	logger "zerotierextend/internal/logger"
	myservice "zerotierextend/internal/service"

	"github.com/kardianos/service"
)

func main() {

	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}
	log.Println("配置文件初始化成功")

	logFile, err := logger.InitLog(cfg.AppConfig.LogFilePath, cfg.AppConfig.LogMaxLines)
	if err != nil {
		log.Fatalf("设置日志失败: %v", err)
	}
	defer logFile.Close()
	log.Println("日志初始化成功")

	serviceConfig := cfg.ServiceConfig
	svcConfig := &service.Config{
		Name:        serviceConfig.Name,
		DisplayName: serviceConfig.DisplayName,
		Description: serviceConfig.Description,
		Option: service.KeyValue{
			"StartTimeout":           60 * time.Second,
			"OnFailure":              serviceConfig.Options.OnFailure,
			"FailureResetPeriod":     serviceConfig.Options.FailureResetPeriod,
			"FailureRestartInterval": serviceConfig.Options.FailureRestartInterval,
		},
	}
	prg, err := myservice.NewProgram(cfg)
	if err != nil {
		log.Printf("创建服务失败: %v", err)
		return
	}
	s, err := service.New(prg, svcConfig)

	if err != nil {
		log.Printf("创建服务失败: %v", err)
		return
	}

	// 处理命令行参数
	if len(os.Args) > 1 {
		cmd := os.Args[1]
		status, err := s.Status()
		if err != nil && err != service.ErrNotInstalled {
			log.Printf("获取服务状态失败: %v\n", err)
			return
		}
		if status == service.StatusUnknown {
			if cmd == "install" {
				log.Println("开始安装服务...")
				err = s.Install()
				if err != nil {
					log.Printf("安装服务失败: %v\n", err)
					return
				}
				log.Println("服务安装成功")
				return
			}
			log.Println("服务未安装")
			return
		}
		switch cmd {
		case "install":
			log.Println("服务已安装")
		case "uninstall":
			// 如果服务正在运行，先停止服务
			if status == service.StatusRunning {
				log.Println("服务正在运行，开始停止服务...")
				err = s.Stop()
				if err != nil {
					log.Printf("停止服务失败: %v\n", err)
					return
				}
				log.Println("服务已成功停止")
			}

			// 开始卸载服务
			log.Println("开始卸载服务...")
			err = s.Uninstall()
			if err != nil {
				log.Printf("卸载服务失败: %v\n", err)
				return
			}
			log.Println("服务卸载成功")
		case "start":
			if status == service.StatusRunning {
				log.Println("服务已经在运行")
				return
			}
			log.Println("开始启动服务...")
			err = s.Start()
			if err != nil {
				log.Printf("启动服务失败: %v\n", err)
			} else {
				log.Println("服务启动成功")
			}
		case "stop":
			if status == service.StatusStopped {
				log.Println("服务已经停止")
				return
			}
			log.Println("开始停止服务...")
			err = s.Stop()
			if err != nil {
				log.Printf("停止服务失败: %v\n", err)
			} else {
				log.Println("服务停止成功")
			}
		case "restart":
			if status == service.StatusStopped {
				log.Println("服务已经停止，无法重启")
				return
			}
			log.Println("开始重启服务...")
			err = s.Restart()
			if err != nil {
				log.Printf("重启服务失败: %v\n", err)
			} else {
				log.Println("服务重启成功")
			}
		default:
			log.Printf("未知命令: %s\n", cmd)
			fmt.Println("可用命令: install, uninstall, start, stop, restart")
		}
		return
	}
	// 没有参数，直接运行服务（交给 SCM 管理）
	err = s.Run()
	if err != nil {
		log.Fatalf("服务运行失败: %v", err)
	}
	fmt.Println("服务已停止")
}
