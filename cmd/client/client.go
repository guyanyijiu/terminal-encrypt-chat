package main

import (
	"bytes"
	"flag"
	log "github.com/sirupsen/logrus"
	"net"
	"terminal-encrypt-chat/crypto"
	"terminal-encrypt-chat/message"
	"terminal-encrypt-chat/transfer"
	"terminal-encrypt-chat/tui"
	"time"
)

var (
	id                   string
	address              string
	conn                 *transfer.Transfer
	secret               []byte
	tuiInputCh           = make(chan []byte)
	tuiOutputCh          = make(chan []byte)
	sendMessagePrefix    = []byte("> ")
	receiveMessagePrefix = []byte("- ")
)

type logOutput struct {
}

func (l *logOutput) Write(p []byte) (n int, err error) {
	tuiOutputCh <- p
	return len(p), nil
}

type logFormatter struct {
}

func (l *logFormatter) Format(e *log.Entry) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString("[")
	b.WriteString(e.Level.String())
	b.WriteString("] ")
	b.WriteString(e.Message)
	return b.Bytes(), nil
}

func main() {
	// 解析命令行参数
	flag.StringVar(&id, "i", "", "聊天 ID")
	flag.StringVar(&address, "h", "", "服务器地址 ip:port")
	flag.Parse()
	if id == "" || address == "" {
		flag.Usage()
		return
	}

	var err error

	// 设置日志
	log.SetOutput(&logOutput{})
	log.SetFormatter(&logFormatter{})
	//log.SetLevel(log.DebugLevel)
	log.SetLevel(log.InfoLevel)

	// 启动聊天界面
	tuiInputCh, tuiOutputCh, err = tui.New()
	if err != nil {
		log.Errorf("启动聊天界面错误: %s\n", err)
		return
	}

	log.Info("使用 ESC 或 Ctrl + C 退出")

	// 连接服务器
	go Connect()

	// 开始聊天
	tui.Start()
}

func Connect() {
	// 开始连接服务器
	log.Info("正在连接服务器...")
	tcp, err := net.DialTimeout("tcp", address, 30*time.Second)
	if err != nil {
		log.Errorf("连接服务器失败: %s", err)
		return
	}

	log.Info("连接服务器成功")

	cid := []byte(id)

	log.Infof("对方可以使用以下 ID 建立连接: %s", id)

	conn = transfer.NewTransfer(tcp)
	go func() {
		conn.WaitClose()
		tui.StopInput()
		log.Info("已和服务器断开连接")
	}()

	// 发送握手包
	conn.Send(message.NewMessage(message.MTypeHandShake, cid[:]))

	log.Info("等待对方连接...")

	// 接收握手包
	hsMessage := conn.Receive()
	if hsMessage.MType != message.MTypeHandShake {
		log.Errorf("握手失败: 未知的握手包 %v", hsMessage)
		return
	}

	log.Info("对方已连接")

	log.Info("正在协商密钥...")

	ecdh := crypto.NewCurve25519ECDH()
	privateKey, publicKey, err := ecdh.GenerateKey()
	publicKeyData := ecdh.Marshal(publicKey)

	log.Infof("向对方发送公钥: %x", publicKeyData)
	// 发送公钥
	conn.Send(message.NewMessage(message.MTypeSecret, publicKeyData))

	// 接收公钥
	pkMessage := conn.Receive()
	if pkMessage.MType != message.MTypeSecret {
		log.Errorf("协商密钥失败: 未知消息%v", pkMessage)
		return
	}

	// 解码公钥
	receivePublicKey, ok := ecdh.Unmarshal(pkMessage.Content)
	if !ok {
		log.Errorf("协商密钥失败，解码对方公钥错误: %s", err)
		log.Debugf("接收到对方公钥数据: %x", pkMessage.Content)
		return
	}

	log.Infof("接收到对方公钥: %x", pkMessage.Content)

	// 生成密钥
	secret, err = ecdh.GenerateSharedSecret(privateKey, receivePublicKey)
	if err != nil {
		log.Errorf("协商密钥失败，生成对称加密密钥错误: %s", err)
		return
	}

	log.Infof("协商密钥已成功: %x", secret)
	log.Infof("与对方的密钥一致时才能确认加密过程安全")

	// 开始互相传输数据
	tui.StartInput()

	go func() {
		for {
			for i := range tuiInputCh {
				t := make([]byte, len(sendMessagePrefix)+len(i))
				copy(t[:len(sendMessagePrefix)], sendMessagePrefix)
				copy(t[len(sendMessagePrefix):], i)
				tuiOutputCh <- t
				Send(i)
			}
		}
	}()

	go func() {
		for {
			data := Receive()
			if data != nil {
				t := make([]byte, len(receiveMessagePrefix)+len(data))
				copy(t[:len(receiveMessagePrefix)], receiveMessagePrefix)
				copy(t[len(receiveMessagePrefix):], data)
				tuiOutputCh <- t
			}
		}
	}()
}

func Send(data []byte) {
	content, err := crypto.Encrypt(data, secret)
	if err != nil {
		log.Warnf("加密消息失败: %v", err)
		return
	}
	conn.Send(message.NewMessage(message.MTypeData, content))
}

func Receive() []byte {
	m := conn.Receive()
	if m.MType == message.MTypeClose {
		conn.Close()
		log.Info("对方已关闭连接")
		return nil
	}
	if m.MType != message.MTypeData {
		return nil
	}
	content, err := crypto.Decrypt(m.Content, secret)
	if err != nil {
		log.Warnf("解密消息失败: %v", err)
		return nil
	}

	return content
}
