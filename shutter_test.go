package shutter

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)


func TestShutterTerminating(t *testing.T) {
	a := 0
	s := NewWithCallback(func(_ error) {
		time.Sleep(10*time.Millisecond)
		a++
	})
	go func() {
		select {
		case <-s.Terminating():
			assert.Equal(t, 0, a)
		case <-s.Terminated():
			assert.Equal(t, 1, a)

		case <-time.After(50 * time.Millisecond):
			t.Errorf("terminating channel was not closed as expected")
		}
	}()
	s.Shutdown(nil)
}

func TestShutterTerminated(t *testing.T) {
	a := 0
	s := NewWithCallback(func(_ error) {
		time.Sleep(10*time.Millisecond)
		a++
	})
	go func() {
		select {
		case <-s.Terminated():
			assert.Equal(t, 1, a)
		case <-time.After(50 * time.Millisecond):
			t.Errorf("terminating channel was not closed as expected")
		}
	}()
	s.Shutdown(nil)
}


func TestShutterDeadlock(t *testing.T) {
	obj := struct {
		*Shutter
	}{}
	s := NewWithCallback(func(_ error) {
		obj.Shutdown(errors.New("ouch"))
	})
	obj.Shutter = s

	obj.Shutdown(errors.New("first"))

	assert.Equal(t, obj.Err(), errors.New("first"))
}

func TestMultiCallbacks(t *testing.T) {
	s := New()
	var a int
	s.OnShutdown(func(_ error) {
		a++
	})
	s.OnShutdown(func(_ error) {
		a++
	})
	s.Shutdown(nil)
	assert.Equal(t, 2, a)
}

func TestLockedInitAlreadyShutdown(t *testing.T) {
	s := New()
	a := 0
	s.OnShutdown(func(_ error) {
		a--
	})
	s.Shutdown(nil)
	err := s.LockedInit(func() error {
		a++
		return nil
	})

	assert.Equal(t, -1, a)
	assert.Equal(t, ErrShutterWasAlreadyDown, err)
}

func TestLockedInitNotShutdown(t *testing.T) {
	s := New()
	a := 0
	s.OnShutdown(func(_ error) {
		a--
	})
	err := s.LockedInit(func() error {
		a++
		return nil
	})
	assert.NoError(t, err)
	s.Shutdown(nil)
	assert.Equal(t, 0, a)
}

func TestShutdownDuringLockedInit(t *testing.T) {
	s := New()

	a := 0
	s.OnShutdown(func(_ error) {
		a--
	})

	var err error
	inLockedInitCh := make(chan interface{})
	shutdownCalled := make(chan interface{})

	go func() {
		err = s.LockedInit(func() error {
			close(inLockedInitCh)
			select {
			case <-shutdownCalled:
				t.Errorf("Shutdown was called and completed while in LockedInit")
			case <-time.After(50 * time.Millisecond):
				return nil
			}
			return nil
		})
	}()

	<-inLockedInitCh
	go func() {
		s.Shutdown(nil)
		close(shutdownCalled)
	}()
	assert.NoError(t, err)
	<-shutdownCalled
	assert.Equal(t, -1, a)
}
