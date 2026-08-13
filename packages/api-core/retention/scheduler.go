package retention

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// DefaultInterval is the cycle period used when a scheduler is configured
// without one. It is deliberately short relative to any realistic age horizon:
// the cost of a cycle that finds nothing to do is one measurement, and the cost
// of a cycle that runs too rarely is a target that sits over budget for hours.
const DefaultInterval = 15 * time.Minute

// DefaultCycleDuration is the wall-clock allowance for one scheduled cycle. A
// cycle that hits it stops cleanly, reports Incomplete, and resumes on the next
// tick, so a table needing hours of deletes converges across many cycles instead
// of monopolizing the write lock in one.
const DefaultCycleDuration = 2 * time.Minute

// DefaultBatchPause is how long a scheduled cycle waits between delete batches,
// leaving the database reachable by the component that owns it.
//
// It is sized against what it protects rather than against throughput. A health
// probe measured in low hundreds of milliseconds needs a gap of the same order
// to land in; 25ms between batches costs a cycle a few percent of its wall clock
// and is the difference between a prune the rest of the process notices and one
// it does not.
const DefaultBatchPause = 25 * time.Millisecond

// Timer is the subset of time.Timer the scheduler needs, so tests can drive
// cycles without sleeping.
type Timer = schedule.Timer

// Clock supplies the current time and timers. Injecting it is what makes the
// age-bound behavior testable without waiting out a real horizon.
type Clock = schedule.Clock

// realClock is the production Clock.
// SystemClock is the production Clock implementation.
func SystemClock() Clock { return schedule.System() }

// SchedulerConfig configures a retention scheduler.
type SchedulerConfig struct {
	// Engine enforces the budgets each cycle. Required.
	Engine *Engine
	// Interval is the period between cycles. Defaults to DefaultInterval.
	Interval time.Duration
	// Clock supplies time and timers. Defaults to SystemClock.
	Clock Clock
	// RunOnStart runs one cycle immediately instead of waiting out the first
	// interval. A process that has been down long enough to accumulate a
	// backlog should not wait another interval before acting on it.
	RunOnStart bool
	// OnCycle, when set, receives every cycle's results and error. A cycle
	// error never stops the scheduler: a transient failure to open a database
	// must not silently end retention for the life of the process.
	OnCycle func(results []Result, err error)
}

// Scheduler runs an engine on an interval until its context is cancelled.
type Scheduler struct {
	engine     *Engine
	interval   time.Duration
	clock      Clock
	runOnStart bool
	onCycle    func([]Result, error)
	done       chan struct{}
}

// NewScheduler validates cfg and returns a scheduler.
func NewScheduler(cfg SchedulerConfig) (*Scheduler, error) {
	if cfg.Engine == nil {
		return nil, fmt.Errorf("retention scheduler: engine is required")
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	clock := cfg.Clock
	if clock == nil {
		clock = SystemClock()
	}
	return &Scheduler{
		engine:     cfg.Engine,
		interval:   interval,
		clock:      clock,
		runOnStart: cfg.RunOnStart,
		onCycle:    cfg.OnCycle,
		done:       make(chan struct{}),
	}, nil
}

// Interval reports the configured cycle period.
func (s *Scheduler) Interval() time.Duration { return s.interval }

// Run drives cycles until ctx is cancelled, then returns. It blocks; callers
// wanting it in the background start it in a goroutine and wait on Done.
func (s *Scheduler) Run(ctx context.Context) {
	defer close(s.done)

	if s.runOnStart {
		s.cycle(ctx)
	}
	for {
		timer := s.clock.NewTimer(s.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C():
			if ctx.Err() != nil {
				return
			}
			s.cycle(ctx)
		}
	}
}

// Done closes once Run has returned, so shutdown can wait for an in-flight cycle
// instead of racing it.
func (s *Scheduler) Done() <-chan struct{} { return s.done }

func (s *Scheduler) cycle(ctx context.Context) {
	results, err := s.engine.Run(ctx)
	if s.onCycle != nil {
		s.onCycle(results, err)
	}
}
