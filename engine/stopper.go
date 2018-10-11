package engine

type Stopper struct {
	stop chan struct{}
	stopped chan struct{}
}

func (w *Stopper) ShouldStop() bool {
	select {
	case <-w.stop:
		return true
	default:
	}
	return false
}

func (w *Stopper) Stopped() {
	w.stopped <- struct{}{}
}
