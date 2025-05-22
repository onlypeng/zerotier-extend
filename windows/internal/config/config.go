package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 结构体
type Config struct {
	Version        int            `yaml:"version"`
	AppConfig      AppConfig      `yaml:"app"`
	ServerConfig   ServerConfig   `yaml:"server"`
	ZeroTierConfig ZeroTierConfig `yaml:"zerotier"`
	ServiceConfig  ServiceConfig  `yaml:"service"`
}

// AppConfig 应用程序相关配置
type AppConfig struct {
	LogFilePath   string `yaml:"logFilePath"`
	LogMaxLines   int    `yaml:"logMaxLines"`
	IPFilePath    string `yaml:"ipFilePath"`
	ServerIPsPath string `yaml:"serverIPsPath"`
	CheckInterval int    `yaml:"checkInterval"`
}

// ServerConfig 服务器相关配置
type ServerConfig struct {
	Domain    string `yaml:"domain"`
	IPsURL    string `yaml:"ipsUrl"`
	PlanetURL string `yaml:"planetUrl"`
}

// ZeroTierConfig 结构体（ZeroTier 相关配置）
type ZeroTierConfig struct {
	ServiceName string `yaml:"serviceName"`
	PlanetPath  string `yaml:"planetPath"`
}

// Service 配置（扩展服务）
type ServiceConfig struct {
	Name        string         `yaml:"name"`
	DisplayName string         `yaml:"displayName"`
	Description string         `yaml:"description"`
	Options     ServiceOptions `yaml:"options"`
}

// ServiceOptions 结构体（扩展服务的选项）
type ServiceOptions struct {
	OnFailure              string `yaml:"onFailure"`
	FailureResetPeriod     int    `yaml:"failureResetPeriod"`
	FailureRestartInterval int    `yaml:"failureRestartInterval"`
}

// FixRelativePaths 修复配置文件中以Path结尾的字段的相对路径
func FixRelativePaths(cfg interface{}, absoluteDir string) error {
	// 使用反射遍历结构体
	cfgValue := reflect.ValueOf(cfg).Elem()
	cfgType := cfgValue.Type()

	for i := 0; i < cfgValue.NumField(); i++ {
		field := cfgValue.Field(i)
		fieldType := cfgType.Field(i)

		// 如果字段名称以Path结尾且是字符串类型
		if strings.HasSuffix(fieldType.Name, "Path") && field.Kind() == reflect.String {
			path := field.String()
			if !filepath.IsAbs(path) {
				// 如果是相对路径，拼接绝对目录
				absPath := filepath.Join(absoluteDir, path)
				field.SetString(absPath)
			}
		}

		// 如果字段是结构体类型，递归处理
		if field.Kind() == reflect.Struct {
			// 获取结构体指针
			nestedCfg := field.Addr().Interface()
			if err := FixRelativePaths(nestedCfg, absoluteDir); err != nil {
				return err
			}
		}
	}

	return nil
}

// LoadConfig 读取 YAML 配置文件并修复路径
func LoadConfig(configPath string) (*Config, error) {
	// 获取可执行文件所在目录
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("无法获取可执行文件路径: %v", err)
	}
	// 获取可执行文件所在目录
	absoluteDir := filepath.Dir(exePath)
	if !filepath.IsAbs(configPath) {
		// 如果是相对路径，拼接配置文件所在目录
		configPath = filepath.Join(absoluteDir, configPath)
	}
	// 判断配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("配置文件不存在: %v", err)
	}

	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		return nil, fmt.Errorf("解析 YAML 失败: %v", err)
	}

	// 修复相对路径
	if err := FixRelativePaths(&config, absoluteDir); err != nil {
		return nil, fmt.Errorf("修复路径失败: %v", err)
	}

	return &config, nil
}
