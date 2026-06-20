package probes

import (
	"context"
	"fmt"
	"log"
	"time"

	"tunnel-manager/internal/scheduler"
)

const DefaultProbeInterval = time.Minute

// Scheduler runs continuous diagnostics by executing one probe cycle at
// boot and then on each tick. It owns no persistence itself; the Service
// remains the single probe policy and storage boundary.
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
		return nil, fmt.Errorf("probes scheduler requires service")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultProbeInterval
	}
	s := &Scheduler{service: cfg.Service}
	s.runner = scheduler.NewRunner(cfg.Interval, cfg.Logger, cfg.Ticks, s.runProbes)
	return s, nil
}

func (s *Scheduler) Run(ctx context.Context) {
	s.runner.Run(ctx)
}

func (s *Scheduler) runProbes(ctx context.Context, trigger string) {
	results, err := s.service.RunProbes(ctx)
	if err != nil {
		if ctx.Err() == nil {
			s.runner.Logger.Printf("[tunnel-manager] probe scheduler %s failed (will retry): %v", trigger, err)
		}
		return
	}
	if len(results) > 0 {
		s.runner.Logger.Printf("[tunnel-manager] probe scheduler %s recorded results=%d", trigger, len(results))
	}
}
