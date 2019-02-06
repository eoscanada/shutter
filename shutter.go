package shutter

import "sync"

type Shutter struct {
	lock sync.Mutex // shutdown lock
	ch   chan struct{}
	err  error
	once sync.Once

	call     func()
	callLock sync.Mutex
}

func New(f func()) *Shutter {
	return &Shutter{
		ch:   make(chan struct{}),
		call: f,
	}
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
	if s.call != nil {
		s.call()
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

func (s *Shutter) SetCallback(f func()) {
	s.callLock.Lock()
	s.call = f
	s.callLock.Unlock()
}

func (s *Shutter) SetErrorCallback(f func(error)) {
	s.callLock.Lock()
	s.call = func() {
		f(s.Err())
	}
	s.callLock.Unlock()
}

func (s *Shutter) Err() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.err
}
