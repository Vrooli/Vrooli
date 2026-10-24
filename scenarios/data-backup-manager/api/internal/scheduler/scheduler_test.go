package scheduler_test

import (
	"context"
	"testing"
	"time"

	"data-backup-manager/internal/scheduler"

	"github.com/vrooli/api-core/scheduletest"
)

// fakePlanSource is a simple in-test stub that returns a fixed set of plans.
type fakePlanSource struct {
	plans []scheduler.DuePlan
}

func (f *fakePlanSource) SchedulablePlans(_ context.Context) ([]scheduler.DuePlan, error) {
	return f.plans, nil
}

// fakeRunTrigger records all planIDs it is called with, in order.
type fakeRunTrigger struct {
	triggered []string
}

func (f *fakeRunTrigger) TriggerRun(_ context.Context, planID string) error {
	f.triggered = append(f.triggered, planID)
	return nil
}

// TestScheduler_FiresAndManualTrigger is the primary acceptance test:
//
//   - Tick at t0 fires the plan once.
//   - Advance 30m, Tick → no new fire (interval not elapsed).
//   - Advance to t0+1h, Tick → fires again.
//   - TriggerManual → fires immediately regardless of schedule.
func TestScheduler_FiresAndManualTrigger(t *testing.T) {
	ctx := context.Background()

	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := scheduletest.New(t0)

	source := &fakePlanSource{
		plans: []scheduler.DuePlan{
			{ID: "plan-abc", Schedule: "1h", Enabled: true},
		},
	}
	trigger := &fakeRunTrigger{}

	sched := scheduler.New(clk, source, trigger)

	// --- Tick at t0: should fire once ---
	if err := sched.Tick(ctx); err != nil {
		t.Fatalf("Tick at t0: %v", err)
	}
	if len(trigger.triggered) != 1 {
		t.Fatalf("trigger count after t0 Tick = %d, want 1", len(trigger.triggered))
	}
	if trigger.triggered[0] != "plan-abc" {
		t.Fatalf("triggered planID = %q, want plan-abc", trigger.triggered[0])
	}

	// --- Advance 30m, Tick: interval not elapsed, no new fire ---
	clk.Advance(30 * time.Minute)
	if err := sched.Tick(ctx); err != nil {
		t.Fatalf("Tick at t0+30m: %v", err)
	}
	if len(trigger.triggered) != 1 {
		t.Fatalf("trigger count after t0+30m Tick = %d, want still 1", len(trigger.triggered))
	}

	// --- Advance to t0+1h, Tick: interval elapsed, fires again ---
	clk.Advance(30 * time.Minute) // now at t0+1h
	if err := sched.Tick(ctx); err != nil {
		t.Fatalf("Tick at t0+1h: %v", err)
	}
	if len(trigger.triggered) != 2 {
		t.Fatalf("trigger count after t0+1h Tick = %d, want 2", len(trigger.triggered))
	}
	if trigger.triggered[1] != "plan-abc" {
		t.Fatalf("second triggered planID = %q, want plan-abc", trigger.triggered[1])
	}

	// --- TriggerManual: fires immediately ---
	if err := sched.TriggerManual(ctx, "plan-abc"); err != nil {
		t.Fatalf("TriggerManual: %v", err)
	}
	if len(trigger.triggered) != 3 {
		t.Fatalf("trigger count after TriggerManual = %d, want 3", len(trigger.triggered))
	}
	if trigger.triggered[2] != "plan-abc" {
		t.Fatalf("manual triggered planID = %q, want plan-abc", trigger.triggered[2])
	}
}

// TestScheduler_NextFire covers the next-scheduled-fire projection that powers
// TargetStatus.next_scheduled_at: unknown before the first fire, lastFire+
// interval after, and never-predicted for an empty/unparseable schedule.
func TestScheduler_NextFire(t *testing.T) {
	ctx := context.Background()
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clk := scheduletest.New(t0)
	source := &fakePlanSource{plans: []scheduler.DuePlan{{ID: "plan-abc", Schedule: "1h", Enabled: true}}}
	sched := scheduler.New(clk, source, &fakeRunTrigger{})

	// Before any fire this lifetime: no durable basis to predict.
	if _, ok := sched.NextFire("plan-abc", "1h"); ok {
		t.Fatal("NextFire before first fire must report ok=false")
	}
	// Empty / unparseable schedule: never predicted.
	if _, ok := sched.NextFire("plan-abc", ""); ok {
		t.Fatal("NextFire with empty schedule must be ok=false")
	}
	if _, ok := sched.NextFire("plan-abc", "nonsense"); ok {
		t.Fatal("NextFire with unparseable schedule must be ok=false")
	}

	// After a fire at t0, next fire is t0 + interval.
	if err := sched.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	next, ok := sched.NextFire("plan-abc", "1h")
	if !ok {
		t.Fatal("NextFire after a fire must report ok=true")
	}
	if want := t0.Add(time.Hour); !next.Equal(want) {
		t.Fatalf("NextFire = %v, want %v (lastFire + interval)", next, want)
	}
}

// TestScheduler_DisabledPlanSkipped proves disabled plans are never auto-fired.
func TestScheduler_DisabledPlanSkipped(t *testing.T) {
	ctx := context.Background()
	clk := scheduletest.New(time.Time{})
	source := &fakePlanSource{
		plans: []scheduler.DuePlan{
			{ID: "plan-off", Schedule: "1h", Enabled: false},
		},
	}
	trigger := &fakeRunTrigger{}
	sched := scheduler.New(clk, source, trigger)

	if err := sched.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if len(trigger.triggered) != 0 {
		t.Fatalf("disabled plan triggered: %v", trigger.triggered)
	}
}

// TestScheduler_EmptyScheduleSkipped proves manual-only plans are skipped by Tick.
func TestScheduler_EmptyScheduleSkipped(t *testing.T) {
	ctx := context.Background()
	clk := scheduletest.New(time.Time{})
	source := &fakePlanSource{
		plans: []scheduler.DuePlan{
			{ID: "plan-manual", Schedule: "", Enabled: true},
		},
	}
	trigger := &fakeRunTrigger{}
	sched := scheduler.New(clk, source, trigger)

	if err := sched.Tick(ctx); err != nil {
		t.Fatalf("Tick: %v", err)
	}
	if len(trigger.triggered) != 0 {
		t.Fatalf("empty-schedule plan auto-triggered: %v", trigger.triggered)
	}
}
