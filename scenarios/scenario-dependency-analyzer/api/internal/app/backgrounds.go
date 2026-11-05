package app

import (
	"context"
	"errors"
	"sync"
)

// backgroundWorkers owns the detached loops that must stop at a maintenance
// fence. Every worker receives one shared cancellable context and Stop waits
// until all workers have observed cancellation before reporting quiescence.
type backgroundWorkers struct {
	mu      sync.Mutex
	workers []func(context.Context)
	cancel  context.CancelFunc
	done    chan struct{}
}

func newBackgroundWorkers(workers ...func(context.Context)) *backgroundWorkers {
	return &backgroundWorkers{workers: workers}
}

func (b *backgroundWorkers) Start(context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cancel != nil {
		return errors.New("background workers are already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	b.cancel = cancel
	b.done = done
	go func() {
		var wg sync.WaitGroup
		for _, worker := range b.workers {
			worker := worker
			wg.Add(1)
			go func() {
				defer wg.Done()
				worker(ctx)
			}()
		}
		wg.Wait()
		close(done)
	}()
	return nil
}

func (b *backgroundWorkers) Stop(ctx context.Context) error {
	b.mu.Lock()
	cancel, done := b.cancel, b.done
	b.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	select {
	case <-done:
		b.mu.Lock()
		if b.done == done {
			b.cancel = nil
			b.done = nil
		}
		b.mu.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
