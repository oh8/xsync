package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 主配置结构
type Config struct {
	NodeID       string        `yaml:"node_id"`
	Role         string        `yaml:"role"` // "master" or "slave"
	Key          string        `yaml:"key"`  // AES-256密钥
	UDPPort      int           `yaml:"udp_port"`
	MonitorPaths []MonitorPath `yaml:"monitor_paths"` // Master专用
	MasterAddr   string        `yaml:"master_addr"`   // Slave专用
	SyncPath     string        `yaml:"sync_path"`     // Slave专用
	WebServer    *WebConfig    `yaml:"web_server"`    // Web服务配置（Master专用）
}

// WebConfig Web服务配置
type WebConfig struct {
	Enabled   bool   `yaml:"enabled"`    // 是否启用Web服务
	Port      int    `yaml:"port"`       // Web服务端口，默认8081
	Username  string `yaml:"username"`   // Basic Auth用户名
	Password  string `yaml:"password"`   // Basic Auth密码
	UploadDir string `yaml:"upload_dir"` // 上传目录，默认uploads
}

// MonitorPath Master监控路径配置
type MonitorPath struct {
	Path   string   `yaml:"path"`
	Slaves []string `yaml:"slaves"`
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 从环境变量读取密钥（安全考虑）
	if envKey := os.Getenv("XSYNC_KEY"); envKey != "" {
		config.Key = envKey
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate 验证配置有效性
func (c *Config) Validate() error {
	if c.NodeID == "" {
		return fmt.Errorf("node_id不能为空")
	}

	if c.Role != "master" && c.Role != "slave" {
		return fmt.Errorf("role必须是master或slave")
	}

	if len(c.Key) != 32 {
		return fmt.Errorf("AES密钥必须是32字节")
	}

	if c.UDPPort <= 0 || c.UDPPort > 65535 {
		return fmt.Errorf("UDP端口必须在1-65535范围内")
	}

	if c.Role == "master" {
		if len(c.MonitorPaths) == 0 {
			return fmt.Errorf("Master节点必须配置monitor_paths")
		}
	} else {
		if c.MasterAddr == "" {
			return fmt.Errorf("Slave节点必须配置master_addr")
		}
		if c.SyncPath == "" {
			return fmt.Errorf("Slave节点必须配置sync_path")
		}
	}

	return nil
}

// IsMaster 判断是否为Master节点
func (c *Config) IsMaster() bool {
	return c.Role == "master"
}

// IsSlave 判断是否为Slave节点
func (c *Config) IsSlave() bool {
	return c.Role == "slave"
}
