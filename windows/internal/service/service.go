package service

import (
	"fmt"
	"log"
	"time"
	"zerotierextend/internal/config"
	myutiles "zerotierextend/internal/utiles"

	"github.com/kardianos/service"
)

type ProgramImpl struct {
	exit            chan struct{}
	config          *config.Config
	zerotierService *myutiles.WindowsServiceManager
}

// 修改构造函数，注入配置：
func NewProgram(cfg *config.Config) (*ProgramImpl, error) {
	zerotierService, err := myutiles.NewWindowsServiceManager(cfg.ZeroTierConfig.ServiceName)
	if err != nil {
		return nil, fmt.Errorf("创建WindowsServiceManager失败\n %v", err)
	}
	return &ProgramImpl{
		exit:            make(chan struct{}),
		config:          cfg,
		zerotierService: zerotierService,
	}, nil
}

func (p *ProgramImpl) Start(s service.Service) error {
	go p.run()
	return nil
}
func (p *ProgramImpl) Stop(s service.Service) error {
	close(p.exit)
	return nil
}

func (p *ProgramImpl) doCheck(config *config.Config) {
	appConfig := config.AppConfig
	serverConfig := config.ServerConfig
	checkInterval := appConfig.CheckInterval

	// 1. 检查服务状态
	statusStr, err := p.zerotierService.Status()
	if err != nil || statusStr != "正在运行" {
		log.Printf("服务 %s 未运行，跳过本次检查\n", config.ZeroTierConfig.ServiceName)
		return
	}
	// 2. 获取当前IP
	currentIPs, err := myutiles.GetCurrentIPs(serverConfig.Domain)
	if err != nil {
		log.Printf("获取当前IP失败: %v\n", err)
		return
	}
	log.Printf("获取当前IP成功，当前IP: %v", currentIPs)
	// 3. 比较历史IP
	localIPs, err := myutiles.GetLocalIPs(appConfig.IPFilePath)
	if err != nil {
		log.Printf("获取本地IP失败: %v\n", err)
		return
	}
	log.Printf("获取本地IP成功，本地IP: %v", localIPs)
	if currentIPs == localIPs {
		log.Printf("IP未变化，跳过更新")
		return
	}
	log.Printf("检测到IP已变更，等待服务器文件更新")
	// 4. 等待服务器文件更新
	serverIPs, err := myutiles.WaitForPlanetFileUpdate(serverConfig.IPsURL, appConfig.ServerIPsPath, checkInterval)
	if err != nil {
		log.Printf("等待服务器文件更新失败: %v\n", err)
		return
	}
	log.Printf("服务器文件已更新，开始更新planet文件")
	// 5. 下载并planet文件
	if err := myutiles.Download(serverConfig.PlanetURL, config.ZeroTierConfig.PlanetPath); err != nil {
		log.Printf("下载planet文件失败: %v\n", err)
		return
	}
	log.Printf("下载planet文件成功")
	// 6. 替换planet文件
	if err := myutiles.ReplacePlanetFile(config.ZeroTierConfig.PlanetPath); err != nil {
		log.Printf("替换planet文件失败: %v\n", err)
		return
	}
	log.Printf("替换planet文件成功")
	// 7. 重启服务
	err = p.zerotierService.Restart()
	if err := p.zerotierService.Start(); err != nil {
		log.Printf("重启服务失败: %v\n", err)
		return
	}
	log.Printf("重启服务成功")
	// 8. 保存新IP记录
	if err := myutiles.SaveNewIPs(currentIPs, appConfig.IPFilePath, serverIPs, appConfig.ServerIPsPath); err != nil {
		log.Printf("保存新IP记录失败: %v\n", err)
		return
	}
	log.Printf("保存新IP记录成功")
	log.Printf("更新完成，%d秒后重新开始检测", checkInterval)
}
func (p *ProgramImpl) run() {
	config := p.config
	checkInterval := config.AppConfig.CheckInterval
	ticker := time.NewTicker(time.Duration(checkInterval) * time.Second)
	defer ticker.Stop()
	p.doCheck(p.config)
	for {
		select {
		case <-p.exit:
			fmt.Println("服务收到退出信号，停止检测循环")
			return
		case <-ticker.C:
			p.doCheck(p.config)
		}
	}
}
