package exposure

import (
	"context"
	"fmt"
	"log"
	"time"

	"tunnel-manager/internal/scheduler"
)

const DefaultReconcileInterval = 5 * time.Minute

// Scheduler runs the exposure broker's idempotent reconcile loop. One
// reconcile runs at boot, then each tick re-ensures CORE routes and reaps
// expired leases.
type Scheduler struct {
	service Service
	runner  *scheduler.Runner
}

type SchedulerConfig struct {
	Service  Service
	Interval time.Duration
	Logger   *log.Logger

	// Ticks is a test seam. Production leaves it nil and the scheduler owns a
	// time.Ticker; tests inject a channel to advance the loop without sleeping.
	Ticks <-chan time.Time
}

func NewScheduler(cfg SchedulerConfig) (*Scheduler, error) {
	if cfg.Service == nil {
		return nil, fmt.Errorf("exposure scheduler requires service")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultReconcileInterval
	}
	s := &Scheduler{service: cfg.Service}
	s.runner = scheduler.NewRunner(cfg.Interval, cfg.Logger, cfg.Ticks, s.reconcile)
	return s, nil
}

func (s *Scheduler) Run(ctx context.Context) {
	s.runner.Run(ctx)
}

func (s *Scheduler) reconcile(ctx context.Context, trigger string) {
	coreEnsured, leasesReaped, err := s.service.Reconcile(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.runner.Logger.Printf("[tunnel-manager] exposure reconcile %s failed (will retry): %v", trigger, err)
		}
		return
	}
	if coreEnsured > 0 || leasesReaped > 0 {
		s.runner.Logger.Printf("[tunnel-manager] exposure reconcile %s applied core_ensured=%d leases_reaped=%d", trigger, coreEnsured, leasesReaped)
	}
}
