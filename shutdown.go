package shutdown

import "sync"

type Shutdown struct {
	lock sync.Mutex
	ch   chan struct{}
	err  error
	once sync.Once
	call func()
}

func New(f func()) *Shutdown {
	return &Shutdown{
		ch:   make(chan struct{}),
		call: f,
	}
}

func (s *Shutdown) Done() chan struct{} {
	return s.ch
}

func (s *Shutdown) Shut(err error) {
	s.once.Do(func() {
		s.lock.Lock()

		s.err = err
		close(s.ch)

		s.lock.Unlock()

		if s.call != nil {
			s.call()
		}
	})
}

func (s *Shutdown) Down() bool {
	select {
	case <-s.ch:
		return true
	default:
		return false
	}
}

func (s *Shutdown) Err() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.err
}
