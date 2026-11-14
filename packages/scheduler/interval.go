// Package scheduler contains the shared wait-from-end primitive for periodic
// work. A ticker is wrong for filesystem walks: it buffers a tick while a walk
// is still running and can leave the host under continuous metadata load.
package scheduler

import (
	"context"
	"sync"
	"time"
)

type Cycle struct {
	StartedAt time.Time
	Duration  time.Duration
	Err       error
	Overran   bool
}
type Stats struct {
	Cycles, Overruns int64
	Last             Cycle
}
type Runner struct {
	interval time.Duration
	run      func(context.Context) error
	observe  func(Cycle)
	once     sync.Once
	mu       sync.Mutex
	stats    Stats
}

func New(interval time.Duration, run func(context.Context) error) *Runner {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Runner{interval: interval, run: run}
}

func (r *Runner) Observe(fn func(Cycle)) *Runner {
	if r != nil {
		r.observe = fn
	}
	return r
}

// WithObserver is the domain-neutral spelling retained for callers that use
// the scheduler as a fluent constructor.
func (r *Runner) WithObserver(fn func(Cycle)) *Runner { return r.Observe(fn) }

func (r *Runner) Stats() Stats {
	if r == nil {
		return Stats{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stats
}

func (r *Runner) Start(ctx context.Context) {
	if r == nil || r.run == nil {
		return
	}
	r.once.Do(func() { go r.loop(ctx) })
}

// RunOnce executes one cycle synchronously. It is intentionally separate from
// Start so product acceptance and lifecycle tests can exercise the exact
// operation without waiting for the first interval.
func (r *Runner) RunOnce(ctx context.Context) error {
	if r == nil || r.run == nil {
		return nil
	}
	return r.run(ctx)
}

func (r *Runner) loop(ctx context.Context) {
	timer := time.NewTimer(r.interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		start := time.Now()
		err := r.run(ctx)
		c := Cycle{StartedAt: start, Duration: time.Since(start), Err: err}
		c.Overran = c.Duration >= r.interval
		r.mu.Lock()
		r.stats.Cycles++
		if c.Overran {
			r.stats.Overruns++
		}
		r.stats.Last = c
		r.mu.Unlock()
		if r.observe != nil {
			r.observe(c)
		}
		if ctx.Err() != nil {
			return
		}
		timer.Reset(r.interval)
	}
}
