package transports

import (
	"errors"
	"io"
	"sync"
	"time"
)

type ChannelWriter struct {
	dialer   Dialer
	address  string
	capacity int
	mutex    *sync.Mutex
	incoming [][]byte
	outgoing [][]byte
	writer   io.WriteCloser
	closed   bool
}

func NewChannelWriter(dialer Dialer, address string, capacity int) io.WriteCloser {
	this := &ChannelWriter{dialer: dialer, address: address, mutex: &sync.Mutex{}, capacity: capacity}
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
		return 0, ErrBufferFull
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
	for this.process() {
	}
}
func (this *ChannelWriter) process() bool {
	if closed, hasWork := this.read(); closed {
		return false
	} else if hasWork {
		this.tryWrite()
	} else {
		time.Sleep(time.Millisecond) // waiting for work
	}

	return true
}

func (this *ChannelWriter) read() (closed bool, hasWork bool) {
	this.mutex.Lock()
	closed = this.closed
	hasWork = len(this.incoming) > 0 || len(this.outgoing) > 0
	this.swapBuffers()
	this.mutex.Unlock()
	return closed, hasWork
}
func (this *ChannelWriter) swapBuffers() {
	if this.closed || len(this.outgoing) > 0 || len(this.incoming) == 0 {
		return
	}
	temp := this.outgoing
	this.outgoing = this.incoming
	this.incoming = temp
}

func (this *ChannelWriter) tryWrite() {
	if !this.openWriter() {
		return
	}

	if this.writeBuffer() {
		return
	}

	this.closeWriter() // write failed, close
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

var ErrBufferFull = errors.New("buffer full")
