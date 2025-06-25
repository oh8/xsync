package slave

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xsync/protocol"
	"xsync/transport"
)

// Config 配置结构（从main包导入的类型定义）
type Config struct {
	NodeID       string        `yaml:"node_id"`
	Role         string        `yaml:"role"`
	Key          string        `yaml:"key"`
	UDPPort      int           `yaml:"udp_port"`
	MonitorPaths []MonitorPath `yaml:"monitor_paths"`
	MasterAddr   string        `yaml:"master_addr"`
	SyncPath     string        `yaml:"sync_path"`
	WebServer    *WebConfig    `yaml:"web_server"`
}

// WebConfig Web服务配置
type WebConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Port      int    `yaml:"port"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	UploadDir string `yaml:"upload_dir"`
}

// MonitorPath Master监控路径配置
type MonitorPath struct {
	Path   string   `yaml:"path"`
	Slaves []string `yaml:"slaves"`
}

// IsMaster 判断是否为Master节点
func (c *Config) IsMaster() bool {
	return c.Role == "master"
}

// IsSlave 判断是否为Slave节点
func (c *Config) IsSlave() bool {
	return c.Role == "slave"
}

// Slave 从节点
type Slave struct {
	config    *Config
	transport transport.Transport
	done      chan bool
	stats     *SlaveStats
}

// SlaveStats 从节点统计信息
type SlaveStats struct {
	ReceivedPackets int64
	AppliedFiles    int64
	Errors          int64
	LastSync        time.Time
}

// NewSlave 创建Slave节点
func NewSlave(cfg *Config) (*Slave, error) {
	if !cfg.IsSlave() {
		return nil, fmt.Errorf("配置不是Slave节点")
	}

	// 创建传输层
	transport := transport.NewQUICTransport([]byte(cfg.Key))

	s := &Slave{
		config:    cfg,
		transport: transport,
		done:      make(chan bool),
		stats:     &SlaveStats{},
	}

	return s, nil
}

// Start 启动Slave节点
func (s *Slave) Start() error {
	log.Printf("启动Slave节点: %s", s.config.NodeID)

	// 确保同步目录存在
	if err := os.MkdirAll(s.config.SyncPath, 0755); err != nil {
		return fmt.Errorf("创建同步目录失败: %v", err)
	}

	// 启动传输层监听
	if err := s.transport.Listen(s.config.UDPPort, s.handleSyncPacket); err != nil {
		return fmt.Errorf("启动传输层监听失败: %v", err)
	}

	log.Printf("Slave节点启动完成，监听端口: %d，同步目录: %s", s.config.UDPPort, s.config.SyncPath)
	
	// 启动后延迟2秒发送全量同步请求，确保Master已准备好
	go func() {
		time.Sleep(2 * time.Second)
		if err := s.RequestFullSync(); err != nil {
			log.Printf("请求全量同步失败: %v", err)
		}
	}()
	
	return nil
}

// handleSyncPacket 处理同步数据包
func (s *Slave) handleSyncPacket(packet *protocol.SyncPacket, remoteAddr string) error {
	s.stats.ReceivedPackets++
	s.stats.LastSync = time.Now()

	log.Printf("接收同步包: %s %s from %s", packet.Op, packet.Path, remoteAddr)

	// 构建完整文件路径
	fullPath := filepath.Join(s.config.SyncPath, packet.Path)

	// 根据操作类型处理
	switch packet.Op {
	case "CREATE", "MODIFY":
		return s.handleCreateOrModify(fullPath, packet.Content)
	case "DELETE":
		return s.handleDelete(fullPath)
	case "SYNC_REQUEST":
		return s.handleSyncRequest(remoteAddr)
	case "HEARTBEAT":
		// 心跳包，无需处理
		return nil
	default:
		s.stats.Errors++
		return fmt.Errorf("未知的操作类型: %s", packet.Op)
	}
}

// handleCreateOrModify 处理创建或修改文件
func (s *Slave) handleCreateOrModify(fullPath string, content []byte) error {
	// 确保父目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.stats.Errors++
		return fmt.Errorf("创建目录失败 %s: %v", dir, err)
	}

	// 检查文件是否已存在且内容相同
	if existingContent, err := ioutil.ReadFile(fullPath); err == nil {
		if string(existingContent) == string(content) {
			log.Printf("文件内容未变化，跳过: %s", fullPath)
			return nil
		}
	}

	// 写入文件内容
	if err := ioutil.WriteFile(fullPath, content, 0644); err != nil {
		s.stats.Errors++
		return fmt.Errorf("写入文件失败 %s: %v", fullPath, err)
	}

	s.stats.AppliedFiles++
	log.Printf("文件同步成功: %s (%d bytes)", fullPath, len(content))
	return nil
}

// handleDelete 处理删除文件
func (s *Slave) handleDelete(fullPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		log.Printf("文件已不存在，跳过删除: %s", fullPath)
		return nil
	}

	// 删除文件
	if err := os.Remove(fullPath); err != nil {
		s.stats.Errors++
		return fmt.Errorf("删除文件失败 %s: %v", fullPath, err)
	}

	s.stats.AppliedFiles++
	log.Printf("文件删除成功: %s", fullPath)

	// 尝试删除空目录
	s.cleanupEmptyDirs(filepath.Dir(fullPath))

	return nil
}

// cleanupEmptyDirs 清理空目录
func (s *Slave) cleanupEmptyDirs(dir string) {
	// 不要删除根同步目录
	if dir == s.config.SyncPath || dir == "." || dir == "/" {
		return
	}

	// 检查目录是否为空
	entries, err := ioutil.ReadDir(dir)
	if err != nil || len(entries) > 0 {
		return
	}

	// 删除空目录
	if err := os.Remove(dir); err == nil {
		log.Printf("删除空目录: %s", dir)
		// 递归检查父目录
		s.cleanupEmptyDirs(filepath.Dir(dir))
	}
}

// Stop 停止Slave节点
func (s *Slave) Stop() error {
	log.Printf("停止Slave节点: %s", s.config.NodeID)

	// 发送停止信号
	close(s.done)

	// 关闭传输层
	if err := s.transport.Close(); err != nil {
		return fmt.Errorf("关闭传输层失败: %v", err)
	}

	log.Printf("Slave节点已停止")
	return nil
}

// GetStats 获取统计信息
func (s *Slave) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"node_id":         s.config.NodeID,
		"role":            "slave",
		"sync_path":       s.config.SyncPath,
		"received_packets": s.stats.ReceivedPackets,
		"applied_files":   s.stats.AppliedFiles,
		"errors":          s.stats.Errors,
		"last_sync":       s.stats.LastSync.Format(time.RFC3339),
		"uptime":          time.Now().Format(time.RFC3339),
	}

	return stats
}

// SendHeartbeat 发送心跳到Master（可选功能）
func (s *Slave) SendHeartbeat() error {
	heartbeat := protocol.NewSyncPacket("HEARTBEAT", s.config.NodeID, nil)
	return s.transport.Send(s.config.MasterAddr, heartbeat)
}

// StartHeartbeat 启动心跳定时器
func (s *Slave) StartHeartbeat(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := s.SendHeartbeat(); err != nil {
					log.Printf("发送心跳失败: %v", err)
				}
			case <-s.done:
				return
			}
		}
	}()
}

// RequestFullSync 请求全量同步
func (s *Slave) RequestFullSync() error {
	log.Printf("请求全量同步从Master: %s", s.config.MasterAddr)
	syncRequest := protocol.NewSyncPacket("SYNC_REQUEST", s.config.NodeID, nil)
	return s.transport.Send(s.config.MasterAddr, syncRequest)
}

// handleSyncRequest 处理同步请求（实际上这个方法在Slave中不会被调用，因为Slave不接收SYNC_REQUEST）
func (s *Slave) handleSyncRequest(remoteAddr string) error {
	// Slave节点不处理SYNC_REQUEST，这个方法保留用于扩展
	return nil
}

// getLocalFileList 获取本地文件列表
func (s *Slave) getLocalFileList() (map[string]bool, error) {
	files := make(map[string]bool)
	
	err := filepath.Walk(s.config.SyncPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		if !info.IsDir() {
			// 获取相对路径
			relPath, err := filepath.Rel(s.config.SyncPath, path)
			if err != nil {
				return err
			}
			// 统一使用正斜杠
			relPath = strings.ReplaceAll(relPath, "\\", "/")
			files[relPath] = true
		}
		return nil
	})
	
	return files, err
}