#!/bin/bash

# xsync 系统服务安装脚本
# 用于将xsync安装为系统服务

set -e

# 检查是否以root权限运行
if [ "$(id -u)" -ne 0 ]; then
    echo "错误: 请使用root权限运行此脚本"
    exit 1
fi

# 配置变量
APP_NAME="xsync"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/xsync"
DATA_DIR="/var/lib/xsync"
LOG_DIR="/var/log/xsync"
SYSTEMD_DIR="/etc/systemd/system"

echo "=== 开始安装 $APP_NAME ==="

# 检查二进制文件是否存在
if [ ! -f "$APP_NAME" ]; then
    echo "错误: $APP_NAME 二进制文件不存在，请先编译"
    exit 1
fi

# 创建目录
echo "创建目录..."
mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"{/master,/slave1,/slave2}
mkdir -p "$LOG_DIR"

# 复制二进制文件
echo "安装二进制文件..."
cp "$APP_NAME" "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/$APP_NAME"

# 复制配置文件模板
echo "安装配置文件..."
if [ -f "xsync.yaml.example" ]; then
    cp "xsync.yaml.example" "$CONFIG_DIR/"
fi

# 检查配置文件是否存在
if [ -d "config" ]; then
    if [ -f "master.yaml" ]; then
    cp "master.yaml" "$CONFIG_DIR/"
    fi
    if [ -f "slave1.yaml" ]; then
        cp "slave1.yaml" "$CONFIG_DIR/"
    fi
    if [ -f "slave2.yaml" ]; then
        cp "slave2.yaml" "$CONFIG_DIR/"
    fi
fi

# 创建Master服务单元
echo "创建Master服务单元..."
cat > "$SYSTEMD_DIR/$APP_NAME-master.service" << EOF
[Unit]
Description=xsync Master Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=$INSTALL_DIR/$APP_NAME -c $CONFIG_DIR/master.yaml
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536
WorkingDirectory=$DATA_DIR/master
StandardOutput=append:$LOG_DIR/master.log
StandardError=append:$LOG_DIR/master.log

[Install]
WantedBy=multi-user.target
EOF

# 创建Slave1服务单元
echo "创建Slave1服务单元..."
cat > "$SYSTEMD_DIR/$APP_NAME-slave1.service" << EOF
[Unit]
Description=xsync Slave1 Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=$INSTALL_DIR/$APP_NAME -c $CONFIG_DIR/slave1.yaml
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536
WorkingDirectory=$DATA_DIR/slave1
StandardOutput=append:$LOG_DIR/slave1.log
StandardError=append:$LOG_DIR/slave1.log

[Install]
WantedBy=multi-user.target
EOF

# 创建Slave2服务单元
echo "创建Slave2服务单元..."
cat > "$SYSTEMD_DIR/$APP_NAME-slave2.service" << EOF
[Unit]
Description=xsync Slave2 Service
After=network.target

[Service]
Type=simple
User=root
ExecStart=$INSTALL_DIR/$APP_NAME -c $CONFIG_DIR/slave2.yaml
Restart=on-failure
RestartSec=5s
LimitNOFILE=65536
WorkingDirectory=$DATA_DIR/slave2
StandardOutput=append:$LOG_DIR/slave2.log
StandardError=append:$LOG_DIR/slave2.log

[Install]
WantedBy=multi-user.target
EOF

# 重新加载systemd配置
echo "重新加载systemd配置..."
systemctl daemon-reload

echo "=== $APP_NAME 安装完成 ==="
echo ""
echo "使用以下命令启动服务:"
echo "  systemctl start $APP_NAME-master"
echo "  systemctl start $APP_NAME-slave1"
echo "  systemctl start $APP_NAME-slave2"
echo ""
echo "使用以下命令设置开机自启:"
echo "  systemctl enable $APP_NAME-master"
echo "  systemctl enable $APP_NAME-slave1"
echo "  systemctl enable $APP_NAME-slave2"
echo ""
echo "日志文件位置: $LOG_DIR"
echo "配置文件位置: $CONFIG_DIR"
echo ""
echo "注意: 请确保在启动服务前正确配置密钥和网络设置"