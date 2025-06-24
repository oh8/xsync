package transport

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/quic-go/quic-go"
	"xsync/protocol"
)

// Transport 传输层接口
type Transport interface {
	Send(addr string, packet *protocol.SyncPacket) error
	Listen(port int, handler PacketHandler) error
	Close() error
}

// PacketHandler 数据包处理函数
type PacketHandler func(packet *protocol.SyncPacket, remoteAddr string) error

// QUICTransport QUIC传输实现
type QUICTransport struct {
	key       []byte
	listener  *quic.Listener
	conns     map[string]quic.Connection
	connMutex sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewQUICTransport 创建QUIC传输器
func NewQUICTransport(key []byte) *QUICTransport {
	ctx, cancel := context.WithCancel(context.Background())
	return &QUICTransport{
		key:    key,
		conns:  make(map[string]quic.Connection),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Send 发送数据包
func (qt *QUICTransport) Send(addr string, packet *protocol.SyncPacket) error {
	// 加密数据包
	encryptedData, err := packet.Encrypt(qt.key)
	if err != nil {
		return fmt.Errorf("加密数据包失败: %v", err)
	}

	// 获取或创建连接
	conn, err := qt.getConnection(addr)
	if err != nil {
		return fmt.Errorf("获取连接失败: %v", err)
	}

	// 打开流
	stream, err := conn.OpenStreamSync(qt.ctx)
	if err != nil {
		return fmt.Errorf("打开流失败: %v", err)
	}
	defer stream.Close()

	// 发送数据长度
	lengthBytes := make([]byte, 4)
	lengthBytes[0] = byte(len(encryptedData) >> 24)
	lengthBytes[1] = byte(len(encryptedData) >> 16)
	lengthBytes[2] = byte(len(encryptedData) >> 8)
	lengthBytes[3] = byte(len(encryptedData))

	if _, err := stream.Write(lengthBytes); err != nil {
		return fmt.Errorf("发送数据长度失败: %v", err)
	}

	// 发送加密数据
	if _, err := stream.Write(encryptedData); err != nil {
		return fmt.Errorf("发送数据失败: %v", err)
	}

	log.Printf("发送数据包到 %s: %s %s", addr, packet.Op, packet.Path)
	return nil
}

// getConnection 获取或创建到指定地址的连接
func (qt *QUICTransport) getConnection(addr string) (quic.Connection, error) {
	qt.connMutex.RLock()
	conn, exists := qt.conns[addr]
	qt.connMutex.RUnlock()

	if exists && conn.Context().Err() == nil {
		return conn, nil
	}

	qt.connMutex.Lock()
	defer qt.connMutex.Unlock()

	// 双重检查
	if conn, exists := qt.conns[addr]; exists && conn.Context().Err() == nil {
		return conn, nil
	}

	// 创建新连接
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"xsync"},
	}

	conn, err := quic.DialAddr(qt.ctx, addr, tlsConfig, &quic.Config{
		KeepAlivePeriod: 30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("创建QUIC连接失败: %v", err)
	}

	qt.conns[addr] = conn
	log.Printf("创建新的QUIC连接到: %s", addr)

	// 监控连接状态
	go qt.monitorConnection(addr, conn)

	return conn, nil
}

// monitorConnection 监控连接状态
func (qt *QUICTransport) monitorConnection(addr string, conn quic.Connection) {
	<-conn.Context().Done()
	qt.connMutex.Lock()
	delete(qt.conns, addr)
	qt.connMutex.Unlock()
	log.Printf("连接已断开: %s", addr)
}

// Listen 监听指定端口
func (qt *QUICTransport) Listen(port int, handler PacketHandler) error {
	tlsConfig := generateTLSConfig()

	listener, err := quic.ListenAddr(fmt.Sprintf(":%d", port), tlsConfig, &quic.Config{
		KeepAlivePeriod: 30 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("启动QUIC监听失败: %v", err)
	}

	qt.listener = listener
	log.Printf("QUIC服务器监听端口: %d", port)

	// 处理连接
	go qt.acceptConnections(handler)

	return nil
}

// acceptConnections 接受连接
func (qt *QUICTransport) acceptConnections(handler PacketHandler) {
	for {
		select {
		case <-qt.ctx.Done():
			return
		default:
			conn, err := qt.listener.Accept(qt.ctx)
			if err != nil {
				if qt.ctx.Err() != nil {
					return // 正常关闭
				}
				log.Printf("接受连接失败: %v", err)
				continue
			}

			go qt.handleConnection(conn, handler)
		}
	}
}

// handleConnection 处理连接
func (qt *QUICTransport) handleConnection(conn quic.Connection, handler PacketHandler) {
	remoteAddr := conn.RemoteAddr().String()
	log.Printf("接受新连接: %s", remoteAddr)

	for {
		select {
		case <-qt.ctx.Done():
			return
		case <-conn.Context().Done():
			return
		default:
			stream, err := conn.AcceptStream(qt.ctx)
			if err != nil {
				if qt.ctx.Err() != nil || conn.Context().Err() != nil {
					return // 正常关闭
				}
				log.Printf("接受流失败: %v", err)
				continue
			}

			go qt.handleStream(stream, handler, remoteAddr)
		}
	}
}

// handleStream 处理数据流
func (qt *QUICTransport) handleStream(stream quic.Stream, handler PacketHandler, remoteAddr string) {
	defer stream.Close()

	// 读取数据长度
	lengthBytes := make([]byte, 4)
	if _, err := io.ReadFull(stream, lengthBytes); err != nil {
		log.Printf("读取数据长度失败: %v", err)
		return
	}

	dataLength := int(lengthBytes[0])<<24 | int(lengthBytes[1])<<16 | int(lengthBytes[2])<<8 | int(lengthBytes[3])
	if dataLength <= 0 || dataLength > 100*1024*1024 { // 限制最大100MB
		log.Printf("无效的数据长度: %d", dataLength)
		return
	}

	// 读取加密数据
	encryptedData := make([]byte, dataLength)
	if _, err := io.ReadFull(stream, encryptedData); err != nil {
		log.Printf("读取加密数据失败: %v", err)
		return
	}

	// 解密数据包
	packet, err := protocol.DecryptPacket(encryptedData, qt.key)
	if err != nil {
		log.Printf("解密数据包失败: %v", err)
		return
	}

	log.Printf("接收数据包从 %s: %s %s", remoteAddr, packet.Op, packet.Path)

	// 处理数据包
	if err := handler(packet, remoteAddr); err != nil {
		log.Printf("处理数据包失败: %v", err)
	}
}

// Close 关闭传输器
func (qt *QUICTransport) Close() error {
	qt.cancel()

	// 关闭所有连接
	qt.connMutex.Lock()
	for _, conn := range qt.conns {
		conn.CloseWithError(0, "shutdown")
	}
	qt.connMutex.Unlock()

	// 关闭监听器
	if qt.listener != nil {
		return qt.listener.Close()
	}

	return nil
}

// generateTLSConfig 生成TLS配置
func generateTLSConfig() *tls.Config {
	// 动态生成自签名证书
	cert, err := generateSelfSignedCert()
	if err != nil {
		panic(err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"xsync"},
	}
}

// generateSelfSignedCert 生成自签名证书
func generateSelfSignedCert() (tls.Certificate, error) {
	// 生成私钥
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	// 创建证书模板
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"XSync"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour), // 1年有效期
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{"localhost"},
	}

	// 生成证书
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	// 创建TLS证书
	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}, nil
}