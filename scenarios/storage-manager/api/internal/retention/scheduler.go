package retention

import (
	"context"
	"time"

	sharedscheduler "github.com/vrooli/vrooli/packages/scheduler"
)

// Scheduler periodically runs the control-plane retention adapter. It waits
// one interval before its first cycle so service readiness never causes a
// surprise reclaim operation; operators can invoke the retention endpoint for
// an explicit immediate cycle.
type Scheduler struct {
	runner *sharedscheduler.Runner
}

func NewScheduler(interval time.Duration, run func(context.Context) error) *Scheduler {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return &Scheduler{runner: sharedscheduler.New(interval, run)}
}

func (s *Scheduler) WithObserver(observe func(sharedscheduler.Cycle)) *Scheduler {
	if s != nil {
		s.runner.Observe(observe)
	}
	return s
}

func (s *Scheduler) Start(ctx context.Context) {
	if s == nil || s.runner == nil {
		return
	}
	s.runner.Start(ctx)
}
