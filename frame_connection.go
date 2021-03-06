package transports

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

type FrameConnection struct {
	net.Conn
}

func NewFrameConnection(inner net.Conn) FrameConnection {
	return FrameConnection{Conn: inner}
}

func (this FrameConnection) Write(buffer []byte) (int, error) {
	payloadSize := len(buffer)
	if payloadSize == 0 {
		return 0, nil
	}

	if payloadSize > MaxWriteSize {
		return 0, BufferTooLarge
	}

	if err := binary.Write(this.Conn, byteOrdering, uint16(payloadSize)); err != nil {
		return 0, err
	}

	return this.Conn.Write(buffer)
}
func (this FrameConnection) Read(buffer []byte) (int, error) {
	if length, err := this.ReadHeader(); err != nil {
		return 0, err
	} else if length > len(buffer) {
		return 0, io.ErrShortBuffer
	} else {
		return this.ReadBody(buffer[0:length])
	}
}
func (this FrameConnection) ReadHeader() (int, error) {
	var length uint16
	if err := binary.Read(this.Conn, byteOrdering, &length); err != nil {
		return 0, err
	} else {
		return int(length), nil
	}
}
func (this FrameConnection) ReadBody(buffer []byte) (int, error) {
	return io.ReadFull(this.Conn, buffer)
}

const MaxWriteSize = 64*1024 - 2

var (
	BufferTooLarge = errors.New("buffer larger than the max frame size")
	byteOrdering   = binary.LittleEndian
)
