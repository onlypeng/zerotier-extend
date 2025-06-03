package utiles

import (
	"fmt"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// WindowsServiceManager 提供对 Windows 服务的控制
type WindowsServiceManager struct {
	ServiceName string
	mgr         *mgr.Mgr
	service     *mgr.Service
}

// NewWindowsServiceManager 创建服务管理器并连接到指定服务
func NewWindowsServiceManager(name string) (*WindowsServiceManager, error) {
	wm := &WindowsServiceManager{ServiceName: name}
	if err := wm.connect(); err != nil {
		return nil, err
	}
	return wm, nil
}

// connect 连接到服务管理器和服务
func (wm *WindowsServiceManager) connect() error {
	var err error
	wm.mgr, err = mgr.Connect()
	if err != nil {
		return fmt.Errorf("连接服务管理器失败: %v", err)
	}
	wm.service, err = wm.mgr.OpenService(wm.ServiceName)
	if err != nil {
		wm.mgr.Disconnect()
		return fmt.Errorf("打开服务 %s 失败: %v", wm.ServiceName, err)
	}
	return nil
}

// Reconnect 用于在服务删除/重装后重新连接
func (wm *WindowsServiceManager) Reconnect() error {
	wm.Close()
	return wm.connect()
}

// Close 释放资源
func (wm *WindowsServiceManager) Close() error {
	if wm.service != nil {
		wm.service.Close()
		wm.service = nil
	}
	if wm.mgr != nil {
		err := wm.mgr.Disconnect()
		wm.mgr = nil
		return err
	}
	return nil
}

// checkReady 检查连接是否存在
func (wm *WindowsServiceManager) checkReady() error {
	if wm.mgr == nil || wm.service == nil {
		return fmt.Errorf("服务未连接")
	}
	return nil
}

// IsInstalled 判断服务是否存在
func (wm *WindowsServiceManager) IsInstalled() (bool, error) {
	if err := wm.checkReady(); err != nil {
		return false, err
	}
	return true, nil
}

// Start 启动服务（仅在非运行状态下启动）
func (wm *WindowsServiceManager) Start() error {
	if err := wm.checkReady(); err != nil {
		return err
	}
	state, err := wm.GetCurrentStatus()
	if err != nil {
		return err
	}
	if state == svc.Running {
		return fmt.Errorf("服务已在运行，无需重复启动")
	}
	if err := wm.service.Start(); err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}
	return wm.waitForStatus(svc.Running, 10*time.Second)
}

// Stop 停止服务（仅在运行状态下停止）
func (wm *WindowsServiceManager) Stop() error {
	if err := wm.checkReady(); err != nil {
		return err
	}
	state, err := wm.GetCurrentStatus()
	if err != nil {
		return err
	}
	if state == svc.Stopped {
		return fmt.Errorf("服务已停止，无需重复停止")
	}
	status, err := wm.service.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("发送停止命令失败: %v", err)
	}
	// 等待停止
	for i := 0; i < 10; i++ {
		if status.State == svc.Stopped {
			return nil
		}
		time.Sleep(time.Second)
		status, err = wm.service.Query()
		if err != nil {
			return fmt.Errorf("查询服务状态失败: %v", err)
		}
	}
	return fmt.Errorf("服务未在超时内停止")
}

// Restart 重启服务（避免重复操作）
func (wm *WindowsServiceManager) Restart() error {
	if err := wm.checkReady(); err != nil {
		return err
	}
	state, err := wm.GetCurrentStatus()
	if err != nil {
		return err
	}
	if state == svc.Running {
		if err := wm.Stop(); err != nil {
			return fmt.Errorf("重启失败（停止失败）: %v", err)
		}
	}
	// 即便之前是 Stopped 也照常启动
	if err := wm.Start(); err != nil {
		return fmt.Errorf("重启失败（启动失败）: %v", err)
	}
	return nil
}

// Status 返回服务状态字符串
func (wm *WindowsServiceManager) Status() (string, error) {
	state, err := wm.GetCurrentStatus()
	if err != nil {
		return "", err
	}
	switch state {
	case svc.Stopped:
		return "已停止", nil
	case svc.Running:
		return "正在运行", nil
	default:
		return fmt.Sprintf("未知状态码: %d", state), nil
	}
}

// GetCurrentStatus 返回当前 svc.State
func (wm *WindowsServiceManager) GetCurrentStatus() (svc.State, error) {
	if err := wm.checkReady(); err != nil {
		return svc.Stopped, err
	}
	status, err := wm.service.Query()
	if err != nil {
		return svc.Stopped, fmt.Errorf("查询服务状态失败: %v", err)
	}
	return status.State, nil
}

// waitForStatus 等待服务达到指定状态
func (wm *WindowsServiceManager) waitForStatus(target svc.State, timeout time.Duration) error {
	if err := wm.checkReady(); err != nil {
		return err
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			status, err := wm.service.Query()
			if err != nil {
				return fmt.Errorf("查询服务状态失败: %v", err)
			}
			if status.State == target {
				return nil
			}
		case <-timeoutCh:
			return fmt.Errorf("等待服务状态 %v 超时", target)
		}
	}
}
