// Package scheduler is the in-process backup scheduler. It wakes on each Tick,
// reads SchedulablePlans from a PlanSource, and fires RunTrigger for any plan
// whose interval has elapsed since its last fire.
//
// The scheduler owns no domain knowledge about plans or runs concretely; it
// depends only on the two narrow seams declared below. The runs service will
// satisfy RunTrigger; the plans service will satisfy PlanSource.
//
// Schedule format for v1: a Go duration string parsed by time.ParseDuration
// (e.g. "1h", "30m"). An empty schedule means manual-only — the plan never
// auto-fires but TriggerManual still works.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// RunTrigger fires a run for a given plan id. The runs service implements this.
//
// seam: RunTrigger is the interface the scheduler uses to start a run; the runs
// domain satisfies it without the scheduler importing runs concretely.
type RunTrigger interface {
	TriggerRun(ctx context.Context, planID string) error
}

// DuePlan is the narrow projection of a plan that the scheduler needs to decide
// whether to fire.
type DuePlan struct {
	ID       string
	Schedule string
	Enabled  bool
}

// PlanSource lists schedulable plans for the scheduler. The plans service
// satisfies this via plans.Service.SchedulablePlans.
//
// seam: PlanSource is the interface the scheduler uses to fetch plans; the
// plans domain satisfies it without the scheduler importing plans concretely.
type PlanSource interface {
	SchedulablePlans(ctx context.Context) ([]DuePlan, error)
}

// Scheduler is the in-process backup scheduler.
type Scheduler struct {
	clk     schedule.Clock
	source  PlanSource
	trigger RunTrigger

	mu       sync.Mutex
	lastFire map[string]time.Time // plan id → last time TriggerRun was called
}

// New constructs a Scheduler.
func New(clk schedule.Clock, source PlanSource, trigger RunTrigger) *Scheduler {
	return &Scheduler{
		clk:      clk,
		source:   source,
		trigger:  trigger,
		lastFire: make(map[string]time.Time),
	}
}

// Tick reads SchedulablePlans and fires RunTrigger for each plan whose schedule
// interval has elapsed. It is safe to call concurrently but typical use is a
// single goroutine looping on a wall-clock ticker.
func (s *Scheduler) Tick(ctx context.Context) error {
	plans, err := s.source.SchedulablePlans(ctx)
	if err != nil {
		return fmt.Errorf("scheduler: list schedulable plans: %w", err)
	}

	now := s.clk.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, p := range plans {
		if !p.Enabled {
			continue
		}
		if p.Schedule == "" {
			continue
		}
		interval, err := time.ParseDuration(p.Schedule)
		if err != nil {
			// Unparseable schedule: skip silently; callers validate on write.
			continue
		}

		last, fired := s.lastFire[p.ID]
		if !fired || now.Sub(last) >= interval {
			if trigErr := s.trigger.TriggerRun(ctx, p.ID); trigErr != nil {
				// Best-effort: continue to next plan even if one trigger fails.
				continue
			}
			s.lastFire[p.ID] = now
		}
	}
	return nil
}

// NextFire reports the next scheduled fire time for planID, computed as its
// last fire plus the schedule interval. ok is false when the schedule is empty
// or unparseable, or when the plan has not fired during this process's lifetime
// — the lastFire history is in-memory and reset by a restart, so there is no
// durable basis to predict the next fire until the plan fires once (after which
// the next Tick will have recorded it). Callers pass the plan's schedule string
// (they already hold it) so the scheduler need not re-resolve the plan.
func (s *Scheduler) NextFire(planID, schedule string) (time.Time, bool) {
	interval, err := time.ParseDuration(schedule)
	if err != nil || interval <= 0 {
		return time.Time{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	last, fired := s.lastFire[planID]
	if !fired {
		return time.Time{}, false
	}
	return last.Add(interval), true
}

// TriggerManual calls RunTrigger directly for planID, bypassing the schedule.
// It records the fire time so the next Tick sees the updated lastFire.
func (s *Scheduler) TriggerManual(ctx context.Context, planID string) error {
	if err := s.trigger.TriggerRun(ctx, planID); err != nil {
		return fmt.Errorf("scheduler: manual trigger %q: %w", planID, err)
	}
	s.mu.Lock()
	s.lastFire[planID] = s.clk.Now()
	s.mu.Unlock()
	return nil
}
