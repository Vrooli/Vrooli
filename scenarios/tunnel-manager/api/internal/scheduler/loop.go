// Package scheduler contains small lifecycle primitives shared by the
// scenario's background jobs.
package scheduler

import (
	"context"
	"time"

	"tunnel-manager/internal/logging"
)

type Action func(context.Context, string)

type Runner struct {
	Interval time.Duration
	Logger   logging.Logger
	Ticks    <-chan time.Time
	Action   Action
}

func NewRunner(interval time.Duration, logger logging.Logger, ticks <-chan time.Time, action Action) *Runner {
	if logger == nil {
		logger = logging.Default()
	}
	return &Runner{
		Interval: interval,
		Logger:   logger,
		Ticks:    ticks,
		Action:   action,
	}
}

func (r *Runner) Run(ctx context.Context) {
	Loop(ctx, r.Interval, r.Ticks, r.Action)
}

// Loop runs action once at boot and then once per tick until ctx is canceled.
// Production leaves ticks nil so Loop owns a ticker; tests inject ticks for
// deterministic advancement.
func Loop(ctx context.Context, interval time.Duration, ticks <-chan time.Time, action Action) {
	action(ctx, "boot")
	if ctx.Err() != nil {
		return
	}

	var ticker *time.Ticker
	if ticks == nil {
		ticker = time.NewTicker(interval)
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
			action(ctx, "tick")
		}
	}
}
