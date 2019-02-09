package shutter

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
