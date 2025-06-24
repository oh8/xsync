# xsync - 跨服务器文件同步守护程序

[![Go Version](https://img.shields.io/badge/Go-%3E%3D1.18-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 概述

xsync是一个高性能的跨服务器文件同步工具，采用主从架构设计，支持一主多从的实时文件同步。基于QUIC协议实现可靠的UDP传输，并使用AES-GCM加密保证数据安全。

### 核心特性

- 🚀 **实时同步**: 基于fsnotify的文件系统事件监控，毫秒级响应
- 🔒 **安全传输**: AES-256-GCM加密，保证数据传输安全
- 🌐 **可靠网络**: 基于QUIC协议，内置丢包重传和拥塞控制
- ⚡ **高性能**: 支持大文件传输，目标传输速度≥200MB/s
- 🎯 **低延迟**: 事件处理延迟≤1s（内网环境）
- 💾 **低内存**: 单节点内存占用≤50MB

## 快速开始

### 1. 编译安装

```bash
# 克隆项目
git clone https://github.com/your-org/xsync.git
cd xsync

# 下载依赖
go mod download

# 编译
go build -o xsync .
```

### 2. 配置文件

复制配置模板并修改：

```bash
# 创建配置目录
mkdir -p config

# 复制并编辑Master配置
cp xsync.yaml.example config/master.yaml

# 复制并编辑Slave配置
cp xsync.yaml.example config/slave1.yaml
cp xsync.yaml.example config/slave2.yaml
```

### 3. 设置加密密钥

**推荐方式**：使用环境变量（更安全）

```bash
# 生成32字节随机密钥
export XSYNC_KEY="$(openssl rand -hex 16)"
echo "Generated key: $XSYNC_KEY"
```

**备选方式**：在配置文件中设置（不推荐生产环境）

```yaml
key: "your-32-byte-aes-key-here-change-me"
```

### 4. 创建测试目录

```bash
# 创建同步目录
mkdir -p data01 data02 data03
```

### 5. 启动节点

**终端1 - 启动Master节点：**
```bash
./xsync -c config/master.yaml
```

**终端2 - 启动Slave1节点：**
```bash
./xsync -c config/slave1.yaml
```

**终端3 - 启动Slave2节点：**
```bash
./xsync -c config/slave2.yaml
```

### 6. 测试同步

```bash
# 在Master目录创建文件
echo "Hello xsync!" > data01/test.txt

# 检查Slave目录是否同步
ls -la data02/ data03/
cat data02/test.txt data03/test.txt

# 修改文件内容
echo "Modified content" >> data01/test.txt

# 删除文件测试
rm data01/test.txt
```

## 配置说明

### Master节点配置

```yaml
node_id: "master"          # 节点标识
role: "master"             # 节点角色
key: "32-byte-aes-key"     # AES-256密钥
udp_port: 9401             # 监听端口
monitor_paths:             # 监控路径列表
  - path: "./data01"       # 监控目录
    slaves:                # 目标Slave节点
      - "127.0.0.1:9402"
      - "127.0.0.1:9403"
```

### Slave节点配置

```yaml
node_id: "slave1"          # 节点标识
role: "slave"              # 节点角色
key: "32-byte-aes-key"     # AES-256密钥（与Master相同）
master_addr: "127.0.0.1:9401"  # Master节点地址
sync_path: "./data02"      # 本地同步目录
udp_port: 9402             # 监听端口
```

## 部署指南

### 生产环境部署

1. **网络配置**
   - 确保Master和Slave节点之间网络互通
   - 开放配置的UDP端口（防火墙/安全组）
   - 建议使用内网环境以获得最佳性能

2. **安全配置**
   - 使用强随机密钥：`openssl rand -hex 16`
   - 通过环境变量传递密钥，避免配置文件泄露
   - 定期轮换加密密钥

3. **性能优化**
   - 根据网络带宽调整并发连接数
   - 监控内存使用情况，必要时调整缓冲区大小
   - 使用SSD存储以提高I/O性能

### 系统服务安装

使用提供的安装脚本：

```bash
# 安装为系统服务
sudo ./install.sh

# 启动服务
sudo systemctl start xsync-master
sudo systemctl start xsync-slave1

# 设置开机自启
sudo systemctl enable xsync-master
sudo systemctl enable xsync-slave1
```

## 监控和运维

### 日志监控

```bash
# 查看实时日志
tail -f /var/log/xsync/master.log
tail -f /var/log/xsync/slave1.log

# 查看错误日志
grep "ERROR" /var/log/xsync/*.log
```

### 性能监控

程序每60秒输出一次统计信息：

```
节点状态: map[applied_files:42 errors:0 last_sync:2024-01-15T10:30:45Z node_id:slave1 received_packets:42 role:slave sync_path:./data02]
```

### 健康检查

```bash
# 检查进程状态
ps aux | grep xsync

# 检查端口监听
netstat -ulnp | grep 940[1-3]

# 检查文件同步状态
find data01 -type f | wc -l
find data02 -type f | wc -l
find data03 -type f | wc -l
```

## 故障处理

### 常见问题

#### 1. 连接失败

**症状**：Slave无法连接到Master

**排查步骤**：
```bash
# 检查网络连通性
ping <master_ip>
telnet <master_ip> <master_port>

# 检查防火墙
sudo ufw status
sudo iptables -L

# 检查端口占用
sudo netstat -ulnp | grep <port>
```

**解决方案**：
- 确保防火墙开放相应端口
- 检查网络路由配置
- 验证配置文件中的地址和端口

#### 2. 加密失败

**症状**：日志显示"解密失败"错误

**解决方案**：
- 确保所有节点使用相同的32字节密钥
- 检查密钥是否包含特殊字符
- 重新生成并分发密钥

#### 3. 文件同步延迟

**症状**：文件变更后很久才同步

**排查步骤**：
```bash
# 检查文件系统事件
inotifywait -m -r ./data01

# 检查网络延迟
ping -c 10 <slave_ip>

# 检查系统负载
top
iostat 1
```

**解决方案**：
- 检查磁盘I/O性能
- 优化网络配置
- 调整防抖动时间

#### 4. 内存占用过高

**症状**：进程内存使用超过50MB

**解决方案**：
- 检查是否有大文件传输
- 调整缓冲区大小
- 重启服务释放内存

### 紧急恢复

#### 数据不一致处理

```bash
# 停止所有节点
sudo systemctl stop xsync-*

# 备份当前数据
cp -r data02 data02.backup
cp -r data03 data03.backup

# 从Master重新同步
rm -rf data02/* data03/*

# 重启服务
sudo systemctl start xsync-master
sleep 5
sudo systemctl start xsync-slave1
sudo systemctl start xsync-slave2
```

#### 配置回滚

```bash
# 恢复配置文件
cp config/master.yaml.backup config/master.yaml
cp config/slave1.yaml.backup config/slave1.yaml

# 重启服务
sudo systemctl restart xsync-*
```

## 性能调优

### 网络优化

```bash
# 调整UDP缓冲区大小
echo 'net.core.rmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 134217728' >> /etc/sysctl.conf
sudo sysctl -p
```

### 文件系统优化

```bash
# 增加inotify监控限制
echo 'fs.inotify.max_user_watches = 524288' >> /etc/sysctl.conf
echo 'fs.inotify.max_user_instances = 512' >> /etc/sysctl.conf
sudo sysctl -p
```

## 开发和贡献

### 项目结构

```
xsync/
├── config/          # 配置管理
├── protocol/        # 同步协议
├── transport/       # 网络传输层
├── watcher/         # 文件监控
├── master/          # Master节点实现
├── slave/           # Slave节点实现
└── main.go          # 主程序入口
```

### 运行测试

```bash
# 运行单元测试
go test ./...

# 运行基准测试
go test -bench=. ./...

# 生成测试覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 压力测试

```bash
# 使用fio生成高并发文件操作
sudo apt install fio
fio --name=random-write --ioengine=libaio --rw=randwrite --bs=4k --size=100M --numjobs=4 --directory=./data01
```

## 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 支持

- 📧 邮件支持：support@example.com
- 🐛 问题反馈：[GitHub Issues](https://github.com/your-org/xsync/issues)
- 📖 文档：[Wiki](https://github.com/your-org/xsync/wiki)