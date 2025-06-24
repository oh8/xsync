package master

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xsync/config"
	"xsync/protocol"
	"xsync/transport"
	"xsync/watcher"
	"xsync/webserver"
)

// Master 主节点
type Master struct {
	config    *config.Config
	transport transport.Transport
	watchers  map[string]*watcher.FileWatcher
	webServer *webserver.WebServer
	mutex     sync.RWMutex
	done      chan bool
}

// NewMaster 创建Master节点
func NewMaster(cfg *config.Config) (*Master, error) {
	if !cfg.IsMaster() {
		return nil, fmt.Errorf("配置不是Master节点")
	}

	// 创建传输层
	transport := transport.NewQUICTransport([]byte(cfg.Key))

	m := &Master{
		config:    cfg,
		transport: transport,
		watchers:  make(map[string]*watcher.FileWatcher),
		done:      make(chan bool),
	}

	// 如果启用了Web服务，创建Web服务器
	if cfg.WebServer != nil && cfg.WebServer.Enabled {
		ws, err := webserver.NewWebServer(cfg.WebServer)
		if err != nil {
			return nil, fmt.Errorf("创建Web服务器失败: %v", err)
		}
		m.webServer = ws
	}

	return m, nil
}

// Start 启动Master节点
func (m *Master) Start() error {
	log.Printf("启动Master节点: %s", m.config.NodeID)

	// 启动传输层监听（用于接收Slave的心跳等）
	if err := m.transport.Listen(m.config.UDPPort, m.handlePacket); err != nil {
		return fmt.Errorf("启动传输层监听失败: %v", err)
	}

	// 启动Web服务器（如果启用）
	if m.webServer != nil {
		if err := m.webServer.Start(); err != nil {
			return fmt.Errorf("启动Web服务器失败: %v", err)
		}
	}

	// 为每个监控路径创建文件监控器
	for _, monitorPath := range m.config.MonitorPaths {
		if err := m.startWatcher(monitorPath); err != nil {
			log.Printf("启动文件监控失败 %s: %v", monitorPath.Path, err)
			continue
		}
	}

	log.Printf("Master节点启动完成，监听端口: %d", m.config.UDPPort)
	if m.webServer != nil {
		log.Printf("Web服务器已启动，端口: %d", m.webServer.GetPort())
	}
	return nil
}

// startWatcher 启动文件监控器
func (m *Master) startWatcher(monitorPath config.MonitorPath) error {
	// 检查路径是否存在
	if _, err := os.Stat(monitorPath.Path); os.IsNotExist(err) {
		return fmt.Errorf("监控路径不存在: %s", monitorPath.Path)
	}

	// 创建文件监控器（5秒防抖动）
	fw, err := watcher.NewFileWatcher(monitorPath.Path, 5000)
	if err != nil {
		return fmt.Errorf("创建文件监控器失败: %v", err)
	}

	// 启动监控
	fw.Start()

	// 保存监控器
	m.mutex.Lock()
	m.watchers[monitorPath.Path] = fw
	m.mutex.Unlock()

	log.Printf("启动文件监控: %s -> %v", monitorPath.Path, monitorPath.Slaves)

	// 处理文件事件
	go m.handleFileEvents(fw, monitorPath)

	return nil
}

// handleFileEvents 处理文件事件
func (m *Master) handleFileEvents(fw *watcher.FileWatcher, monitorPath config.MonitorPath) {
	for {
		select {
		case event, ok := <-fw.GetEventChan():
			if !ok {
				return
			}
			m.processFileEvent(event, monitorPath)

		case <-m.done:
			return
		}
	}
}

// processFileEvent 处理单个文件事件
func (m *Master) processFileEvent(event *watcher.FileEvent, monitorPath config.MonitorPath) {
	log.Printf("处理文件事件: %s %s", event.Op, event.Path)

	// 创建同步包
	syncPacket, err := watcher.CreateSyncPacket(event, monitorPath.Path)
	if err != nil {
		log.Printf("创建同步包失败: %v", err)
		return
	}

	// 发送到所有Slave节点
	var wg sync.WaitGroup
	for _, slaveAddr := range monitorPath.Slaves {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			m.sendToSlave(addr, syncPacket)
		}(slaveAddr)
	}

	// 等待所有发送完成（最多等待10秒）
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("文件事件处理完成: %s %s", event.Op, event.Path)
	case <-time.After(10 * time.Second):
		log.Printf("文件事件处理超时: %s %s", event.Op, event.Path)
	}
}

// sendToSlave 发送数据包到Slave节点
func (m *Master) sendToSlave(slaveAddr string, syncPacket *protocol.SyncPacket) {
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		if err := m.transport.Send(slaveAddr, syncPacket); err != nil {
			log.Printf("发送到Slave失败 %s (尝试 %d/%d): %v", slaveAddr, i+1, maxRetries, err)
			if i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * time.Second) // 指数退避
			}
			continue
		}
		log.Printf("成功发送到Slave: %s", slaveAddr)
		return
	}
	log.Printf("发送到Slave最终失败: %s", slaveAddr)
}

// handlePacket 处理接收到的数据包（如心跳等）
func (m *Master) handlePacket(packet *protocol.SyncPacket, remoteAddr string) error {
	log.Printf("Master收到数据包: %s %s from %s", packet.Op, packet.Path, remoteAddr)
	
	switch packet.Op {
	case "SYNC_REQUEST":
		return m.handleSyncRequest(remoteAddr)
	case "HEARTBEAT":
		// 心跳包，记录日志即可
		log.Printf("收到来自 %s 的心跳", remoteAddr)
		return nil
	default:
		// 其他类型的数据包
		return nil
	}
}

// handleSyncRequest 处理同步请求
func (m *Master) handleSyncRequest(slaveAddr string) error {
	log.Printf("处理来自 %s 的全量同步请求", slaveAddr)
	
	// 为每个监控路径发送所有文件
	for _, monitorPath := range m.config.MonitorPaths {
		// 检查这个Slave是否在监控路径的目标列表中
		if !m.isSlaveInPath(slaveAddr, monitorPath) {
			continue
		}
		
		log.Printf("开始向 %s 同步路径: %s", slaveAddr, monitorPath.Path)
		
		// 遍历目录并发送所有文件
		err := filepath.Walk(monitorPath.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			// 跳过目录
			if info.IsDir() {
				return nil
			}
			
			// 获取相对路径
			relPath, err := filepath.Rel(monitorPath.Path, path)
			if err != nil {
				return err
			}
			
			// 统一使用正斜杠
			relPath = strings.ReplaceAll(relPath, "\\", "/")
			
			// 读取文件内容
			content, err := ioutil.ReadFile(path)
			if err != nil {
				log.Printf("读取文件失败 %s: %v", path, err)
				return nil // 继续处理其他文件
			}
			
			// 创建同步包
			syncPacket := protocol.NewSyncPacket("CREATE", relPath, content)
			
			// 发送到Slave
			if err := m.transport.Send(slaveAddr, syncPacket); err != nil {
				log.Printf("发送文件到Slave失败 %s -> %s: %v", relPath, slaveAddr, err)
			} else {
				log.Printf("已发送文件到Slave: %s -> %s (%d bytes)", relPath, slaveAddr, len(content))
			}
			
			return nil
		})
		
		if err != nil {
			log.Printf("遍历目录失败 %s: %v", monitorPath.Path, err)
		}
	}
	
	log.Printf("完成向 %s 的全量同步", slaveAddr)
	return nil
}

// isSlaveInPath 检查Slave是否在监控路径的目标列表中
func (m *Master) isSlaveInPath(slaveAddr string, monitorPath config.MonitorPath) bool {
	for _, slave := range monitorPath.Slaves {
		if slave == slaveAddr {
			return true
		}
	}
	return false
}

// Stop 停止Master节点
func (m *Master) Stop() error {
	log.Printf("停止Master节点: %s", m.config.NodeID)

	// 发送停止信号
	close(m.done)

	// 停止所有文件监控器
	m.mutex.Lock()
	for path, fw := range m.watchers {
		fw.Stop()
		log.Printf("停止文件监控: %s", path)
	}
	m.mutex.Unlock()

	// 停止Web服务器（如果启用）
	if m.webServer != nil {
		if err := m.webServer.Stop(); err != nil {
			log.Printf("停止Web服务器失败: %v", err)
		} else {
			log.Printf("Web服务器已停止")
		}
	}

	// 关闭传输层
	if err := m.transport.Close(); err != nil {
		return fmt.Errorf("关闭传输层失败: %v", err)
	}

	log.Printf("Master节点已停止")
	return nil
}

// GetStats 获取统计信息
func (m *Master) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	stats := map[string]interface{}{
		"node_id":        m.config.NodeID,
		"role":           "master",
		"monitor_paths":  len(m.config.MonitorPaths),
		"active_watchers": len(m.watchers),
		"uptime":         time.Now().Format(time.RFC3339),
	}

	return stats
}

// SyncInitialFiles 同步初始文件（启动时）
func (m *Master) SyncInitialFiles() error {
	log.Printf("开始同步初始文件...")

	for _, monitorPath := range m.config.MonitorPaths {
		if err := m.syncDirectoryToSlaves(monitorPath); err != nil {
			log.Printf("同步目录失败 %s: %v", monitorPath.Path, err)
			continue
		}
	}

	log.Printf("初始文件同步完成")
	return nil
}

// syncDirectoryToSlaves 同步目录到所有Slave
func (m *Master) syncDirectoryToSlaves(monitorPath config.MonitorPath) error {
	return filepath.Walk(monitorPath.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 获取相对路径
		relPath, err := filepath.Rel(monitorPath.Path, path)
		if err != nil {
			return err
		}

		// 创建文件事件
		event := &watcher.FileEvent{
			Op:   "CREATE",
			Path: relPath,
		}

		// 处理文件事件
		m.processFileEvent(event, monitorPath)

		return nil
	})
}