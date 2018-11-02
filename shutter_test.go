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
	s := New(func() {
		obj.Shutdown(errors.New("ouch"))
	})
	obj.Shutter = s

	obj.Shutdown(errors.New("first"))

	assert.Equal(t, obj.Err(), errors.New("first"))
}
