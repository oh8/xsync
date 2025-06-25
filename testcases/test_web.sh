#!/bin/bash

# XSync Web服务器测试脚本

WEB_URL="http://localhost:8081"
USERNAME="admin"
PASSWORD="password"
TEST_FILE="test_upload.txt"

echo "=== XSync Web服务器功能测试 ==="

# 创建测试文件
echo "这是一个测试文件，用于验证XSync Web服务器的上传下载功能" > $TEST_FILE
echo "创建测试文件: $TEST_FILE"

# 测试健康检查
echo "\n1. 测试健康检查..."
curl -s "$WEB_URL/health" | jq . 2>/dev/null || curl -s "$WEB_URL/health"

# 测试文件上传
echo "\n2. 测试文件上传..."
RESPONSE=$(curl -s -u "$USERNAME:$PASSWORD" \
  -X POST \
  -F "file=@$TEST_FILE" \
  "$WEB_URL/upload")

echo "上传响应: $RESPONSE"

# 提取下载URL
DOWNLOAD_URL=$(echo $RESPONSE | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('download_url', ''))" 2>/dev/null)
if [ -z "$DOWNLOAD_URL" ]; then
    # 如果python3不可用，使用sed作为备选方案
    DOWNLOAD_URL=$(echo $RESPONSE | sed -n 's/.*"download_url":"\([^"]*\)".*/\1/p')
fi
echo "下载URL: $DOWNLOAD_URL"

# 测试文件下载
if [ ! -z "$DOWNLOAD_URL" ]; then
    echo "\n3. 测试文件下载..."
    curl -s "$DOWNLOAD_URL" > downloaded_file.txt
    
    echo "原文件内容:"
    cat $TEST_FILE
    echo "\n下载文件内容:"
    cat downloaded_file.txt
    
    # 比较文件内容
    if diff $TEST_FILE downloaded_file.txt > /dev/null; then
        echo "\n✅ 文件上传下载测试成功！"
    else
        echo "\n❌ 文件内容不匹配！"
    fi
    
    # 清理下载的文件
    rm -f downloaded_file.txt
else
    echo "\n❌ 未能获取下载URL"
fi

# 测试无认证访问
echo "\n4. 测试无认证访问..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$WEB_URL/upload")
if [ "$HTTP_CODE" = "401" ]; then
    echo "✅ 无认证访问被正确拒绝 (HTTP $HTTP_CODE)"
else
    echo "❌ 无认证访问未被拒绝 (HTTP $HTTP_CODE)"
fi

# 测试错误认证
echo "\n5. 测试错误认证..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -u "wrong:credentials" "$WEB_URL/upload")
if [ "$HTTP_CODE" = "401" ]; then
    echo "✅ 错误认证被正确拒绝 (HTTP $HTTP_CODE)"
else
    echo "❌ 错误认证未被拒绝 (HTTP $HTTP_CODE)"
fi

# 测试不存在的文件下载
echo "\n6. 测试下载不存在的文件..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -u "$USERNAME:$PASSWORD" "$WEB_URL/uploads/nonexistent-file.txt")
if [ "$HTTP_CODE" = "404" ]; then
    echo "✅ 不存在文件返回正确的404状态码"
else
    echo "❌ 不存在文件未返回404状态码 (HTTP $HTTP_CODE)"
fi

# 清理测试文件
rm -f $TEST_FILE

echo "\n=== 测试完成 ==="