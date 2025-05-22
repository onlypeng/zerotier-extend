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
	mgr         *mgr.Mgr     // 保存服务管理器连接
	service     *mgr.Service // 保存服务连接
}

// NewWindowsServiceManager 实例化
func NewWindowsServiceManager(name string) (*WindowsServiceManager, error) {
	wm := &WindowsServiceManager{ServiceName: name}

	// 初始化时连接服务管理器
	m, err := mgr.Connect()
	if err != nil {
		return nil, fmt.Errorf("无法连接到服务管理器: %v", err)
	}
	wm.mgr = m

	// 初始化时连接服务
	s, err := m.OpenService(name)
	if err != nil {
		m.Disconnect() // 如果服务连接失败，关闭服务管理器连接
		return nil, fmt.Errorf("打开服务失败: %v", err)
	}
	wm.service = s

	return wm, nil
}

// Close 关闭服务管理器和服务连接
func (wm *WindowsServiceManager) Close() error {
	if wm.service != nil {
		wm.service.Close()
	}
	if wm.mgr != nil {
		return wm.mgr.Disconnect()
	}
	return nil
}

// IsInstalled 判断服务是否已安装
func (wm *WindowsServiceManager) IsInstalled() (bool, error) {
	if wm.service == nil {
		return false, fmt.Errorf("服务未连接")
	}
	return true, nil
}

// waitForStatus 等待服务到达目标状态
func (wm *WindowsServiceManager) waitForStatus(targetState svc.State, timeout time.Duration) error {
	if wm.service == nil {
		return fmt.Errorf("服务未连接")
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeoutCh := time.After(timeout)

	for {
		select {
		case <-ticker.C:
			status, err := wm.service.Query()
			if err != nil {
				return fmt.Errorf("查询状态失败: %v", err)
			}
			if status.State == targetState {
				return nil
			}
		case <-timeoutCh:
			return fmt.Errorf("等待服务状态超时")
		}
	}
}

// Status 返回服务当前状态
func (wm *WindowsServiceManager) Status() (string, error) {
	m, err := mgr.Connect()
	if err != nil {
		return "", fmt.Errorf("无法连接到服务管理器: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(wm.ServiceName)
	if err != nil {
		return "服务不存在", nil // 明确返回服务不存在的状态
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return "未知", fmt.Errorf("无法查询服务状态: %v", err)
	}

	switch status.State {
	case svc.Stopped:
		return "已停止", nil
	case svc.Running:
		return "正在运行", nil
	default:
		return fmt.Sprintf("状态码: %d", status.State), nil
	}
}

// Start 启动服务
func (wm *WindowsServiceManager) Start() error {
	// 检查服务是否存在
	installed, err := wm.IsInstalled()
	if err != nil {
		return fmt.Errorf("检查服务是否存在失败: %v", err)
	}
	if !installed {
		return fmt.Errorf("服务 %s 不存在", wm.ServiceName)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("无法连接到服务管理器: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(wm.ServiceName)
	if err != nil {
		return fmt.Errorf("打开服务失败: %v", err)
	}
	defer s.Close()

	err = s.Start()
	if err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}

	// 等待运行状态
	return wm.waitForStatus(svc.Running, 10*time.Second)
}

// Stop 停止服务
func (wm *WindowsServiceManager) Stop() error {
	// 检查服务是否存在
	installed, err := wm.IsInstalled()
	if err != nil {
		return fmt.Errorf("检查服务是否存在失败: %v", err)
	}
	if !installed {
		return fmt.Errorf("服务 %s 不存在", wm.ServiceName)
	}

	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("无法连接到服务管理器: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(wm.ServiceName)
	if err != nil {
		return fmt.Errorf("打开服务失败: %v", err)
	}
	defer s.Close()

	status, err := s.Control(svc.Stop)
	if err != nil {
		return fmt.Errorf("停止服务失败: %v", err)
	}

	// 等待停止状态
	for i := 0; i < 10; i++ {
		if status.State == svc.Stopped {
			return nil
		}
		time.Sleep(1 * time.Second)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("查询停止状态失败: %v", err)
		}
	}
	return fmt.Errorf("服务未在超时内停止")
}

// Restart 重启服务
func (wm *WindowsServiceManager) Restart() error {
	// 检查服务是否存在
	installed, err := wm.IsInstalled()
	if err != nil {
		return fmt.Errorf("检查服务是否存在失败: %v", err)
	}
	if !installed {
		return fmt.Errorf("服务 %s 不存在", wm.ServiceName)
	}

	// 先停止服务
	err = wm.Stop()
	if err != nil {
		return fmt.Errorf("停止服务失败: %v", err)
	}

	// 再启动服务
	err = wm.Start()
	if err != nil {
		return fmt.Errorf("启动服务失败: %v", err)
	}

	return nil
}

// GetCurrentStatus 获取服务的当前状态
func (wm *WindowsServiceManager) GetCurrentStatus() (svc.State, error) {
	// 检查服务是否存在
	installed, err := wm.IsInstalled()
	if err != nil {
		return svc.Stopped, fmt.Errorf("检查服务是否存在失败: %v", err)
	}
	if !installed {
		return svc.Stopped, fmt.Errorf("服务 %s 不存在", wm.ServiceName)
	}

	m, err := mgr.Connect()
	if err != nil {
		return svc.Stopped, fmt.Errorf("无法连接到服务管理器: %v", err)
	}
	defer m.Disconnect()

	s, err := m.OpenService(wm.ServiceName)
	if err != nil {
		return svc.Stopped, fmt.Errorf("打开服务失败: %v", err)
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return svc.Stopped, fmt.Errorf("无法查询服务状态: %v", err)
	}

	return status.State, nil
}
