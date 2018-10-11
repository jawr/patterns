package engine

import (
	"sync/atomic"
	"sync"
)

type Item interface {}
type HandlerFunc func(Item)


type Engine struct {
	BufferSize int

	running int32
	runningMtx sync.Mutex

	started chan struct{}

	stopping int32
	stoppingMtx sync.Mutex

	stop chan struct{}
	stopped chan struct{}

	incoming chan Item

	stoppers []*Stopper
	stoppersMtx sync.Mutex

	handlers  []HandlerFunc
	handlersMtx sync.Mutex
}

// Engines exist for as long as they are run
func (e *Engine) Start() error {
	if atomic.LoadInt32(&e.running) == 1 {
		return ErrAlreadyRunning
	}

	e.runningMtx.Lock()
	if e.running == 1 {
		return ErrAlreadyRunning
	}
	e.running = 1
	e.runningMtx.Unlock()

	e.started = make(chan struct{}, 0)

	e.stop = make(chan struct{}, 0)
	e.stopped = make(chan struct{}, 0)

	e.incoming = make(chan Item, e.BufferSize)

	go func() {
MAIN:
		for {
			select {
			case e.started<-struct{}{}:
				continue

			case <-e.stop:
				// drain the queue
				close(e.incoming)

				e.handlersMtx.Lock()
				for i := range e.incoming {
					for _, h := range e.handlers {
						h(i)
					}
				}
				e.handlersMtx.Unlock()

				// tell the stoppers to quit
				e.stoppersMtx.Lock()
				for _, s := range e.stoppers {
					s.stop <- struct{}{}
					<-s.stopped
				}
				e.stoppersMtx.Unlock()

				break MAIN

			case i := <-e.incoming:
				// potential long wait for the addHandlers (which should be an init thing usually), but better
				// than the performance hit with allocating and locking
				e.handlersMtx.Lock()
				// run synchronously
				var hwg sync.WaitGroup
				for _, h := range e.handlers {
					hwg.Add(1)
					go func(h HandlerFunc) {
						defer hwg.Done()
						h(i)
					}(h)
				}
				e.handlersMtx.Unlock()

				hwg.Wait()
			}
		}

		e.stopped <- struct{}{}
	}()

	<-e.started

	return nil
}

func (e *Engine) Stop() error {
	if atomic.LoadInt32(&e.running) == 0 {
		return ErrNotRunning
	}

	if atomic.LoadInt32(&e.stopping) == 1 {
		return ErrAlreadyStopping
	}

	e.stoppingMtx.Lock()
	if e.stopping == 1 {
		return ErrAlreadyStopping
	}
	e.stopping = 1
	e.stoppingMtx.Unlock()

	e.stop <- struct{}{}

	<-e.stopped

	return nil
}

func (e *Engine) Add(i Item) error {
	if atomic.LoadInt32(&e.running) == 0 {
		return ErrNotRunning
	}
	if atomic.LoadInt32(&e.stopping) == 1 {
		return ErrStopping
	}


	e.incoming <- i
	return nil
}

func (e *Engine) RegisterHandler(h HandlerFunc) {
	e.handlersMtx.Lock()
	e.handlers = append(e.handlers, h)
	e.handlersMtx.Unlock()
}

func (e *Engine) GetStopper() *Stopper {
	s := &Stopper{
		stop: make(chan struct{}, 0),
		stopped: make(chan struct{}, 0),
	}
	e.stoppersMtx.Lock()
	e.stoppers = append(e.stoppers, s)
	e.stoppersMtx.Unlock()
	return s
}
