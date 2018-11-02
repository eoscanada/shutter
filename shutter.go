package shutter

import "sync"

type Shutter struct {
	lock sync.Mutex
	ch   chan struct{}
	err  error
	once sync.Once
	call func()
}

func New(f func()) *Shutter {
	return &Shutter{
		ch:   make(chan struct{}),
		call: f,
	}
}

func (s *Shutter) Done() chan struct{} {
	return s.ch
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
	s.err = err
	close(s.ch)
	s.lock.Unlock()

	if s.call != nil {
		s.call()
	}
}

func (s *Shutter) IsDown() bool {
	select {
	case <-s.ch:
		return true
	default:
		return false
	}
}

func (s *Shutter) Err() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.err
}
