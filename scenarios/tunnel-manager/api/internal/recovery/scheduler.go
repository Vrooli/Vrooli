package recovery

import (
	"context"
	"fmt"
	"log"
	"time"
)

const DefaultEvaluationInterval = time.Minute

// Scheduler owns the live recovery evaluation loop. Recovery attempts still
// flow through Service.Evaluate, which applies failure thresholds, backoff,
// idempotency, and the circuit breaker before any cloudflared restart.
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
		return nil, fmt.Errorf("recovery scheduler requires service")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultEvaluationInterval
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
	s.evaluate(ctx, "boot")
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
			s.evaluate(ctx, "tick")
		}
	}
}

func (s *Scheduler) evaluate(ctx context.Context, trigger string) {
	evt, acted, err := s.service.Evaluate(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.logger.Printf("[tunnel-manager] recovery scheduler %s failed (will retry): %v", trigger, err)
		}
		return
	}
	if acted {
		s.logger.Printf("[tunnel-manager] recovery scheduler %s action=%s outcome=%s attempt=%d", trigger, evt.Action, evt.Outcome, evt.Attempt)
	}
}
