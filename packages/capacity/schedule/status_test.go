package schedule

import (
	"context"
	"testing"
	"time"
)

func TestStatusReportsQueueReasonsAndBudget(t *testing.T) {
	h := newGrantHarness(t, Policy{
		Mode:                 ModeBalanced,
		Location:             time.UTC,
		MaxSecondsPerHour:    100,
		MaxJobsPerHour:       10,
		MaxModelLoadsPerHour: 5,
	})
	oldest := testEpoch.Add(-5 * time.Minute)
	oldestNotBefore := oldest
	h.submit(t,
		jobSpec("embed-a", "embed", "alpha", func(j *Job) { j.NotBefore = oldestNotBefore }),
		jobSpec("embed-b", "embed", "beta", func(j *Job) { j.ModelAffinity = "winnow:e4b" }),
		jobSpec("annotate", "annotate", "alpha", func(j *Job) { j.ModelAffinity = "winnow:e4b" }),
	)
	if _, err := h.store.Defer(context.Background(), DeferOptions{
		JobID:     "embed-b",
		Reason:    "waiting for the annotation window",
		NotBefore: testEpoch.Add(time.Hour),
	}); err != nil {
		t.Fatalf("defer: %v", err)
	}

	status := h.status(t)
	if status.Mode != ModeBalanced {
		t.Fatalf("mode = %q", status.Mode)
	}
	if status.Queue.Depth != 3 {
		t.Fatalf("depth = %d, want 3 waiting jobs", status.Queue.Depth)
	}
	if status.Queue.ByKind["embed"] != 2 || status.Queue.ByKind["annotate"] != 1 {
		t.Fatalf("by kind = %v", status.Queue.ByKind)
	}
	if status.Queue.ByModel["nomic-embed-text"] != 1 || status.Queue.ByModel["winnow:e4b"] != 2 {
		t.Fatalf("by model = %v", status.Queue.ByModel)
	}
	if status.Queue.ByOwner["alpha"] != 2 || status.Queue.ByOwner["beta"] != 1 {
		t.Fatalf("by owner = %v", status.Queue.ByOwner)
	}
	if status.Queue.OldestDue == nil || !status.Queue.OldestDue.Equal(oldest) {
		t.Fatalf("oldest due = %v, want %s", status.Queue.OldestDue, oldest)
	}
	if len(status.Deferrals) != 1 || status.Deferrals[0].Count != 1 {
		t.Fatalf("deferrals = %+v, want the window reason", status.Deferrals)
	}
	if status.Deferrals[0].Reason != "waiting for the annotation window" {
		t.Fatalf("deferral reason = %q", status.Deferrals[0].Reason)
	}
	if status.Budget.SecondsRemaining != 100 || status.Budget.JobsRemaining != 10 || status.Budget.ModelLoadsRemaining != 5 {
		t.Fatalf("budget = %+v, want full budget before any run", status.Budget)
	}

	if result := h.mustTick(t); result.Ran != 2 {
		t.Fatalf("tick = %+v, want the two due jobs to run", result)
	}
	status = h.status(t)
	if status.Queue.Depth != 1 {
		t.Fatalf("depth after tick = %d, want only the deferred job", status.Queue.Depth)
	}
	if status.Queue.Running != 0 {
		t.Fatalf("running = %d, want 0", status.Queue.Running)
	}
	if status.LastRun == nil {
		t.Fatal("last run must be reported")
	}
	if status.LastRun.Outcome != runOutcomeCompleted {
		t.Fatalf("last run outcome = %q", status.LastRun.Outcome)
	}
	if status.LastRun.ActualSeconds != status.LastRun.EstimatedSeconds {
		t.Fatalf("last run actual/estimated = %v/%v, want the executor's actual cost", status.LastRun.ActualSeconds, status.LastRun.EstimatedSeconds)
	}
	if status.Budget.SecondsUsed != 20 {
		t.Fatalf("seconds used = %v, want 20", status.Budget.SecondsUsed)
	}
	if status.Budget.SecondsRemaining != 80 {
		t.Fatalf("seconds remaining = %v, want 80", status.Budget.SecondsRemaining)
	}
	if status.Budget.JobsUsed != 2 || status.Budget.JobsRemaining != 8 {
		t.Fatalf("jobs used/remaining = %d/%d", status.Budget.JobsUsed, status.Budget.JobsRemaining)
	}
	if status.Budget.ModelLoads != 2 {
		t.Fatalf("model loads = %d, want 2 cold models", status.Budget.ModelLoads)
	}
}

func TestStatusReportsWindowState(t *testing.T) {
	window, err := ParseWindow([]time.Weekday{time.Wednesday}, "10:00", "11:00")
	if err != nil {
		t.Fatalf("parse window: %v", err)
	}
	h := newGrantHarness(t, Policy{Mode: ModeBalanced, Location: time.UTC, Windows: []Window{window}})
	status := h.status(t)
	if !status.Window.Configured || status.Window.Open {
		t.Fatalf("window = %+v, want configured and closed", status.Window)
	}
	want := time.Date(2026, time.November, 18, 10, 0, 0, 0, time.UTC)
	if status.Window.NextOpen == nil || !status.Window.NextOpen.Equal(want) {
		t.Fatalf("next open = %v, want %s", status.Window.NextOpen, want)
	}
	h.clock.Set(want)
	status = h.status(t)
	if !status.Window.Open || status.Window.NextOpen != nil {
		t.Fatalf("window = %+v, want open with no next opening", status.Window)
	}
}

func TestStatusReportsIdleGate(t *testing.T) {
	observer := &fakeObserver{state: HostState{Load1: 4, AvailableRAMBytes: 1 << 30}}
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeIdle, Location: time.UTC, IdleLoadBelow: 1, IdleRAMFreeBytes: 4 << 30},
		Admission: &fakeAdmission{},
		Executor:  &fakeExecutor{},
		Observer:  observer,
	})
	status := h.status(t)
	if !status.Idle.Required || status.Idle.Idle {
		t.Fatalf("idle = %+v, want required but busy", status.Idle)
	}
	if status.Idle.Reason == "" || !status.Idle.Observed {
		t.Fatalf("idle = %+v, want an observed reason", status.Idle)
	}
	observer.set(HostState{Load1: 0.1, AvailableRAMBytes: 8 << 30})
	status = h.status(t)
	if !status.Idle.Idle {
		t.Fatalf("idle = %+v, want idle", status.Idle)
	}
}

func TestStatusReportsRunningJob(t *testing.T) {
	h := newGrantHarness(t, Policy{Mode: ModeBalanced, Location: time.UTC})
	h.submit(t, jobSpec("job", "embed", "owner"))
	if _, err := h.store.Lease(context.Background(), LeaseOptions{
		JobIDs: []string{"job"},
		Owner:  "runner",
		TTL:    time.Hour,
	}); err != nil {
		t.Fatalf("lease: %v", err)
	}
	status := h.status(t)
	if status.Queue.Running != 1 || status.Queue.Depth != 0 {
		t.Fatalf("queue = %+v, want one running job and an empty waiting depth", status.Queue)
	}
	if len(status.Running) != 1 || status.Running[0].JobID != "job" {
		t.Fatalf("running = %+v, want the leased job", status.Running)
	}
}

func TestSortDeferrals(t *testing.T) {
	counts := []DeferCount{{Reason: "b", Count: 1}, {Reason: "a", Count: 1}, {Reason: "c", Count: 5}}
	SortDeferrals(counts)
	if counts[0].Reason != "c" || counts[1].Reason != "a" || counts[2].Reason != "b" {
		t.Fatalf("sorted = %+v, want count then reason order", counts)
	}
}
