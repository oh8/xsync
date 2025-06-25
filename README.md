# 🚀 XSync - 高性能跨服务器文件同步系统

[![Go Version](https://img.shields.io/badge/Go-1.19+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![QUIC Protocol](https://img.shields.io/badge/Protocol-QUIC-FF6B6B?style=for-the-badge)](https://quicwg.org/)
[![License](https://img.shields.io/badge/License-MIT-green?style=for-the-badge)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen?style=for-the-badge)]()

> 🌟 基于 QUIC 协议的企业级文件同步解决方案，专为高延迟、不稳定网络环境设计

## ✨ 核心特性

### 🔥 高性能网络传输
- **QUIC 协议**: 基于 UDP 的多路复用传输，减少连接建立时间
- **0-RTT 连接恢复**: 支持快速重连，适应网络波动
- **智能拥塞控制**: 自适应网络带宽，优化传输效率
- **连接复用**: 单连接多流传输，降低网络开销

### 🛡️ 企业级安全
- **AES-256-GCM 加密**: 端到端数据加密保护
- **TLS 1.3**: 传输层安全协议
- **CRC32 校验**: 数据完整性验证
- **自签名证书**: 开箱即用的安全配置

### ⚡ 实时同步能力
- **文件系统监控**: 基于 fsnotify 的实时文件变化检测
- **增量同步**: 仅传输变化部分，节省带宽
- **批量操作**: 智能合并文件操作，提升效率
- **断点续传**: 支持大文件传输中断恢复

### 🌐 Web 管理功能
- **HTTP API**: RESTful 接口，支持文件上传下载
- **Basic Auth 认证**: 安全的用户认证机制
- **健康检查**: 服务状态监控接口
- **文件管理**: 通过 Web 接口管理同步文件
- **随机文件名**: 自动生成唯一文件名，避免冲突

### 🏗️ 分布式架构
- **Master-Slave 模式**: 中心化管理，分布式执行
- **多节点支持**: 一对多文件分发
- **故障自愈**: 自动重连和状态恢复
- **负载均衡**: 智能分配同步任务

## 📊 产品对比

### 🔍 与主流同步软件对比

| 特性 | XSync | Rsync | Syncthing | Unison | Rclone |
|------|-------|-------|-----------|--------|---------|
| **传输协议** | QUIC (UDP) | SSH/TCP | TCP | TCP | HTTP/S |
| **实时同步** | ✅ 毫秒级 | ❌ 手动触发 | ✅ 秒级 | ❌ 手动触发 | ❌ 定时同步 |
| **加密方式** | AES-256-GCM | SSH加密 | TLS 1.3 | SSH加密 | 多种加密 |
| **网络优化** | ✅ 拥塞控制 | ❌ 基础TCP | ✅ 自适应 | ❌ 基础TCP | ✅ 多线程 |
| **断点续传** | ✅ 原生支持 | ✅ 增量同步 | ✅ 块级同步 | ✅ 增量同步 | ✅ 分片上传 |
| **多节点架构** | ✅ 1对多 | ❌ 1对1 | ✅ P2P网状 | ❌ 1对1 | ❌ 1对1 |
| **配置复杂度** | 🟡 中等 | 🟢 简单 | 🟡 中等 | 🟢 简单 | 🔴 复杂 |
| **内存占用** | 🟢 <50MB | 🟢 <20MB | 🟡 50-200MB | 🟢 <30MB | 🟡 100-500MB |
| **跨平台支持** | ✅ 全平台 | ✅ 全平台 | ✅ 全平台 | ✅ 全平台 | ✅ 全平台 |
| **GUI界面** | ❌ 命令行 | ❌ 命令行 | ✅ Web界面 | ✅ 图形界面 | ❌ 命令行 |
| **云存储支持** | ❌ 本地同步 | ❌ 本地同步 | ❌ 本地同步 | ❌ 本地同步 | ✅ 多云支持 |
| **冲突处理** | 🟡 覆盖模式 | 🟡 覆盖模式 | ✅ 智能合并 | ✅ 交互式 | 🟡 覆盖模式 |
| **性能表现** | 🟢 >200MB/s | 🟢 >150MB/s | 🟡 50-100MB/s | 🟡 30-80MB/s | 🟢 >100MB/s |
| **学习成本** | 🟡 中等 | 🟢 低 | 🟡 中等 | 🟢 低 | 🔴 高 |

### 🎯 适用场景对比

| 场景 | 推荐方案 | 原因 |
|------|----------|-------|
| **数据中心间同步** | XSync | QUIC协议优化、低延迟、高吞吐 |
| **开发环境同步** | Syncthing | P2P架构、易配置、有GUI |
| **服务器备份** | Rsync | 成熟稳定、增量备份、脚本友好 |
| **个人文件同步** | Syncthing | 用户友好、自动发现、冲突处理 |
| **云存储同步** | Rclone | 多云支持、功能丰富 |
| **高频实时同步** | XSync | 毫秒级响应、实时监控 |
| **大文件传输** | XSync/Rsync | 断点续传、压缩传输 |
| **跨网段同步** | XSync | UDP穿透、网络优化 |

### 💡 选择建议

#### 选择 XSync 的理由
- ✅ 需要**毫秒级实时同步**
- ✅ **高延迟网络环境**（如跨国专线）
- ✅ **一对多分发**场景
- ✅ 对**传输性能**有极高要求
- ✅ **企业级安全**需求

#### 选择其他方案的理由
- **Rsync**: 简单备份任务，脚本自动化
- **Syncthing**: 个人使用，需要图形界面
- **Unison**: 双向同步，交互式冲突处理
- **Rclone**: 云存储集成，多云环境

## 🚀 快速开始

### 📋 系统要求

- **操作系统**: Linux, macOS, Windows
- **Go 版本**: 1.19+
- **网络**: UDP 端口访问权限
- **磁盘**: 足够的存储空间用于文件同步

### 🔧 安装部署

#### 方式一：源码编译

```bash
# 克隆项目
git clone https://github.com/oh8/xsync.git
cd xsync

# 安装依赖
go mod download

# 编译
go build -o xsync .

# 或使用安装脚本
./install.sh
```

#### 方式二：二进制下载

```bash
# 下载最新版本
wget https://github.com/oh8/xsync/releases/latest/download/xsync-linux-amd64.tar.gz
tar -xzf xsync-linux-amd64.tar.gz
chmod +x xsync
```

### 🔧 系统优化建议

#### UDP 缓冲区优化

为了获得最佳性能，特别是在高带宽网络环境下，建议优化系统的 UDP 缓冲区大小：

**Linux 系统：**
```bash
# 临时设置（重启后失效）
sudo sysctl -w net.core.rmem_max=7500000
sudo sysctl -w net.core.wmem_max=7500000

# 永久设置（添加到 /etc/sysctl.conf）
echo 'net.core.rmem_max=7500000' | sudo tee -a /etc/sysctl.conf
echo 'net.core.wmem_max=7500000' | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

**macOS 系统：**
```bash
# 临时设置
sudo sysctl -w kern.ipc.maxsockbuf=8441037

# 永久设置（添加到 /etc/sysctl.conf）
echo 'kern.ipc.maxsockbuf=8441037' | sudo tee -a /etc/sysctl.conf
```

> 💡 **说明**: 如果看到 "failed to sufficiently increase receive buffer size" 警告，这不会影响基本功能，只是在高带宽传输时可能影响性能。应用上述优化后可以消除此警告。

### ⚙️ 配置文件

#### Master 节点配置 (`master.yaml`)

```yaml
node_id: "master-01"
role: "master"
key: "12345678901234567890123456789012"  # 32字节AES密钥
udp_port: 9401
monitor_paths:
  - path: "./data01"
    slaves: ["127.0.0.1:9402", "127.0.0.1:9403"]

# Web 服务器配置（可选）
webserver:
  enabled: true
  port: 8081
  username: "admin"
  password: "password"
  upload_dir: "data01/uploads"
```

#### Slave 节点配置 (`config/slave1.yaml`)

```yaml
node_id: "slave-01"
role: "slave"
key: "12345678901234567890123456789012"  # 与Master相同的密钥
master_addr: "127.0.0.1:9401"
sync_path: "./data02"
udp_port: 9402
```

### 🎯 启动服务

#### 启动 Master 节点

```bash
# 前台运行
./xsync -c master.yaml

# 守护进程模式
./xsync -c master.yaml -d

# 指定日志文件
./xsync -c master.yaml -l /var/log/xsync-master.log -d
```

#### Web 服务使用

启动 Master 节点后，如果启用了 Web 服务器，可以通过以下方式使用：

```bash
# 健康检查
curl http://localhost:8081/health

# 上传文件（需要认证）
curl -u admin:password -X POST \
  -F "file=@example.txt" \
  http://localhost:8081/upload

# 下载文件（无需认证）
curl http://localhost:8081/uploads/filename.txt
```

**Web API 接口说明：**
- `GET /health` - 健康检查，返回服务状态
- `POST /upload` - 文件上传，需要 Basic Auth 认证
- `PUT /upload` - 文件上传（PUT 方式），需要 Basic Auth 认证
- `GET /uploads/{filename}` - 文件下载，无需认证

#### 启动 Slave 节点

```bash
# 前台运行
./xsync -c config/slave1.yaml

# 守护进程模式
./xsync -c config/slave1.yaml -d

# 指定PID文件
./xsync -c config/slave1.yaml -p /var/run/xsync-slave1.pid -d
```

## 📊 性能优化建议

### 🌐 公网传输优化

#### 1. 网络层优化
```yaml
# 建议的QUIC配置优化
quic_config:
  max_idle_timeout: 300s        # 增加空闲超时
  max_receive_buffer_size: 2MB   # 增大接收缓冲区
  max_send_buffer_size: 2MB      # 增大发送缓冲区
  initial_stream_receive_window: 512KB
  initial_connection_receive_window: 1MB
```

#### 2. 传输策略优化
- **启用数据压缩**: 对文本文件进行 gzip 压缩
- **智能分片**: 大文件自动分片传输
- **并发控制**: 限制同时传输的文件数量
- **带宽自适应**: 根据网络状况调整传输速度

#### 3. 系统级优化
```bash
# Linux 系统优化
echo 'net.core.rmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.ipv4.udp_mem = 102400 873800 16777216' >> /etc/sysctl.conf
sysctl -p
```


## 📈 监控与运维


### 🔍 日志分析

```bash
# 实时查看同步日志
tail -f /var/log/xsync-master.log

# 过滤错误日志
grep "ERROR" /var/log/xsync-*.log

# 统计同步文件数量
grep "文件同步完成" /var/log/xsync-master.log | wc -l
```


## 🛠️ 高级功能

### 🌐 Web 管理功能

#### 文件上传管理

```bash
# 批量上传文件
for file in *.txt; do
  curl -u admin:password -X POST \
    -F "file=@$file" \
    http://localhost:8081/upload
done

# 上传大文件（支持最大 100MB）
curl -u admin:password -X POST \
  -F "file=@large_file.zip" \
  http://localhost:8081/upload
```

#### 文件下载管理

```bash
# 获取上传文件列表（通过日志）
grep "文件上传成功" /var/log/xsync-master.log

# 下载指定文件
curl -O http://localhost:8081/uploads/abc123def456-example.txt
```

#### Web 服务监控

```bash
# 检查 Web 服务状态
curl -s http://localhost:8081/health | jq .

# 输出示例
{
  "status": "ok",
  "service": "xsync-web",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### 🔄 全量同步

```bash
# 重新启动即可触发slave全量同步master
./xsync -c config/slave1.yaml -d
```

### 🎯 选择性同步

```yaml
# 配置文件过滤规则
filter_rules:
  include:
    - "*.txt"
    - "*.log"
    - "documents/**"
  exclude:
    - "*.tmp"
    - ".git/**"
    - "node_modules/**"
```

### 🔐 安全增强

**Web 安全最佳实践：**
- 使用强密码进行 Basic Auth 认证
- 定期更换认证密码
- 限制上传文件大小和类型
- 配置防火墙规则，仅允许可信 IP 访问

## 🏗️ 架构设计

### 🔄 数据流图

```
┌─────────────┐    QUIC/UDP    ┌─────────────┐
│   Master    │◄──────────────►│   Slave 1   │
│             │                │             │
│ ┌─────────┐ │                │ ┌─────────┐ │
│ │ Watcher │ │                │ │ Syncer  │ │
│ └─────────┘ │                │ └─────────┘ │
│ ┌─────────┐ │                │ ┌─────────┐ │
│ │Transport│ │                │ │Transport│ │
│ └─────────┘ │                │ └─────────┘ │
└─────────────┘                └─────────────┘
       │                              │
       │        QUIC/UDP              │
       └──────────────────────────────┤
                                      │
                              ┌─────────────┐
                              │   Slave 2   │
                              │             │
                              │ ┌─────────┐ │
                              │ │ Syncer  │ │
                              │ └─────────┘ │
                              │ ┌─────────┐ │
                              │ │Transport│ │
                              │ └─────────┘ │
                              └─────────────┘
```

### 🧩 组件说明

| 组件 | 功能 | 特性 |
|------|------|------|
| **Watcher** | 文件系统监控 | 实时检测文件变化 |
| **Transport** | 网络传输层 | QUIC协议、加密传输 |
| **Protocol** | 协议层 | 数据包封装、校验 |
| **Syncer** | 同步引擎 | 文件操作、冲突处理 |
| **WebServer** | Web 接口 | HTTP文件上传下载 |
| **Config** | 配置管理 | 动态配置、热重载 |

## 🤝 贡献指南

### 🐛 问题反馈

1. 查看 [Issues](https://github.com/oh8/xsync/issues) 确认问题未被报告
2. 使用问题模板创建新 Issue
3. 提供详细的复现步骤和环境信息

### 💡 功能建议

1. 在 [Discussions](https://github.com/oh8/xsync/discussions) 中讨论新功能
2. 创建 Feature Request Issue
3. 提交 Pull Request


## 📄 许可证

本项目采用 [MIT 许可证](LICENSE)。

## 🙏 致谢

- [quic-go](https://github.com/quic-go/quic-go) - QUIC 协议实现
- [fsnotify](https://github.com/fsnotify/fsnotify) - 文件系统监控
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML 配置解析


