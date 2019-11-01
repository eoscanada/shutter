package shutter

import (
	"errors"
	"sync"
)

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

var ErrShutterWasAlreadyDown = errors.New("saferun was called on an already-shutdown shutter")

// LockedInit allows you to run a function only if the shutter is not down yet,
// with the assurance that the it will not run its callback functions
// during the execution of your function.
//
// This is useful to prevent race conditions, where the func given to "LockedInit"
// should increase a counter and the func given to OnShutdown should decrease it.
//
// WARNING: never call Shutdown from within your LockedInit function,
// it will deadlock. Also, keep these init functions as short as
// possible.
//
// NOTE: This was previously named SafeRun
func (s *Shutter) LockedInit(fn func() error) (err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.IsDown() {
		return ErrShutterWasAlreadyDown
	}
	return fn()
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
