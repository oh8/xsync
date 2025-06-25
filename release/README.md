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

### 3. Web 管理界面

访问 http://localhost:8081 使用 Web 管理功能：
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