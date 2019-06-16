package message

import (
	"encoding/binary"
	"io"
)

const (
	MTypeHandShake = '1'
	MTypeSecret    = '2'
	MTypeData      = '3'
	MTypeClose     = '4'

	SizeMType  = 1
	SizeLength = 8
)

type Message struct {
	MType   byte
	length  uint64
	Content []byte
}

func NewMessage(mtype byte, content []byte) *Message {
	return &Message{
		MType:   mtype,
		length:  uint64(len(content)),
		Content: content,
	}
}

func (m *Message) Pack(writer io.Writer) error {
	buf := make([]byte, SizeMType+SizeLength+m.length)
	buf[0] = m.MType

	binary.BigEndian.PutUint64(buf[1:SizeMType+SizeLength], m.length)
	copy(buf[SizeMType+SizeLength:], m.Content)

	_, err := writer.Write(buf)
	return err
}

func (m *Message) Unpack(reader io.Reader) error {
	var err error
	mtype := make([]byte, SizeMType)
	length := make([]byte, SizeLength)

	_, err = io.ReadFull(reader, mtype)
	if err != nil {
		return err
	}

	m.MType = mtype[0]

	_, err = io.ReadFull(reader, length)
	if err != nil {
		return err
	}
	m.length = binary.BigEndian.Uint64(length)
	content := make([]byte, m.length)
	_, err = io.ReadFull(reader, content)
	if err != nil {
		return err
	}

	m.Content = content

	return nil
}
