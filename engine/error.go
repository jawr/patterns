package engine

import (
	"fmt"
)

var ErrNotRunning = Errorf("engine: not running")
var ErrAlreadyRunning = Errorf("engine: already running")
var ErrAlreadyStopping = Errorf("engine: already stopping")

type Error struct {
	s string
}

func Errorf(s string, args ...interface{}) Error {
	return Error{s: fmt.Sprintf(s, args...)}
}

func (err Error) Error() string {
	return err.s
}
