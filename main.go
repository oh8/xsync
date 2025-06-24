package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"xsync/config"
	"xsync/master"
	"xsync/slave"
)

var (
	configPath = flag.String("c", "xsync.yaml", "配置文件路径")
	version    = flag.Bool("v", false, "显示版本信息")
	help       = flag.Bool("h", false, "显示帮助信息")
	daemon     = flag.Bool("d", false, "以daemon模式运行")
	pidFile    = flag.String("p", "", "PID文件路径")
	logFile    = flag.String("l", "", "日志文件路径")
)

const (
	VERSION = "1.0.0"
	APP_NAME = "xsync"
)

func main() {
	flag.Parse()

	if *version {
		fmt.Printf("%s version %s\n", APP_NAME, VERSION)
		return
	}

	if *help {
		printUsage()
		return
	}

	// 如果是daemon模式，进行进程分离
	if *daemon {
		if err := daemonize(); err != nil {
			log.Fatalf("daemon化失败: %v", err)
		}
		return
	}

	// 设置日志
	setupLogging()

	// 创建PID文件
	if *pidFile != "" {
		if err := createPidFile(*pidFile); err != nil {
			log.Fatalf("创建PID文件失败: %v", err)
		}
		defer removePidFile(*pidFile)
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	log.Printf("启动 %s 节点: %s", cfg.Role, cfg.NodeID)

	// 根据角色启动相应的节点
	var node Node
	if cfg.IsMaster() {
		node, err = startMaster(cfg)
	} else {
		node, err = startSlave(cfg)
	}

	if err != nil {
		log.Fatalf("启动节点失败: %v", err)
	}

	// 等待中断信号
	waitForShutdown(node)
}

// Node 节点接口
type Node interface {
	Stop() error
	GetStats() map[string]interface{}
}

// startMaster 启动Master节点
func startMaster(cfg *config.Config) (Node, error) {
	m, err := master.NewMaster(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建Master节点失败: %v", err)
	}

	if err := m.Start(); err != nil {
		return nil, fmt.Errorf("启动Master节点失败: %v", err)
	}

	// 延迟同步初始文件（给Slave节点启动时间）
	go func() {
		time.Sleep(3 * time.Second)
		if err := m.SyncInitialFiles(); err != nil {
			log.Printf("同步初始文件失败: %v", err)
		}
	}()

	return m, nil
}

// startSlave 启动Slave节点
func startSlave(cfg *config.Config) (Node, error) {
	s, err := slave.NewSlave(cfg)
	if err != nil {
		return nil, fmt.Errorf("创建Slave节点失败: %v", err)
	}

	if err := s.Start(); err != nil {
		return nil, fmt.Errorf("启动Slave节点失败: %v", err)
	}

	// 启动心跳（每30秒）
	s.StartHeartbeat(30 * time.Second)

	return s, nil
}

// waitForShutdown 等待关闭信号
func waitForShutdown(node Node) {
	// 创建信号通道
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动状态报告定时器
	statsTicker := time.NewTicker(60 * time.Second)
	defer statsTicker.Stop()

	log.Printf("节点运行中，按 Ctrl+C 停止...")

	for {
		select {
		case sig := <-sigChan:
			log.Printf("接收到信号: %v，开始关闭...", sig)
			if err := node.Stop(); err != nil {
				log.Printf("关闭节点失败: %v", err)
				os.Exit(1)
			}
			log.Printf("节点已安全关闭")
			os.Exit(0)

		case <-statsTicker.C:
			stats := node.GetStats()
			log.Printf("节点状态: %+v", stats)
		}
	}
}

// printUsage 打印使用说明
func printUsage() {
	fmt.Printf(`%s - 跨服务器文件同步守护程序 v%s

`, APP_NAME, VERSION)
	fmt.Printf("用法:\n")
	fmt.Printf("  %s [选项]\n\n", APP_NAME)
	fmt.Printf("选项:\n")
	fmt.Printf("  -c <配置文件>    指定配置文件路径 (默认: xsync.yaml)\n")
	fmt.Printf("  -d              以daemon模式运行\n")
	fmt.Printf("  -p <PID文件>     指定PID文件路径\n")
	fmt.Printf("  -l <日志文件>    指定日志文件路径\n")
	fmt.Printf("  -v              显示版本信息\n")
	fmt.Printf("  -h              显示此帮助信息\n\n")
	fmt.Printf("示例:\n")
	fmt.Printf("  # 前台启动Master节点\n")
	fmt.Printf("  %s -c config/master.yaml\n\n", APP_NAME)
	fmt.Printf("  # daemon模式启动Master节点\n")
	fmt.Printf("  %s -d -c config/master.yaml -p /var/run/xsync-master.pid -l /var/log/xsync-master.log\n\n", APP_NAME)
	fmt.Printf("  # daemon模式启动Slave节点\n")
	fmt.Printf("  %s -d -c config/slave1.yaml -p /var/run/xsync-slave1.pid -l /var/log/xsync-slave1.log\n\n", APP_NAME)
	fmt.Printf("环境变量:\n")
	fmt.Printf("  XSYNC_KEY       AES-256加密密钥 (32字节)\n\n")
	fmt.Printf("信号处理:\n")
	fmt.Printf("  SIGTERM/SIGINT  优雅停止服务\n")
	fmt.Printf("  SIGHUP          重新加载配置\n")
	fmt.Printf("  SIGUSR1         输出状态信息\n\n")
	fmt.Printf("更多信息请参考: https://github.com/your-org/xsync\n")
}

// daemonize 进程daemon化
func daemonize() error {
	// 获取当前可执行文件路径
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("获取可执行文件路径失败: %v", err)
	}

	// 构建子进程参数（去掉-d参数）
	args := make([]string, 0, len(os.Args)-1)
	for i, arg := range os.Args {
		if i == 0 {
			continue // 跳过程序名
		}
		if arg == "-d" || arg == "--daemon" {
			continue // 跳过daemon参数
		}
		args = append(args, arg)
	}

	// 启动子进程
	cmd := exec.Command(execPath, args...)
	cmd.Env = os.Environ()

	// 重定向标准输入输出
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// 启动子进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动daemon进程失败: %v", err)
	}

	fmt.Printf("daemon进程已启动，PID: %d\n", cmd.Process.Pid)
	return nil
}

// setupLogging 设置日志
func setupLogging() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("[xsync] ")

	// 如果指定了日志文件，重定向日志输出
	if *logFile != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(*logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("创建日志目录失败: %v", err)
			return
		}

		// 打开日志文件
		file, err := os.OpenFile(*logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("打开日志文件失败: %v", err)
			return
		}

		// 重定向日志输出
		log.SetOutput(file)

		// 同时重定向标准输出和错误输出
		os.Stdout = file
		os.Stderr = file
	}
}

// createPidFile 创建PID文件
func createPidFile(pidPath string) error {
	// 检查PID文件是否已存在
	if _, err := os.Stat(pidPath); err == nil {
		// 读取现有PID
		data, err := os.ReadFile(pidPath)
		if err == nil {
			if pid, err := strconv.Atoi(string(data)); err == nil {
				// 检查进程是否还在运行
				if process, err := os.FindProcess(pid); err == nil {
					if err := process.Signal(syscall.Signal(0)); err == nil {
						return fmt.Errorf("进程已在运行，PID: %d", pid)
					}
				}
			}
		}
	}

	// 确保PID文件目录存在
	pidDir := filepath.Dir(pidPath)
	if err := os.MkdirAll(pidDir, 0755); err != nil {
		return fmt.Errorf("创建PID目录失败: %v", err)
	}

	// 创建PID文件
	file, err := os.Create(pidPath)
	if err != nil {
		return fmt.Errorf("创建PID文件失败: %v", err)
	}
	defer file.Close()

	// 写入当前进程PID
	if _, err := file.WriteString(strconv.Itoa(os.Getpid())); err != nil {
		return fmt.Errorf("写入PID失败: %v", err)
	}

	return nil
}

// removePidFile 删除PID文件
func removePidFile(pidPath string) {
	if err := os.Remove(pidPath); err != nil {
		log.Printf("删除PID文件失败: %v", err)
	}
}