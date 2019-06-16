package main

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"terminal-encrypt-chat/message"
	"terminal-encrypt-chat/transfer"
	"time"
)

type chat struct {
	Id       string
	Conn     net.Conn
	Target   *chat
	Transfer *transfer.Transfer
}

var (
	chats = make(map[string]*chat)
	mutex = &sync.Mutex{}
)

func main() {
	log.SetLevel(log.DebugLevel)
	address := ":9468"
	listen, err := net.Listen("tcp", address)
	if err != nil {
		log.Error("Fail to listen address: ", address)
		return
	}

	go func() {
		time.Sleep(300 * time.Second)
		buf := new(bytes.Buffer)
		for k, v := range chats {
			buf.WriteString(k)
			buf.WriteString(",")
			buf.WriteString(v.Conn.RemoteAddr().String())
			buf.WriteString("\n")
		}
		fmt.Println(buf.String())
	}()

	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Warn("Fail to accept:", err)
			continue
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()
	log.Debugf("%s 建立连接\n", remoteAddr)

	tf := transfer.NewTransfer(conn)

	// 接收握手包
	hsMessage := tf.Receive()
	if hsMessage.MType != message.MTypeHandShake {
		log.Debugf("%s 接收到非握手包\n", remoteAddr)
		log.Debug(hsMessage)
		return
	}

	id := string(hsMessage.Content)
	log.Debugf("%s 获取到聊天 ID: %s\n", remoteAddr, id)

	c1 := &chat{
		Id:       id,
		Conn:     conn,
		Target:   nil,
		Transfer: tf,
	}

	mutex.Lock()
	if c2, ok := chats[id]; ok {
		// 判断是否已经建立了连接
		if c2.Target == c1 {
			c1.Transfer.Send(message.NewMessage(message.MTypeClose, []byte("ID 已被占用")))
			log.Debugf("%s ID 已被占用\n", remoteAddr)
		} else {
			c1.Target = c2
			c2.Target = c1
			c1.Transfer.Send(hsMessage)
			c2.Transfer.Send(hsMessage)
			log.Debugf("%s 已建立联系", remoteAddr)
		}

	} else {
		chats[id] = c1
		log.Debugf("%s 未建立联系", remoteAddr)
	}
	mutex.Unlock()

	// 转发消息
	go func() {
		for {
			m := c1.Transfer.Receive()
			if c1.Target != nil {
				log.Debugf("%s -> %s : %v\n", remoteAddr, c1.Target.Conn.RemoteAddr().String(), m)
				c1.Target.Transfer.Send(m)
			}
		}
	}()

	// 如果断开连接
	c1.Transfer.WaitClose()
	log.Debugf("%s 连接关闭", remoteAddr)
	delete(chats, c1.Id)
	if c1.Target != nil {
		c1.Target.Transfer.Send(message.NewMessage(message.MTypeClose, []byte("对方已断开")))
	}
}
