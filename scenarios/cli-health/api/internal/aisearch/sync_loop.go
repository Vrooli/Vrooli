package aisearch

import (
	"context"
	"errors"
	"log"
	"runtime/debug"
	"time"
)

// SyncLoop drives Reconciler.RunOnce on a configurable interval.
type SyncLoop struct {
	Reconciler *Reconciler
	Interval   time.Duration
	Disabled   bool
	Clock      func() time.Time
}

// NewSyncLoop constructs a SyncLoop with config from the environment.
func NewSyncLoop(r *Reconciler) *SyncLoop {
	cfg := LoadConfigFromEnv()
	return &SyncLoop{
		Reconciler: r,
		Interval:   cfg.SyncInterval,
		Disabled:   cfg.SyncDisabled,
		Clock:      time.Now,
	}
}

// RunOnce delegates to the Reconciler. ErrReconcileBusy is a no-op success.
func (s *SyncLoop) RunOnce(ctx context.Context) (*DriftReport, *ApplyResult, error) {
	plan, apply, err := s.Reconciler.RunOnce(ctx)
	if errors.Is(err, ErrReconcileBusy) {
		return plan, apply, nil
	}
	return plan, apply, err
}

// Start drives a periodic loop until ctx is canceled.
func (s *SyncLoop) Start(ctx context.Context) {
	if s == nil || s.Reconciler == nil {
		log.Printf("[cli-health/aisearch] sync_loop disabled (no reconciler)")
		return
	}
	if s.Disabled {
		log.Printf("[cli-health/aisearch] sync_loop disabled via %s", EnvSyncDisabled)
		return
	}
	if s.Interval <= 0 {
		log.Printf("[cli-health/aisearch] sync_loop disabled (non-positive interval)")
		return
	}

	log.Printf("[cli-health/aisearch] sync_loop enabled (interval=%s)", s.Interval)
	t := time.NewTicker(s.Interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[cli-health/aisearch] sync_loop stopping: %v", ctx.Err())
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *SyncLoop) tick(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[cli-health/aisearch] sync_loop tick panic: %v\n%s", r, debug.Stack())
		}
	}()
	plan, apply, err := s.RunOnce(ctx)
	if err != nil {
		log.Printf("[cli-health/aisearch] sync_loop tick failed: %v", err)
		return
	}
	if plan != nil && apply != nil && (plan.HasWork() || len(apply.Errors) > 0) {
		log.Printf("[cli-health/aisearch] sync_loop tick: plan=%d collections, errors=%d",
			len(plan.Collections), len(apply.Errors))
	}
}
