package destinations

import (
	"context"
	"errors"
	"sync"
	"time"
)

var errWorkerStopping = errors.New("worker is stopping")

// periodicWorker owns the start/stop boundary shared by DBM's periodic
// monitors. A stop remains visible until the active reconciliation and its
// post-stop drain have both completed, so a concurrent Start cannot overlap
// work that quiescence is still waiting to finish.
type periodicWorker struct {
	mu          sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
	stopping    bool
	stopPending bool
	stopDone    chan struct{}
}

func (w *periodicWorker) start(interval time.Duration, beforeStart func(), reconcile func(context.Context)) error {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopping || w.stopPending {
		return errWorkerStopping
	}
	if w.cancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	w.cancel, w.done = cancel, done
	if beforeStart != nil {
		beforeStart()
	}
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		reconcile(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reconcile(ctx)
			}
		}
	}()
	return nil
}

func (w *periodicWorker) stop(ctx context.Context, afterStop func(context.Context) error) error {
	if w == nil {
		return nil
	}
	for {
		w.mu.Lock()
		cancel, done := w.cancel, w.done
		if cancel == nil {
			w.mu.Unlock()
			return nil
		}
		if w.stopping {
			stopDone := w.stopDone
			w.mu.Unlock()
			select {
			case <-stopDone:
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		w.stopping = true
		stopDone := make(chan struct{})
		w.stopDone = stopDone
		w.mu.Unlock()

		cancel()
		var err error
		select {
		case <-done:
		case <-ctx.Done():
			err = ctx.Err()
		}
		if err == nil && afterStop != nil {
			err = afterStop(ctx)
		}

		w.mu.Lock()
		if w.stopDone == stopDone {
			if err == nil {
				w.cancel, w.done = nil, nil
				w.stopPending = false
			} else {
				// Keep Start closed until a later Stop retries the remaining join or
				// post-stop drain. The generation is canceled even when quiescence
				// did not finish, so treating it as running would lose the worker.
				w.stopPending = true
			}
			w.stopping, w.stopDone = false, nil
			close(stopDone)
		}
		w.mu.Unlock()
		return err
	}
}
