package engine

import (
	"sync/atomic"
	"sync"
)

type Item interface {}
type HandlerFunc func(Item)


type wrapper struct{
	handler string
	item Item
}

type Engine struct {
	BufferSize int
	Workers int

	running int32
	runningMtx sync.Mutex

	started chan struct{}

	stopping int32
	stoppingMtx sync.Mutex

	stop chan struct{}
	stopped chan struct{}

	incoming chan wrapper

	stoppers []*Stopper
	stoppersMtx sync.Mutex

	handlers  map[string]HandlerFunc
	handlersMtx sync.RWMutex
	handlersOnce sync.Once
}

func (e *Engine) initHandlers() {
	e.handlersOnce.Do(func() {
		e.handlers = make(map[string]HandlerFunc, 0)
	})
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

	e.incoming = make(chan wrapper, e.BufferSize)

	e.initHandlers()

	go func() {

		// semaphore for workers
		if e.Workers == 0 {
			e.Workers = 1
		}

		sem := make(chan struct{}, e.Workers)

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

			case w := <-e.incoming:

				e.handlersMtx.RLock()
				fn, ok := e.handlers[w.handler]
				e.handlersMtx.RUnlock()

				if !ok {
					continue
				}

				sem <- struct{}{}

				go func(i Item) {
					fn(i)
					<-sem
				}(w.item)
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

func (e *Engine) Add(h string, i Item) error {
	if atomic.LoadInt32(&e.running) == 0 {
		return ErrNotRunning
	}
	if atomic.LoadInt32(&e.stopping) == 1 {
		return ErrStopping
	}

	e.incoming <- wrapper{h, i}

	return nil
}

func (e *Engine) RegisterHandler(h string, fn HandlerFunc) {
	e.initHandlers()

	e.handlersMtx.Lock()
	e.handlers[h] = fn
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
