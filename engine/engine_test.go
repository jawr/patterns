package engine

import (
	"testing"
	"sync"
	"fmt"
	"sync/atomic"
)

type M struct {
	Message string
}

type Suzuki struct {
	Engine
}

func createEngine(h HandleFunc) *Suzuki {
	s := Suzuki{}
	s.RegisterHandler(h)
	return &s
}

func TestStartSendStop(t *testing.T) {
	var wg sync.WaitGroup

	e := createEngine(func(i Item) {
		if i == nil {
			t.Errorf("Failed; item is nil")
		}

		if i.(M).Message != "pass" {
			t.Errorf("Failed to cast and read item")
		}

		wg.Done()
	})

	err := e.Start()
	if err != nil {
		t.Errorf("Failed to start")
	}

	wg.Add(1)
	err = e.Add(M{"pass"})
	if err != nil {
		t.Errorf("Failed to add item")
	}

	err = e.Stop()
	if err != nil {
		t.Errorf("Failed to stop")
	}

	wg.Wait()
}

func TestDoubleStart(t *testing.T) {
	e := createEngine(func(i Item) {})

	err := e.Start()
	if err != nil {
		t.Errorf("Failed to start")
	}
	err = e.Start()
	if err != ErrAlreadyRunning {
		t.Errorf("Double start lock did not work")
	}
}

func TestDoubleStop(t *testing.T) {
	e := createEngine(func(i Item) {})

	err := e.Start()
	if err != nil {
		t.Errorf("Failed to start")
	}
	err = e.Stop()
	if err != nil {
		t.Errorf("Failed to stop")
	}

	err = e.Stop()
	if err != ErrAlreadyStopping {
		t.Errorf("Double stop lock did not work")
	}
}

func TestAddNotRunning(t *testing.T) {
	e := createEngine(func(i Item) {})

	err := e.Add(M{"pass"})
	if err != ErrNotRunning {
		t.Errorf("Can add whilest not running")
	}
}

func TestStopNotRunning(t *testing.T) {
	e := createEngine(func(i Item) {})

	err := e.Stop()
	if err != ErrNotRunning {
		t.Errorf("Can stop whilest not running")
	}
}

func TestDrain(t *testing.T) {
	var count int32
	var wg sync.WaitGroup

	var expected int32 = 10

	e := createEngine(func(i Item) {
		atomic.AddInt32(&count, 1)
		wg.Done()
	})
	e.BufferSize = int(expected * 2)

	err := e.Start()
	if err != nil {
		t.Errorf("Failed to start")
	}

	for i := 0; i < int(expected); i++ {
		wg.Add(1)
		e.Add(struct{}{})
	}

	err = e.Stop()
	if err != nil {
		t.Errorf("Failed to stop")
	}

	wg.Wait()

	// should be synced
	if count != expected {
		t.Errorf("expected %d and got %d", expected, count)
	}
}


func TestStoppers(t *testing.T) {
	e := createEngine(func(i Item) {})

	err := e.Start()
	if err != nil {
		t.Errorf("Failed to start")
	}

	s := e.GetStopper()
	fmt.Printf("got stopper %v", s)
	go func() {
		for {
			if s.ShouldStop() {
				s.Stopped()
				return
			}
		}
	}()

	err = e.Stop()
	if err != nil {
		t.Errorf("Failed to stop")
	}
}

