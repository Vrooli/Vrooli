package probes

import (
	"context"
	"fmt"
	"log"
	"time"
)

const DefaultProbeInterval = time.Minute

// Scheduler runs continuous diagnostics by executing one probe cycle at
// boot and then on each tick. It owns no persistence itself; the Service
// remains the single probe policy and storage boundary.
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
		return nil, fmt.Errorf("probes scheduler requires service")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultProbeInterval
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
	s.runProbes(ctx, "boot")
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
			s.runProbes(ctx, "tick")
		}
	}
}

func (s *Scheduler) runProbes(ctx context.Context, trigger string) {
	results, err := s.service.RunProbes(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.logger.Printf("[tunnel-manager] probe scheduler %s failed (will retry): %v", trigger, err)
		}
		return
	}
	if len(results) > 0 {
		s.logger.Printf("[tunnel-manager] probe scheduler %s recorded results=%d", trigger, len(results))
	}
}
