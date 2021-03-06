package transports

import (
	"errors"
	"io"
	"sync"
	"time"
)

type ChannelWriter struct {
	inner   io.WriteCloser
	channel chan []byte
	once    sync.Once
}

func NewChannelWriter(inner io.WriteCloser, capacity int) io.WriteCloser {
	this := &ChannelWriter{inner: inner, channel: make(chan []byte, capacity)}
	go this.listen() // design note: who should start the listener?
	return this
}

func (this *ChannelWriter) Write(buffer []byte) (int, error) {
	select {
	case this.channel <- buffer:
		return len(buffer), nil
	default:
		return 0, ErrBufferFull
	}
}

func (this *ChannelWriter) listen() {
	defer this.inner.Close()

	for buffer := range this.channel {
		for attempt := time.Millisecond; !this.write(buffer) && attempt < 10; attempt++ {
			time.Sleep(attempt * 100)
		}
	}
}
func (this *ChannelWriter) write(buffer []byte) bool {
	_, err := this.inner.Write(buffer)
	return err == nil
}

func (this *ChannelWriter) Close() error {
	this.once.Do(func() { close(this.channel) })
	return nil
}

var ErrBufferFull = errors.New("buffer full")
