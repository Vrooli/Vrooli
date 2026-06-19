package exposure

import (
	"context"
	"fmt"
	"log"
	"time"
)

const DefaultReconcileInterval = 5 * time.Minute

// Scheduler runs the exposure broker's idempotent reconcile loop. One
// reconcile runs at boot, then each tick re-ensures CORE routes and reaps
// expired leases.
type Scheduler struct {
	service  Service
	interval time.Duration
	logger   *log.Logger
	ticks    <-chan time.Time
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
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	return &Scheduler{
		service:  cfg.Service,
		interval: cfg.Interval,
		logger:   cfg.Logger,
		ticks:    cfg.Ticks,
	}, nil
}

func (s *Scheduler) Run(ctx context.Context) {
	s.reconcile(ctx, "boot")
	if ctx.Err() != nil {
		return
	}

	ticks := s.ticks
	var ticker *time.Ticker
	if ticks == nil {
		ticker = time.NewTicker(s.interval)
		ticks = ticker.C
	}
	if ticker != nil {
		defer ticker.Stop()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			s.reconcile(ctx, "tick")
		}
	}
}

func (s *Scheduler) reconcile(ctx context.Context, trigger string) {
	coreEnsured, leasesReaped, err := s.service.Reconcile(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.logger.Printf("[tunnel-manager] exposure reconcile %s failed (will retry): %v", trigger, err)
		}
		return
	}
	if coreEnsured > 0 || leasesReaped > 0 {
		s.logger.Printf("[tunnel-manager] exposure reconcile %s applied core_ensured=%d leases_reaped=%d", trigger, coreEnsured, leasesReaped)
	}
}
