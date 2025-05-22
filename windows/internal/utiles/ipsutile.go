package utiles

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

func GetCurrentIPs(domain string) (string, error) {
	addrs, err := net.LookupIP(domain)
	if err != nil {
		return "", fmt.Errorf("DNS查询失败: %w", err)
	}
	ipv4, ipv6 := "", ""
	for _, ip := range addrs {
		if ip.To4() != nil && ipv4 == "" {
			ipv4 = ip.String()
		} else if len(ip) == net.IPv6len && ipv6 == "" {
			ipv6 = ip.String()
		}
		if ipv4 != "" && ipv6 != "" {
			break
		}
	}
	return ipv4 + "," + ipv6, nil
}

func GetLocalIPs(ipFilePath string) (string, error) {
	if _, err := os.Stat(ipFilePath); errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	data, err := os.ReadFile(ipFilePath)
	if err != nil {
		return "", fmt.Errorf("读取IP文件失败: %w", err)
	}
	return string(data), nil
}

func GetServerIPs(serverIPsUrl string) (string, error) {
	resp, err := http.Get(serverIPsUrl)
	if err != nil {
		return "", fmt.Errorf("服务器状态查询失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("无效状态码: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	serverIPs := string(body)
	return serverIPs, nil
}

// 查询并等待服务器planet文件更新
func WaitForPlanetFileUpdate(serverIPsUrl, serverIPsPath string, checkInterval int) (string, error) {
	for {
		serverIPs, err := GetServerIPs(serverIPsUrl)
		if err != nil {
			return "", fmt.Errorf("获取服务器IP失败: %v", err)
		}
		localServerIPs, err := GetLocalIPs(serverIPsPath)
		if err != nil {
			return "", fmt.Errorf("获取本地服务器IP失败: %v", err)
		}
		if localServerIPs == serverIPs {
			log.Printf("服务器文件未更新,%d秒后重试", checkInterval)
			time.Sleep(time.Duration(checkInterval) * time.Second)
			continue
		}
		return serverIPs, nil
	}
}
func ReplacePlanetFile(planetPath string) error {

	bakPath := planetPath + ".bak"
	// 判断文件是否存在
	if _, err := os.Stat(bakPath); errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(planetPath, bakPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("备份失败: %w", err)
		}
	}

	if err := os.Rename(planetPath+".tmp", planetPath); err != nil {
		return fmt.Errorf("替换文件失败: %w", err)
	}
	return nil
}

func Download(url, planetPath string) error {
	tmpFile := planetPath + ".tmp"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("状态码错误: %d", resp.StatusCode)
	}
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	return nil
}

func SaveNewIPs(ips, ipsFilePath, serverIPs, serverIPsFilePath string) error {
	if err := os.WriteFile(ipsFilePath, []byte(ips), 0644); err != nil {
		return fmt.Errorf("保存IP失败: %v", err)
	}
	if err := os.WriteFile(serverIPsFilePath, []byte(serverIPs), 0644); err != nil {
		os.Remove(ipsFilePath)
		return fmt.Errorf("保存服务器IP失败: %v", err)
	}
	return nil
}
