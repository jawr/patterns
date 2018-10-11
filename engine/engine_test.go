package engine

import (
	"testing"
	"sync"
	"sync/atomic"
)

type M struct {
	Message string
}

type Suzuki struct {
	Engine
}

func createEngine(h HandlerFunc) *Suzuki {
	s := Suzuki{}
	s.RegisterHandler("test", h)
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

	e.Workers = 10

	err := e.Start()
	if err != nil {
		t.Errorf("Failed to start")
	}

	wg.Add(1)
	err = e.Add("test", M{"pass"})
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

	err := e.Add("test", M{"pass"})
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
		e.Add("test", struct{}{})
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


func TestDifferentHandlers(t *testing.T) {

	var wg sync.WaitGroup

	wg.Add(4)

	var test, a, b, c bool

	e := createEngine(func(i Item) { 
		test = true 
		wg.Done()
	})
	e.RegisterHandler("a", func(i Item) { 
		a = true 
		wg.Done()
	})
	e.RegisterHandler("b", func(i Item) {
		b = true 
		wg.Done()
	})
	e.RegisterHandler("c", func(i Item) {
		c = true 
		wg.Done()
	})

	err := e.Start()
	if err != nil {
		t.Errorf("Failed to start")
	}

	for _, i := range []string{"test", "a", "b", "c"} {
		e.Add(i, struct{}{})
	}

	err = e.Stop()
	if err != nil {
		t.Errorf("Failed to stop")
	}

	wg.Wait()

	if !(test && a && b && c) {
		t.Errorf("expected all to be true test: %t a: %t b: %t c: %t", test, a, b, c)
	}
}

