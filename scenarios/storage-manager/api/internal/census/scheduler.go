package census

import (
	"context"
	"sync"
	"time"
)

// Scheduler owns periodic, read-only census observations. It deliberately
// accepts the scan operation as a callback so lifecycle wiring can provide the
// owner inventory and durable SnapshotStore without making this package aware
// of HTTP or process globals.
type Scheduler struct {
	interval time.Duration
	run      func(context.Context) error
	once     sync.Once
}

func NewScheduler(interval time.Duration, run func(context.Context) error) *Scheduler {
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	return &Scheduler{interval: interval, run: run}
}

// RunOnce is used by product acceptance and lifecycle tests. It does not
// mutate host files; the supplied operation is responsible for persisting the
// resulting immutable observation.
func (s *Scheduler) RunOnce(ctx context.Context) error {
	if s == nil || s.run == nil {
		return nil
	}
	return s.run(ctx)
}

// Start returns immediately and waits one full interval before its first run.
// A newly started storage-manager therefore never performs a surprise host
// scan during readiness, while long-lived processes still build history.
func (s *Scheduler) Start(ctx context.Context) {
	if s == nil || s.run == nil {
		return
	}
	s.once.Do(func() {
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
	})
}
