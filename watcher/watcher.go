package watcher

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"xsync/protocol"
)

// FileEvent 文件事件
type FileEvent struct {
	Op   string // "CREATE", "MODIFY", "DELETE"
	Path string // 文件路径
}

// FileWatcher 文件监控器
type FileWatcher struct {
	watcher    *fsnotify.Watcher
	basePath   string
	eventChan  chan *FileEvent
	debouncer  map[string]*time.Timer
	mutex      sync.RWMutex
	done       chan bool
	debounceMs int
}

// NewFileWatcher 创建文件监控器
func NewFileWatcher(basePath string, debounceMs int) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("创建fsnotify监控器失败: %v", err)
	}

	fw := &FileWatcher{
		watcher:    watcher,
		basePath:   basePath,
		eventChan:  make(chan *FileEvent, 100),
		debouncer:  make(map[string]*time.Timer),
		done:       make(chan bool),
		debounceMs: debounceMs,
	}

	// 递归添加目录监控
	if err := fw.addRecursive(basePath); err != nil {
		return nil, fmt.Errorf("添加目录监控失败: %v", err)
	}

	return fw, nil
}

// addRecursive 递归添加目录监控
func (fw *FileWatcher) addRecursive(path string) error {
	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if err := fw.watcher.Add(walkPath); err != nil {
				return fmt.Errorf("添加目录监控失败 %s: %v", walkPath, err)
			}
			log.Printf("添加目录监控: %s", walkPath)
		}

		return nil
	})
}

// Start 启动文件监控
func (fw *FileWatcher) Start() {
	go fw.watchLoop()
}

// watchLoop 监控循环
func (fw *FileWatcher) watchLoop() {
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			fw.handleEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("文件监控错误: %v", err)

		case <-fw.done:
			return
		}
	}
}

// handleEvent 处理文件事件
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// 过滤临时文件和隐藏文件
	filename := filepath.Base(event.Name)
	if strings.HasPrefix(filename, ".") || strings.HasSuffix(filename, "~") {
		return
	}

	// 获取相对路径
	relPath, err := filepath.Rel(fw.basePath, event.Name)
	if err != nil {
		log.Printf("获取相对路径失败: %v", err)
		return
	}

	// 处理不同类型的事件
	var op string
	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		op = "CREATE"
		// 如果是新创建的目录，添加监控
		if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
			fw.watcher.Add(event.Name)
			log.Printf("添加新目录监控: %s", event.Name)
		}
	case event.Op&fsnotify.Write == fsnotify.Write:
		op = "MODIFY"
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		op = "DELETE"
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		op = "DELETE" // 重命名视为删除
	default:
		return // 忽略其他事件
	}

	// 防抖动处理
	fw.debounceEvent(op, relPath)
}

// debounceEvent 防抖动事件处理
func (fw *FileWatcher) debounceEvent(op, path string) {
	fw.mutex.Lock()
	defer fw.mutex.Unlock()

	key := fmt.Sprintf("%s:%s", op, path)

	// 取消之前的定时器
	if timer, exists := fw.debouncer[key]; exists {
		timer.Stop()
	}

	// 创建新的定时器
	fw.debouncer[key] = time.AfterFunc(time.Duration(fw.debounceMs)*time.Millisecond, func() {
		fw.mutex.Lock()
		delete(fw.debouncer, key)
		fw.mutex.Unlock()

		// 发送事件
		select {
		case fw.eventChan <- &FileEvent{Op: op, Path: path}:
		default:
			log.Printf("事件通道已满，丢弃事件: %s %s", op, path)
		}
	})
}

// GetEventChan 获取事件通道
func (fw *FileWatcher) GetEventChan() <-chan *FileEvent {
	return fw.eventChan
}

// Stop 停止文件监控
func (fw *FileWatcher) Stop() {
	close(fw.done)
	fw.watcher.Close()
	close(fw.eventChan)
}

// CreateSyncPacket 根据文件事件创建同步包
func CreateSyncPacket(event *FileEvent, basePath string) (*protocol.SyncPacket, error) {
	var content []byte
	var err error

	// 对于CREATE和MODIFY操作，读取文件内容
	if event.Op == "CREATE" || event.Op == "MODIFY" {
		fullPath := filepath.Join(basePath, event.Path)
		content, err = ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("读取文件失败 %s: %v", fullPath, err)
		}
	}

	return protocol.NewSyncPacket(event.Op, event.Path, content), nil
}