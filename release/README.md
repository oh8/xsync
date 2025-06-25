# XSync Release Package

## 文件说明

- `xsync` - XSync 主程序可执行文件
- `master.yaml` - Master 节点配置文件
- `slave.yaml` - Slave 节点配置文件样例

## 快速开始

### 1. 启动 Master 节点

```bash
# 前台运行
./xsync -c master.yaml

# 后台运行
./xsync -c master.yaml -d
```

### 2. 启动 Slave 节点

```bash
# 复制并修改 slave 配置
cp slave.yaml slave1.yaml
# 编辑 slave1.yaml 中的配置项

# 启动 slave 节点
./xsync -c slave1.yaml
```

### 3. Web 上传下载接口

Master 节点提供 HTTP API 接口用于文件上传和下载：

#### 健康检查
```bash
curl http://localhost:8081/health
```

#### 文件上传（需要认证）
```bash
# 使用 POST 方式上传
curl -u admin:password -X POST \
  -F "file=@example.txt" \
  http://localhost:8081/upload

# 使用 PUT 方式上传
curl -u admin:password -X PUT \
  -T "example.txt" \
  http://localhost:8081/upload
```

#### 文件下载（无需认证）
```bash
# 下载指定文件
curl -O http://localhost:8081/uploads/filename.txt

# 下载并重命名
curl http://localhost:8081/uploads/filename.txt -o local_file.txt
```

**API 接口说明：**
- `GET /health` - 服务健康检查
- `POST /upload` - 文件上传（multipart/form-data）
- `PUT /upload` - 文件上传（binary data）
- `GET /uploads/{filename}` - 文件下载

**认证信息：**
- 用户名：admin
- 密码：password

## 配置说明

### Master 节点配置

编辑 `master.yaml` 文件：
- `node_id`: 节点唯一标识
- `monitor_paths`: 监控的目录路径
- `web_server`: Web 服务配置

### Slave 节点配置

编辑 `slave.yaml` 文件：
- `node_id`: 节点唯一标识
- `master_addr`: Master 节点地址
- `sync_path`: 同步目录路径

## 注意事项

1. 确保所有节点使用相同的 AES 密钥
2. 防火墙需要开放相应的 UDP 端口
3. 建议在生产环境中修改默认的 Web 认证密码

更多信息请参考项目文档。