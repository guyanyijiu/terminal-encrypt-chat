package transfer

import (
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"terminal-encrypt-chat/message"
)

type Transfer struct {
	conn       net.Conn
	closeCh    chan struct{}
	mutex      *sync.Mutex
	closed     bool
	readQueue  chan *message.Message
	writeQueue chan *message.Message
}

func NewTransfer(conn net.Conn) *Transfer {
	t := &Transfer{
		conn:       conn,
		closeCh:    make(chan struct{}),
		closed:     false,
		mutex:      &sync.Mutex{},
		readQueue:  make(chan *message.Message, 2),
		writeQueue: make(chan *message.Message, 2),
	}

	// reader
	go t.read()

	// writer
	go t.write()

	return t
}

func (t *Transfer) read() {
	for {
		m := &message.Message{}
		err := m.Unpack(t.conn)
		if err != nil {
			t.Close()
			log.Debug("接收数据失败，连接已被断开")
			break
		}
		t.readQueue <- m
	}

}

func (t *Transfer) write() {
	for {
		m := <-t.writeQueue
		err := m.Pack(t.conn)
		if err != nil {
			t.Close()
			log.Debug("发送数据失败，连接已被断开")
			return
		}
	}
}

func (t *Transfer) Receive() *message.Message {
	return <-t.readQueue
}

func (t *Transfer) Send(m *message.Message) {
	t.writeQueue <- m
}

func (t *Transfer) Close() {
	t.mutex.Lock()
	if !t.closed {
		t.conn.Close()
		close(t.closeCh)
		t.closed = true
	}
	t.mutex.Unlock()
}

func (t *Transfer) WaitClose() {
	<-t.closeCh
}
