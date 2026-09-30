package census

// The census scheduler waits from the end of a pass, never from a ticker,
// and additionally caps the share of wall time a census may consume. A
// filesystem census can outlive its nominal interval; waiting one interval
// after an 85-minute pass on a 30-minute interval still left the host under
// a metadata walk three quarters of the time. The duty-cycle cap makes the
// idle gap grow with the pass: at the default 10% cap a pass of duration d
// is followed by at least 9d of quiet.
//
// IncrementalScanner is deliberately not used here. Its change signature is
// the root directory's own mtime, which only moves when a direct child is
// created or removed, so a multi-root census over a repository and runtime
// homes would reuse a stale report indefinitely while deep trees change.

import (
	"context"
	"sync"
	"time"
)

// DefaultInterval is the scheduled census period when STORAGE_CENSUS_INTERVAL
// is unset. Six hours keeps six samples per 36 hours for the growth fit while
// leaving a multi-million-inode host idle between walks.
const DefaultInterval = 6 * time.Hour

// DefaultDutyCycle is the maximum fraction of wall time a scheduled census
// may spend walking. Values are clamped to (0, 1].
const DefaultDutyCycle = 0.10

type Cycle struct {
	StartedAt time.Time
	Duration  time.Duration
	Err       error
	// Overran is true when the pass took at least one interval.
	Overran bool
	// IdleFor is the quiet gap the scheduler chose before the next pass. It
	// is the interval unless the duty-cycle cap demanded more.
	IdleFor time.Duration
}

type Stats struct {
	Cycles, Overruns int64
	Last             Cycle
}

type Scheduler struct {
	interval  time.Duration
	dutyCycle float64
	run       func(context.Context) error
	observe   func(Cycle)
	once      sync.Once
	mu        sync.Mutex
	stats     Stats
}

func NewScheduler(interval time.Duration, run func(context.Context) error) *Scheduler {
	if interval <= 0 {
		interval = time.Minute
	}
	return &Scheduler{interval: interval, dutyCycle: DefaultDutyCycle, run: run}
}

// WithDutyCycle bounds census work to the given fraction of wall time.
func (s *Scheduler) WithDutyCycle(fraction float64) *Scheduler {
	if s != nil {
		s.dutyCycle = clampDutyCycle(fraction)
	}
	return s
}

func (s *Scheduler) Observe(fn func(Cycle)) *Scheduler {
	if s != nil {
		s.observe = fn
	}
	return s
}

// WithObserver is the fluent spelling retained for existing callers.
func (s *Scheduler) WithObserver(fn func(Cycle)) *Scheduler { return s.Observe(fn) }

func (s *Scheduler) Stats() Stats {
	if s == nil {
		return Stats{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats
}

func (s *Scheduler) Start(ctx context.Context) {
	if s == nil || s.run == nil {
		return
	}
	s.once.Do(func() { go s.loop(ctx) })
}

// RunOnce executes one cycle synchronously. It is intentionally separate from
// Start so product acceptance and lifecycle tests can exercise the exact
// operation without waiting for the first interval.
func (s *Scheduler) RunOnce(ctx context.Context) error {
	if s == nil || s.run == nil {
		return nil
	}
	return s.run(ctx)
}

func (s *Scheduler) loop(ctx context.Context) {
	timer := time.NewTimer(s.interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		start := time.Now()
		err := s.run(ctx)
		c := Cycle{StartedAt: start, Duration: time.Since(start), Err: err}
		c.Overran = c.Duration >= s.interval
		c.IdleFor = IdleAfter(c.Duration, s.interval, s.dutyCycle)
		s.mu.Lock()
		s.stats.Cycles++
		if c.Overran {
			s.stats.Overruns++
		}
		s.stats.Last = c
		s.mu.Unlock()
		if s.observe != nil {
			s.observe(c)
		}
		if ctx.Err() != nil {
			return
		}
		timer.Reset(c.IdleFor)
	}
}

// IdleAfter returns the quiet gap that must follow a pass of the given
// duration: at least one interval, and at least enough that the pass is no
// more than dutyCycle of the pass-plus-gap window.
func IdleAfter(duration, interval time.Duration, dutyCycle float64) time.Duration {
	dutyCycle = clampDutyCycle(dutyCycle)
	idle := interval
	if duration > 0 && dutyCycle < 1 {
		required := time.Duration(float64(duration) * (1 - dutyCycle) / dutyCycle)
		if required > idle {
			idle = required
		}
	}
	return idle
}

func clampDutyCycle(fraction float64) float64 {
	if fraction <= 0 || fraction != fraction {
		return DefaultDutyCycle
	}
	if fraction > 1 {
		return 1
	}
	return fraction
}
