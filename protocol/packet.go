package protocol

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
)

// SyncPacket 同步数据包结构
type SyncPacket struct {
	Op       string `json:"op"`       // "CREATE"/"MODIFY"/"DELETE"
	Path     string `json:"path"`     // 文件相对路径
	Content  []byte `json:"content"`  // 文件内容（DELETE时为空）
	Checksum uint32 `json:"checksum"` // CRC32校验
}

// NewSyncPacket 创建新的同步包
func NewSyncPacket(op, path string, content []byte) *SyncPacket {
	return &SyncPacket{
		Op:       op,
		Path:     path,
		Content:  content,
		Checksum: crc32.ChecksumIEEE(content),
	}
}

// Validate 验证数据包完整性
func (p *SyncPacket) Validate() error {
	if p.Op != "CREATE" && p.Op != "MODIFY" && p.Op != "DELETE" && p.Op != "SYNC_REQUEST" && p.Op != "SYNC_RESPONSE" && p.Op != "HEARTBEAT" {
		return fmt.Errorf("无效的操作类型: %s", p.Op)
	}

	if p.Path == "" {
		return fmt.Errorf("文件路径不能为空")
	}

	// 验证校验和
	if p.Op != "DELETE" {
		expectedChecksum := crc32.ChecksumIEEE(p.Content)
		if p.Checksum != expectedChecksum {
			return fmt.Errorf("校验和不匹配: 期望 %d, 实际 %d", expectedChecksum, p.Checksum)
		}
	}

	return nil
}

// Encrypt 加密数据包
func (p *SyncPacket) Encrypt(key []byte) ([]byte, error) {
	// 序列化数据包
	data, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("序列化数据包失败: %v", err)
	}

	// 创建AES-GCM加密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES加密器失败: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %v", err)
	}

	// 生成随机nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成nonce失败: %v", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// DecryptPacket 解密数据包
func DecryptPacket(encryptedData []byte, key []byte) (*SyncPacket, error) {
	// 创建AES-GCM解密器
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建AES解密器失败: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM模式失败: %v", err)
	}

	// 检查数据长度
	nonceSize := gcm.NonceSize()
	if len(encryptedData) < nonceSize {
		return nil, fmt.Errorf("加密数据太短")
	}

	// 提取nonce和密文
	nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}

	// 反序列化数据包
	var packet SyncPacket
	if err := json.Unmarshal(plaintext, &packet); err != nil {
		return nil, fmt.Errorf("反序列化数据包失败: %v", err)
	}

	// 验证数据包
	if err := packet.Validate(); err != nil {
		return nil, fmt.Errorf("数据包验证失败: %v", err)
	}

	return &packet, nil
}