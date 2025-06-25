package webserver

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WebConfig Web服务配置
type WebConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Port      int    `yaml:"port"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	UploadDir string `yaml:"upload_dir"`
}

// WebServer Web服务器
type WebServer struct {
	config   *WebConfig
	server   *http.Server
	uploadDir string
}

// NewWebServer 创建Web服务器
func NewWebServer(cfg *WebConfig) (*WebServer, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("Web服务未启用")
	}

	// 设置默认值
	if cfg.Port == 0 {
		cfg.Port = 8081
	}
	if cfg.UploadDir == "" {
		cfg.UploadDir = "uploads"
	}

	// 创建上传目录
	uploadDir, err := filepath.Abs(cfg.UploadDir)
	if err != nil {
		return nil, fmt.Errorf("获取上传目录绝对路径失败: %v", err)
	}

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("创建上传目录失败: %v", err)
	}

	ws := &WebServer{
		config:    cfg,
		uploadDir: uploadDir,
	}

	// 设置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/uploads/", ws.handleDownload)  // 下载不需要认证
	mux.HandleFunc("/upload", ws.basicAuth(ws.handleUpload))  // 上传需要认证
	mux.HandleFunc("/health", ws.handleHealth)

	ws.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return ws, nil
}

// Start 启动Web服务器
func (ws *WebServer) Start() error {
	log.Printf("启动Web服务器，端口: %d，上传目录: %s", ws.config.Port, ws.uploadDir)
	
	// 创建一个channel来等待服务器启动结果
	started := make(chan error, 1)
	
	go func() {
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Web服务器启动失败: %v", err)
			started <- err
		} else {
			started <- nil
		}
	}()
	
	// 等待一小段时间确保服务器启动
	select {
	case err := <-started:
		if err != nil {
			return fmt.Errorf("Web服务器启动失败: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		// 100ms后假设启动成功
		log.Printf("Web服务器启动完成")
	}
	
	return nil
}

// Stop 停止Web服务器
func (ws *WebServer) Stop() error {
	if ws.server != nil {
		return ws.server.Close()
	}
	return nil
}

// basicAuth Basic认证中间件
func (ws *WebServer) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 如果没有配置用户名密码，跳过认证
		if ws.config.Username == "" || ws.config.Password == "" {
			next(w, r)
			return
		}

		username, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="XSync Web Server"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 使用constant time比较防止时序攻击
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(ws.config.Username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(ws.config.Password)) == 1

		if !usernameMatch || !passwordMatch {
			w.Header().Set("WWW-Authenticate", `Basic realm="XSync Web Server"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// handleUpload 处理文件上传
func (ws *WebServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 限制上传文件大小为100MB
	r.ParseMultipartForm(100 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "获取上传文件失败: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 生成16位随机前缀
	prefix, err := generateRandomPrefix()
	if err != nil {
		http.Error(w, "生成文件前缀失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 构造新文件名
	filename := fmt.Sprintf("%s-%s", prefix, header.Filename)
	filePath := filepath.Join(ws.uploadDir, filename)

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "创建文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 复制文件内容
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "保存文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 构造下载URL
	downloadURL := fmt.Sprintf("http://%s/uploads/%s", r.Host, filename)

	log.Printf("文件上传成功: %s -> %s", header.Filename, filename)

	// 返回下载地址
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success": true, "download_url": "%s", "filename": "%s"}`, downloadURL, filename)
}

// handleDownload 处理文件下载
func (ws *WebServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 提取文件名
	filename := strings.TrimPrefix(r.URL.Path, "/uploads/")
	if filename == "" {
		http.Error(w, "文件名不能为空", http.StatusBadRequest)
		return
	}

	// 防止路径遍历攻击
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		http.Error(w, "非法文件名", http.StatusBadRequest)
		return
	}

	filePath := filepath.Join(ws.uploadDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "文件不存在", http.StatusNotFound)
		return
	}

	// 提供文件下载
	http.ServeFile(w, r, filePath)
	log.Printf("文件下载: %s", filename)
}

// handleHealth 健康检查
func (ws *WebServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "ok", "service": "xsync-web", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
}

// generateRandomPrefix 生成16位随机前缀（小写字母和数字）
func generateRandomPrefix() (string, error) {
	bytes := make([]byte, 8) // 8字节 = 16个十六进制字符
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetUploadDir 获取上传目录
func (ws *WebServer) GetUploadDir() string {
	return ws.uploadDir
}

// GetPort 获取服务端口
func (ws *WebServer) GetPort() int {
	return ws.config.Port
}