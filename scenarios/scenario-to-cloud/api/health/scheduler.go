package health

import (
	"context"
	"log/slog"
	"time"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
)

// Scheduler periodically asks the lifecycle owner for deployments that need
// observation, then invokes the same typed observation path used by callers.
// The observer owns transition detection and delivery; the scheduler only
// supplies cadence and does not infer health from transport success.
type Scheduler struct {
	Interval time.Duration
	Targets  func(context.Context) ([]string, error)
	Observe  func(context.Context, string) (*healthv1.HealthObservation, error)
	Logger   *slog.Logger
}

// Run observes configured targets on a fixed cadence until ctx is cancelled.
// It is deliberately serial so one slow/unreachable host cannot create an
// unbounded fan-out of management operations.
func (s Scheduler) Run(ctx context.Context) {
	if s.Interval <= 0 || s.Targets == nil || s.Observe == nil {
		return
	}
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	s.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

// RunOnce performs one scheduled observation pass. It is exported so a
// governed scheduler can invoke the same operation without waiting on a
// process-local ticker.
func (s Scheduler) RunOnce(ctx context.Context) {
	if s.Interval <= 0 || s.Targets == nil || s.Observe == nil {
		return
	}
	s.runOnce(ctx)
}

func (s Scheduler) runOnce(ctx context.Context) {
	ids, err := s.Targets(ctx)
	if err != nil {
		s.log("scheduled health target discovery failed", "error", err)
		return
	}
	for _, id := range ids {
		if ctx.Err() != nil {
			return
		}
		if id == "" {
			continue
		}
		observeCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		_, err := s.Observe(observeCtx, id)
		cancel()
		if err != nil {
			s.log("scheduled health observation failed", "deployment_id", id, "error", err)
		}
	}
}

func (s Scheduler) log(message string, args ...any) {
	if s.Logger != nil {
		s.Logger.Error(message, args...)
	}
}
