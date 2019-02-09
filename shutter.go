package shutter

import "sync"

type Shutter struct {
	lock     sync.Mutex // shutdown lock
	ch       chan struct{}
	err      error
	once     sync.Once
	calls    []func(error)
	callLock sync.Mutex
}

func New() *Shutter {
	s := &Shutter{
		ch: make(chan struct{}),
	}
	return s
}

func NewWithCallback(f func(error)) *Shutter {
	s := &Shutter{
		ch: make(chan struct{}),
	}
	s.OnShutdown(f)
	return s
}

func (s *Shutter) Shutdown(err error) {
	var execute = false
	s.once.Do(func() {
		execute = true
	})

	if !execute {
		return
	}

	s.lock.Lock()
	// assign s.err before closing channel, so `IsDown()` and `Done()`
	// return when `Err()` is *always* available.
	s.err = err
	close(s.ch)
	s.lock.Unlock()

	s.callLock.Lock()
	for _, call := range s.calls {
		call(err)
	}
	s.callLock.Unlock()
}

func (s *Shutter) Done() <-chan struct{} {
	return s.ch
}

func (s *Shutter) IsDown() bool {
	select {
	case <-s.ch:
		return true
	default:
		return false
	}
}

// OnShutdown registers an additional handler to be triggered on
// `Shutdown()`. These calls will be blocking. It is unsafe to
// register new callbacks in multiple go-routines.
func (s *Shutter) OnShutdown(f func(error)) {
	s.callLock.Lock()
	s.calls = append(s.calls, f)
	s.callLock.Unlock()
}

func (s *Shutter) Err() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.err
}
