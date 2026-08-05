package retention

import (
	"context"
	"time"
)

// Scheduler periodically runs the control-plane retention adapter. It waits
// one interval before its first cycle so service readiness never causes a
// surprise reclaim operation; operators can invoke the retention endpoint for
// an explicit immediate cycle.
type Scheduler struct {
	interval time.Duration
	run      func(context.Context) error
}

func NewScheduler(interval time.Duration, run func(context.Context) error) *Scheduler {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	return &Scheduler{interval: interval, run: run}
}

func (s *Scheduler) Start(ctx context.Context) {
	if s == nil || s.run == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = s.run(ctx)
			}
		}
	}()
}
