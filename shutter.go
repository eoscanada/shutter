package shutter

import (
	"errors"
	"sync"
)

type Shutter struct {
	lock     sync.Mutex // shutdown lock

	terminatingCh chan struct{}
	terminatedCh chan struct{}

	//ch       chan struct{}
	err      error
	once     sync.Once
	calls    []func(error)
	callLock sync.Mutex
}

func New() *Shutter {
	s := &Shutter{
		terminatingCh: make(chan struct{}),
		terminatedCh:make(chan struct{}),
		//ch: make(chan struct{}),
	}
	return s
}

func NewWithCallback(f func(error)) *Shutter {
	s := &Shutter{
		terminatingCh: make(chan struct{}),
		terminatedCh:make(chan struct{}),
		//ch: make(chan struct{}),
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
	if s.IsTerminating() {
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
	close(s.terminatingCh)
	s.lock.Unlock()

	s.callLock.Lock()
	for _, call := range s.calls {
		call(err)
	}
	s.callLock.Unlock()

	s.lock.Lock()
	// the err has been handle above thus it will be available
	close(s.terminatedCh)
	s.lock.Unlock()
}

func (s *Shutter) Terminating() <-chan struct{} {
	return s.terminatingCh
}

func (s *Shutter) IsTerminating() bool {
	select {
	case <-s.terminatingCh:
		return true
	default:
		return false
	}
}

func (s *Shutter) Terminated() <-chan struct{} {
	return s.terminatedCh
}

func (s *Shutter) IsTerminated() bool {
	select {
	case <-s.terminatedCh:
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
