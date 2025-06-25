# XSync 测试用例

本目录包含 XSync 项目的测试用例和相关配置文件。

## 目录结构

```
testcases/
├── README.md          # 本说明文件
├── configs/           # 测试配置文件
│   ├── slave1.yaml    # Slave 节点1配置
│   └── slave2.yaml    # Slave 节点2配置
├── data01/            # 测试数据目录1
├── data02/            # 测试数据目录2
├── data03/            # 测试数据目录3
└── test_web.sh        # Web 接口测试脚本
```

## 使用说明

### 1. 测试数据目录

- `data01/`, `data02/`, `data03/`: 包含用于测试文件同步功能的示例数据
- 这些目录可以作为 Master 节点的监控目录，用于测试文件变更检测和同步功能

### 2. 测试配置文件

- `configs/slave1.yaml`: Slave 节点1的配置文件示例
- `configs/slave2.yaml`: Slave 节点2的配置文件示例
- 可以根据实际测试需求修改这些配置文件

### 3. 测试脚本

- `test_web.sh`: Web 管理界面的功能测试脚本
- 包含对 Web API 接口的基本测试用例

## 快速测试

1. 启动 Master 节点（在项目根目录）：
   ```bash
   ./xsync -c master.yaml
   ```

2. 启动 Slave 节点：
   ```bash
   ./xsync -c testcases/configs/slave1.yaml
   ./xsync -c testcases/configs/slave2.yaml
   ```

3. 运行 Web 接口测试：
   ```bash
   cd testcases
   ./test_web.sh
   ```

4. 测试文件同步：
   - 在 `testcases/data01/` 目录中添加、修改或删除文件
   - 观察 Slave 节点是否正确同步这些变更

## 注意事项

- 确保在运行测试前已正确配置各节点的网络地址和端口
- 测试数据目录的路径需要在配置文件中正确设置
- 建议在隔离的测试环境中运行，避免影响生产数据