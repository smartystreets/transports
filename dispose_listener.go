package transports

import (
	"io"
	"net"
	"sync"
)

type DisposeListener struct {
	inner   net.Listener
	mutex   *sync.Mutex
	tracked map[io.Closer]struct{} // holds the UNDERLYING/actual socket, not the wrapped/decorated one
}

func NewDisposeListener(inner net.Listener) DisposeListener {
	return DisposeListener{
		inner:   inner,
		mutex:   &sync.Mutex{},
		tracked: make(map[io.Closer]struct{}),
	}
}

func (this *DisposeListener) Accept() (net.Conn, error) {
	if actual, err := this.inner.Accept(); err != nil {
		return nil, err
	} else {
		this.track(actual)
		return NewDisposeConnection(actual, this.dispose), nil
	}
}
func (this *DisposeListener) track(actual net.Conn) {
	this.mutex.Lock()
	this.tracked[actual] = struct{}{}
	this.mutex.Unlock()

}
func (this *DisposeListener) dispose(actual io.Closer) {
	this.mutex.Lock()
	delete(this.tracked, actual)
	this.mutex.Unlock()
}

func (this *DisposeListener) Close() error {
	err := this.inner.Close()

	this.mutex.Lock()
	defer this.mutex.Unlock() // defer because we could a panic if a socket's Close() method is buggy,

	for actual := range this.tracked {
		actual.Close() // closes underlying so this.dispose (with mutex) is never called directly
	}
	this.tracked = nil // everything's closed, we don't need the map anymore

	return err
}

func (this *DisposeListener) IsActive() bool {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	return this.tracked == nil
}