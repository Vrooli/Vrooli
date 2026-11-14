package audit

import (
	"context"
	"sync/atomic"
)

// RunLimiter bounds concurrent audit/validation runs for the whole process.
type RunLimiter struct {
	sem    chan struct{}
	active atomic.Int64
	queued atomic.Int64
}

// NewRunLimiter constructs a limiter. Values below one are treated as one.
func NewRunLimiter(max int) *RunLimiter {
	if max < 1 {
		max = 1
	}
	return &RunLimiter{sem: make(chan struct{}, max)}
}

// Acquire waits for one run slot and respects context cancellation while queued.
func (l *RunLimiter) Acquire(ctx context.Context) (func(), error) {
	if l == nil {
		return func() {}, nil
	}
	l.queued.Add(1)
	select {
	case l.sem <- struct{}{}:
		l.queued.Add(-1)
		l.active.Add(1)
		return func() {
			<-l.sem
			l.active.Add(-1)
		}, nil
	case <-ctx.Done():
		l.queued.Add(-1)
		return nil, ctx.Err()
	}
}

// Active returns the number of currently running audits.
func (l *RunLimiter) Active() int {
	if l == nil {
		return 0
	}
	return int(l.active.Load())
}

// Queued returns the number of audits waiting for a slot.
func (l *RunLimiter) Queued() int {
	if l == nil {
		return 0
	}
	return int(l.queued.Load())
}
