package aisearch

import (
	"context"
	"errors"
	"log"
	"runtime/debug"
	"time"
)

// SyncLoop drives Reconciler.RunOnce on a configurable interval. Name is the
// log prefix (the consuming scenario, e.g. "cli-health"). It is singleton-safe
// (RunOnce guards re-entry) and panic-recovering per tick.
//
// A consumer that can swap its engine in place (an index-time tuning apply that
// re-embeds with a new recipe — see the cli-health command service) must NOT bind
// a fixed *Reconciler here: after the swap the loop would keep reconciling with
// the OLD reconciler's embedder and the recipe-aware drift hash would fight the
// apply, re-embedding back to the old recipe within one interval. Such a consumer
// constructs the loop with NewSyncLoopFunc, which resolves the CURRENT reconciler
// each tick.
type SyncLoop struct {
	Reconciler *Reconciler
	// Resolve, when non-nil, supplies the reconciler to run each tick (and takes
	// precedence over Reconciler). It lets a consumer that swaps its engine in
	// place keep the loop pointed at the live reconciler. A nil return is treated
	// as "no reconciler this tick" (a no-op), so a mid-swap window is safe.
	Resolve  func() *Reconciler
	Interval time.Duration
	Disabled bool
	Name     string
	Clock    func() time.Time
}

// NewSyncLoop builds a SyncLoop bound to a fixed reconciler — the common case for
// a consumer whose engine never changes after boot (KO docs, swarm-manager).
func NewSyncLoop(name string, r *Reconciler, cfg Config) *SyncLoop {
	return &SyncLoop{
		Reconciler: r,
		Interval:   cfg.SyncInterval,
		Disabled:   cfg.SyncDisabled,
		Name:       name,
		Clock:      time.Now,
	}
}

// NewSyncLoopFunc builds a SyncLoop that resolves the current reconciler each
// tick via resolve — for a consumer that can swap its engine in place (live
// index-time tuning apply). resolve may return nil during a swap (no-op tick).
func NewSyncLoopFunc(name string, resolve func() *Reconciler, cfg Config) *SyncLoop {
	return &SyncLoop{
		Resolve:  resolve,
		Interval: cfg.SyncInterval,
		Disabled: cfg.SyncDisabled,
		Name:     name,
		Clock:    time.Now,
	}
}

// current returns the reconciler to drive this tick: the Resolve provider when
// set, else the fixed Reconciler.
func (s *SyncLoop) current() *Reconciler {
	if s.Resolve != nil {
		return s.Resolve()
	}
	return s.Reconciler
}

func (s *SyncLoop) logf(format string, args ...any) {
	name := s.Name
	if name == "" {
		name = "aisearch"
	}
	log.Printf("["+name+"/aisearch] "+format, args...)
}

// RunOnce delegates to the current Reconciler. ErrReconcileBusy is a no-op
// success; a nil reconciler (mid-swap, or never configured) is a no-op too.
func (s *SyncLoop) RunOnce(ctx context.Context) (*DriftReport, *ApplyResult, error) {
	rec := s.current()
	if rec == nil {
		return nil, nil, nil
	}
	plan, apply, err := rec.RunOnce(ctx)
	if errors.Is(err, ErrReconcileBusy) {
		return plan, apply, nil
	}
	return plan, apply, err
}

// Start drives a periodic loop until ctx is canceled.
func (s *SyncLoop) Start(ctx context.Context) {
	if s == nil || (s.Reconciler == nil && s.Resolve == nil) {
		s.logf("sync_loop disabled (no reconciler)")
		return
	}
	if s.Disabled {
		s.logf("sync_loop disabled via config")
		return
	}
	if s.Interval <= 0 {
		s.logf("sync_loop disabled (non-positive interval)")
		return
	}

	s.logf("sync_loop enabled (interval=%s)", s.Interval)
	t := time.NewTicker(s.Interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logf("sync_loop stopping: %v", ctx.Err())
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *SyncLoop) tick(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			s.logf("sync_loop tick panic: %v\n%s", r, debug.Stack())
		}
	}()
	plan, apply, err := s.RunOnce(ctx)
	if err != nil {
		s.logf("sync_loop tick failed: %v", err)
		return
	}
	if plan != nil && apply != nil && (plan.HasWork() || len(apply.Errors) > 0) {
		s.logf("sync_loop tick: plan=%d collections, errors=%d, deferred=%d",
			len(plan.Collections), len(apply.Errors), apply.Deferred)
	}
}
