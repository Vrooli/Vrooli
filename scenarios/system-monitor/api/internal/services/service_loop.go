package services

import (
	"context"
	"sync"
)

// serviceLoop owns the start, stop, and join lifecycle shared by background
// services. At most one generation may run; a restart is accepted after the
// previous generation has returned.
type serviceLoop struct {
	mu         sync.Mutex
	state      serviceLoopState
	generation *serviceLoopGeneration
}

type serviceLoopState uint8

const (
	serviceLoopStopped serviceLoopState = iota
	serviceLoopRunning
	serviceLoopStopping
)

type serviceLoopGeneration struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// Start launches run once. It returns false if a generation is already
// running or stopping.
func (l *serviceLoop) Start(run func(context.Context)) bool {
	l.mu.Lock()
	if l.state != serviceLoopStopped {
		l.mu.Unlock()
		return false
	}

	ctx, cancel := context.WithCancel(context.Background())
	generation := &serviceLoopGeneration{cancel: cancel, done: make(chan struct{})}
	l.generation = generation
	l.state = serviceLoopRunning
	l.mu.Unlock()

	go func() {
		defer func() {
			l.mu.Lock()
			if l.generation == generation {
				l.generation = nil
				l.state = serviceLoopStopped
			}
			close(generation.done)
			l.mu.Unlock()
		}()
		run(ctx)
	}()
	return true
}

// Stop cancels the active generation and joins it. If ctx expires, the
// generation remains in the stopping state and a later Stop can join it.
func (l *serviceLoop) Stop(ctx context.Context) error {
	l.mu.Lock()
	generation := l.generation
	if generation == nil {
		l.mu.Unlock()
		return nil
	}
	if l.state == serviceLoopRunning {
		l.state = serviceLoopStopping
	}
	generation.cancel()
	l.mu.Unlock()

	select {
	case <-generation.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
