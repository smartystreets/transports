package transports

import (
	"errors"
	"io"
	"sync"
)

type ChannelWriter struct {
	dialer   Dialer
	address  string
	writer   io.WriteCloser
	incoming [][]byte
	outgoing [][]byte
	mutex    *sync.Mutex
	capacity int
	closed   bool
}

func NewChannelWriter(dialer Dialer, address string, capacity uint16) io.WriteCloser {
	this := &ChannelWriter{dialer: dialer, address: address, mutex: &sync.Mutex{}}
	go this.listen()
	return this
}

func (this *ChannelWriter) Write(buffer []byte) (int, error) {
	this.mutex.Lock()
	written, err := this.write(buffer)
	this.mutex.Unlock()
	return written, err
}
func (this *ChannelWriter) write(buffer []byte) (int, error) {
	if this.closed {
		return 0, ErrClosedSocket
	}
	if len(this.incoming) >= this.capacity {
		return 0, ErrChannelFull
	}
	this.incoming = append(this.incoming, buffer)
	return len(buffer), nil
}
func (this *ChannelWriter) Close() error {
	this.mutex.Lock()
	this.closed = true
	this.mutex.Unlock()
	return nil
}

func (this *ChannelWriter) listen() {
	defer this.closeWriter()

	for !this.closed {
		this.swapBuffers()
	}
}
func (this *ChannelWriter) swapBuffers() {
	this.mutex.Lock()
	temp := this.outgoing
	this.outgoing = this.incoming
	this.incoming = temp
	this.mutex.Unlock()
}

func (this *ChannelWriter) ensureWrite() {
	for !this.closed {
		if !this.openWriter() {
			continue
		}

		if this.writeBuffer() {
			break // done
		}

		this.closeWriter() // write failed, close
	}
}
func (this *ChannelWriter) writeBuffer() bool {
	for _, message := range this.outgoing {
		if _, err := this.writer.Write(message); err != nil {
			return false
		}
	}

	this.outgoing = this.outgoing[0:0]
	return true
}

func (this *ChannelWriter) openWriter() bool {
	if this.writer != nil {
		return true
	} else if socket, err := this.dialer.Dial("tcp", this.address); err != nil {
		return false
	} else {
		this.writer = socket
		return true
	}
}
func (this *ChannelWriter) closeWriter() {
	if this.writer != nil {
		this.writer.Close()
		this.writer = nil
	}
}

var ErrChannelFull = errors.New("channel full")
