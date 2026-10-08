package recovery

import (
	"context"
	"fmt"
	"log"
	"time"

	"tunnel-manager/internal/scheduler"
)

const DefaultEvaluationInterval = time.Minute

// Scheduler owns the live recovery evaluation loop. Recovery attempts still
// flow through Service.Evaluate, which applies failure thresholds, backoff,
// idempotency, and the circuit breaker before any cloudflared restart.
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
		return nil, fmt.Errorf("recovery scheduler requires service")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultEvaluationInterval
	}
	s := &Scheduler{service: cfg.Service}
	s.runner = scheduler.NewRunner(cfg.Interval, cfg.Logger, cfg.Ticks, s.evaluate)
	return s, nil
}

func (s *Scheduler) Run(ctx context.Context) {
	s.runner.Run(ctx)
}

func (s *Scheduler) evaluate(ctx context.Context, trigger string) {
	evt, acted, err := s.service.Evaluate(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.runner.Logger.Printf("[tunnel-manager] recovery scheduler %s failed (will retry): %v", trigger, err)
		}
		return
	}
	if acted {
		s.runner.Logger.Printf("[tunnel-manager] recovery scheduler %s action=%s outcome=%s attempt=%d", trigger, evt.Action, evt.Outcome, evt.Attempt)
	}
}
