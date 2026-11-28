package schedule

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestModeOffRunsNothing(t *testing.T) {
	h := newGrantHarness(t, Policy{Mode: ModeOff, Location: time.UTC})
	h.submit(t, jobSpec("job", "embed", "owner"))

	result := h.mustTick(t)
	if result.Ran != 0 || result.Gate != "mode: off" {
		t.Fatalf("tick = %+v, want no work and mode: off", result)
	}
	if job := h.job(t, "job"); job.State != StateQueued {
		t.Fatalf("job state = %q, want queued", job.State)
	}
	if requests, _, _ := h.admission.snapshot(); len(requests) != 0 {
		t.Fatalf("admission was called %d times in off mode", len(requests))
	}
}

func TestPassiveRunsOnlyInteractive(t *testing.T) {
	h := newGrantHarness(t, Policy{Mode: ModePassive, Location: time.UTC})
	h.submit(t,
		jobSpec("interactive", "embed", "owner", func(j *Job) { j.Priority = PriorityInteractive }),
		jobSpec("batch", "annotate", "owner", func(j *Job) { j.ModelAffinity = "winnow:e4b" }),
	)

	result := h.mustTick(t)
	if result.Ran != 1 || len(result.Executed) != 1 || result.Executed[0] != "interactive" {
		t.Fatalf("executed = %v, want only the interactive job", result.Executed)
	}
	if job := h.job(t, "batch"); job.State != StateQueued {
		t.Fatalf("batch job state = %q, want queued (no background fill)", job.State)
	}
	if next := h.mustTick(t); next.Gate != "queue: no ready jobs" {
		t.Fatalf("second tick gate = %q, want queue: no ready jobs", next.Gate)
	}
}

func TestIdleModeUsesHostObservation(t *testing.T) {
	observer := &fakeObserver{state: HostState{Load1: 5, AvailableRAMBytes: 1 << 30}}
	h := newHarness(t, harnessOptions{
		Policy: Policy{
			Mode:             ModeIdle,
			Location:         time.UTC,
			IdleLoadBelow:    1,
			IdleRAMFreeBytes: 4 << 30,
		},
		Admission: &fakeAdmission{},
		Executor:  &fakeExecutor{},
		Observer:  observer,
	})
	h.submit(t, jobSpec("job", "embed", "owner"))

	result := h.mustTick(t)
	if result.Ran != 0 || !strings.Contains(result.Gate, "idle: host busy") {
		t.Fatalf("tick = %+v, want the idle gate", result)
	}
	observer.set(HostState{Load1: 0.2, AvailableRAMBytes: 8 << 30})
	result = h.mustTick(t)
	if result.Ran != 1 {
		t.Fatalf("tick = %+v, want the job to run on an idle host", result)
	}
}

func TestIdleModeRequiresObserver(t *testing.T) {
	clock := &fakeClock{t: testEpoch}
	store, err := NewStore(context.Background(), Config{Path: t.TempDir() + "/schedule.db", Clock: clock.Now})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()
	_, err = New(SchedulerConfig{
		Store:     store,
		Policy:    Policy{Mode: ModeIdle, Location: time.UTC},
		Admission: &fakeAdmission{},
		Executor:  &fakeExecutor{},
	})
	if err == nil {
		t.Fatal("idle mode without an observer must be rejected, not silently denied")
	}
}

func TestBalancedRespectsWindows(t *testing.T) {
	window, err := ParseWindow([]time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}, "10:00", "17:00")
	if err != nil {
		t.Fatalf("parse window: %v", err)
	}
	h := newGrantHarness(t, Policy{Mode: ModeBalanced, Location: time.UTC, Windows: []Window{window}})
	h.submit(t, jobSpec("job", "embed", "owner"))

	result := h.mustTick(t)
	if result.Ran != 0 || result.Gate != "window: closed" {
		t.Fatalf("tick = %+v, want window: closed", result)
	}
	h.clock.Set(time.Date(2026, time.November, 18, 10, 0, 0, 0, time.UTC))
	if result := h.mustTick(t); result.Ran != 1 {
		t.Fatalf("tick = %+v, want the job to run inside the window", result)
	}
}

func TestEagerIgnoresWindowsButHonoursRequiredIdle(t *testing.T) {
	window, err := ParseWindow(nil, "01:00", "02:00")
	if err != nil {
		t.Fatalf("parse window: %v", err)
	}
	h := newGrantHarness(t, Policy{Mode: ModeEager, Location: time.UTC, Windows: []Window{window}})
	h.submit(t, jobSpec("job", "embed", "owner"))
	if result := h.mustTick(t); result.Ran != 1 {
		t.Fatalf("eager tick = %+v, want the job to run outside the window", result)
	}

	observer := &fakeObserver{state: HostState{Load1: 9}}
	gated := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeEager, Location: time.UTC, RequireIdle: true, IdleLoadBelow: 1},
		Admission: &fakeAdmission{},
		Executor:  &fakeExecutor{},
		Observer:  observer,
	})
	gated.submit(t, jobSpec("job", "embed", "owner"))
	if result := gated.mustTick(t); result.Ran != 0 || !strings.Contains(result.Gate, "idle") {
		t.Fatalf("tick = %+v, want RequireIdle to gate eager mode", result)
	}
}

func TestBudgetSecondsExhaustion(t *testing.T) {
	h := newGrantHarness(t, Policy{
		Mode:              ModeBalanced,
		Location:          time.UTC,
		MaxSecondsPerHour: 5,
	})
	h.submit(t, jobSpec("job", "embed", "owner", func(j *Job) { j.Estimate.Seconds = 10 }))

	result := h.mustTick(t)
	if result.Ran != 0 || result.Gate != "budget: seconds per hour exhausted" {
		t.Fatalf("tick = %+v, want the seconds budget to refuse the batch", result)
	}
	if job := h.job(t, "job"); job.State != StateQueued {
		t.Fatalf("job state = %q, want queued", job.State)
	}
}

func TestBudgetJobsPerHour(t *testing.T) {
	h := newGrantHarness(t, Policy{
		Mode:           ModeBalanced,
		Location:       time.UTC,
		MaxJobsPerHour: 1,
	})
	h.submit(t,
		jobSpec("a", "embed", "owner"),
		jobSpec("b", "embed", "owner"),
	)
	result := h.mustTick(t)
	if result.Ran != 1 {
		t.Fatalf("first tick ran %d jobs, want 1", result.Ran)
	}
	result = h.mustTick(t)
	if result.Ran != 0 || result.Gate != "budget: jobs per hour exhausted" {
		t.Fatalf("second tick = %+v, want the jobs budget to stop it", result)
	}
	if job := h.job(t, "b"); job.State != StateQueued {
		t.Fatalf("job b state = %q, want queued", job.State)
	}
}

func TestBudgetModelLoadsPerHour(t *testing.T) {
	h := newGrantHarness(t, Policy{
		Mode:                 ModeBalanced,
		Location:             time.UTC,
		MaxModelLoadsPerHour: 1,
	})
	h.submit(t,
		jobSpec("first", "embed", "owner", func(j *Job) { j.ModelAffinity = "nomic-embed-text" }),
		jobSpec("second", "annotate", "owner", func(j *Job) { j.ModelAffinity = "winnow:e4b" }),
	)
	result := h.mustTick(t)
	if result.Ran != 1 {
		t.Fatalf("ran %d jobs, want 1 cold model load", result.Ran)
	}
	if !containsSubstring(result.Warnings, "model loads per hour exhausted") {
		t.Fatalf("warnings = %v, want the model-load budget skip to be reported", result.Warnings)
	}
	if job := h.job(t, "second"); job.State != StateQueued {
		t.Fatalf("second job state = %q, want queued", job.State)
	}
}

func TestModelGroupingAndDwell(t *testing.T) {
	h := newGrantHarness(t, Policy{
		Mode:                 ModeBalanced,
		Location:             time.UTC,
		MaxModelLoadsPerHour: 1,
		ModelDwell:           30 * time.Minute,
		DefaultMaxBatch:      2,
	})
	jobs := []Job{
		jobSpec("a", "embed", "owner", func(j *Job) { j.MaxBatch = 2 }),
		jobSpec("b", "embed", "owner", func(j *Job) { j.MaxBatch = 2 }),
		jobSpec("c", "embed", "owner", func(j *Job) { j.MaxBatch = 2 }),
	}
	h.submit(t, jobs...)

	result := h.mustTick(t)
	if result.Ran != 2 {
		t.Fatalf("first tick ran %d jobs, want a batch of 2", result.Ran)
	}
	requests, _, _ := h.admission.snapshot()
	if len(requests) != 1 {
		t.Fatalf("admission calls = %d, want one call per batch", len(requests))
	}
	if requests[0].JobID != "a,b" {
		t.Fatalf("batched ids = %q, want a,b", requests[0].JobID)
	}

	// The third job reuses the resident model: dwell, not a new load.
	result = h.mustTick(t)
	if result.Ran != 1 || result.Gate != "" {
		t.Fatalf("second tick = %+v, want the remaining job on the hot model", result)
	}
	var loads int
	if err := h.store.db.QueryRow(`SELECT COUNT(*) FROM schedule_runs WHERE model_load = 1`).Scan(&loads); err != nil {
		t.Fatalf("count loads: %v", err)
	}
	if loads != 1 {
		t.Fatalf("model loads = %d, want 1 across both batches", loads)
	}
}

func TestAdmissionDenialDefersWithBackoff(t *testing.T) {
	admission := &fakeAdmission{}
	admission.push(Verdict{Kind: VerdictQueue, Reason: "VRAM contention"})
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeBalanced, Location: time.UTC},
		Admission: admission,
		Executor:  &fakeExecutor{},
	})
	h.submit(t, jobSpec("job", "embed", "owner"))

	result := h.mustTick(t)
	if result.Ran != 0 || result.Deferred != 1 || result.Gate != "admission: queue" {
		t.Fatalf("tick = %+v, want a deferred denial", result)
	}
	job := h.job(t, "job")
	if job.State != StateDeferred || job.Deferrals != 1 {
		t.Fatalf("job = %q/%d, want deferred/1", job.State, job.Deferrals)
	}
	if !strings.Contains(job.LastReason, "VRAM contention") {
		t.Fatalf("reason = %q, want the broker's reason preserved", job.LastReason)
	}
	if !job.NotBefore.Equal(testEpoch.Add(DefaultDeferBase)) {
		t.Fatalf("not before = %s, want one base backoff", job.NotBefore)
	}

	if result := h.mustTick(t); result.Gate != "queue: no ready jobs" {
		t.Fatalf("tick before backoff = %+v, want the queue to stay empty", result)
	}
	h.clock.Advance(DefaultDeferBase)
	result = h.mustTick(t)
	if result.Ran != 1 {
		t.Fatalf("tick after backoff = %+v, want the job to run", result)
	}
}

func TestPriorityOrdering(t *testing.T) {
	h := newGrantHarness(t, Policy{Mode: ModeBalanced, Location: time.UTC})
	h.submit(t,
		jobSpec("batch", "embed", "owner"),
		jobSpec("interactive", "annotate", "owner", func(j *Job) {
			j.Priority = PriorityInteractive
			j.ModelAffinity = "winnow:e4b"
		}),
	)
	result := h.mustTick(t)
	if len(result.Executed) != 2 {
		t.Fatalf("executed = %v, want both jobs", result.Executed)
	}
	if result.Executed[0] != "interactive" {
		t.Fatalf("executed order = %v, want interactive first", result.Executed)
	}
}

func TestExecutorFailureDefersThenFails(t *testing.T) {
	executor := &fakeExecutor{}
	executor.set(func(context.Context, Job) (Result, error) {
		return Result{ActualSeconds: 2}, errTransient
	})
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeBalanced, Location: time.UTC, MaxModelLoadsPerHour: 100},
		Admission: &fakeAdmission{},
		Executor:  executor,
	})
	h.submit(t, jobSpec("job", "embed", "owner", func(j *Job) { j.MaxAttempts = 2 }))

	result := h.mustTick(t)
	if result.Failed != 1 || result.Deferred != 1 {
		t.Fatalf("tick = %+v, want one failed attempt and one deferral", result)
	}
	job := h.job(t, "job")
	if job.State != StateDeferred || job.Attempts != 1 {
		t.Fatalf("job = %q attempts=%d, want deferred/1", job.State, job.Attempts)
	}
	if !strings.Contains(job.LastReason, "transient executor failure") {
		t.Fatalf("reason = %q", job.LastReason)
	}

	h.clock.Advance(DefaultDeferBase)
	result = h.mustTick(t)
	if result.Failed != 1 || result.Ran != 0 {
		t.Fatalf("second tick = %+v, want the final attempt to fail", result)
	}
	job = h.job(t, "job")
	if job.State != StateFailed || job.Attempts != 2 {
		t.Fatalf("job = %q attempts=%d, want failed/2", job.State, job.Attempts)
	}
}

func TestAdmissionLeasesAreReleased(t *testing.T) {
	h := newGrantHarness(t, Policy{Mode: ModeBalanced, Location: time.UTC})
	h.submit(t, jobSpec("job", "embed", "owner"))
	if result := h.mustTick(t); result.Ran != 1 {
		t.Fatalf("tick = %+v, want the job to run", result)
	}
	_, active, granted := h.admission.snapshot()
	if granted != 1 {
		t.Fatalf("granted = %d, want 1", granted)
	}
	if active != 0 {
		t.Fatalf("active leases = %d, want every admission lease released", active)
	}
}

func TestExecutorPanicIsContained(t *testing.T) {
	executor := &fakeExecutor{}
	executor.set(func(context.Context, Job) (Result, error) {
		panic("executor exploded")
	})
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeBalanced, Location: time.UTC},
		Admission: &fakeAdmission{},
		Executor:  executor,
	})
	h.submit(t, jobSpec("job", "embed", "owner"))
	result := h.mustTick(t)
	if result.Failed != 1 || result.Deferred != 1 {
		t.Fatalf("tick = %+v, want the panic treated as a failed attempt", result)
	}
	job := h.job(t, "job")
	if !strings.Contains(job.LastReason, "panicked") {
		t.Fatalf("reason = %q, want the panic recorded", job.LastReason)
	}
}

func TestRunTicksUntilCanceled(t *testing.T) {
	executed := make(chan struct{}, 1)
	var once sync.Once
	executor := &fakeExecutor{}
	executor.set(func(context.Context, Job) (Result, error) {
		once.Do(func() { executed <- struct{}{} })
		return Result{ActualSeconds: 1}, nil
	})
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeBalanced, Location: time.UTC, Interval: time.Millisecond},
		Admission: &fakeAdmission{},
		Executor:  executor,
	})
	h.submit(t, jobSpec("job", "embed", "owner"))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- h.scheduler.Run(ctx) }()
	select {
	case <-executed:
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not execute the job")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("run error = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("runner did not stop after cancel")
	}
	if job := h.job(t, "job"); job.State != StateCompleted {
		t.Fatalf("job state = %q, want completed", job.State)
	}
}

func TestDegradedAdmissionRunsAndWarns(t *testing.T) {
	admission := &fakeAdmission{}
	admission.push(Verdict{Kind: VerdictDegrade, Reason: "admitted a smaller reservation"})
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeBalanced, Location: time.UTC},
		Admission: admission,
		Executor:  &fakeExecutor{},
	})
	h.submit(t, jobSpec("job", "embed", "owner"))
	result := h.mustTick(t)
	if result.Ran != 1 {
		t.Fatalf("tick = %+v, want a degraded grant to run", result)
	}
	if !containsSubstring(result.Warnings, "degraded") {
		t.Fatalf("warnings = %v, want the degradation recorded", result.Warnings)
	}
}

func TestLeaseContentionIsBenign(t *testing.T) {
	var store *Store
	var owner string
	steal := AdmissionFunc(func(ctx context.Context, req AdmissionRequest) (Lease, Verdict, error) {
		if _, err := store.Lease(ctx, LeaseOptions{
			JobIDs: []string{owner},
			Owner:  "competing-runner",
			TTL:    time.Minute,
		}); err != nil {
			return nil, Verdict{Kind: VerdictDeny, Reason: err.Error()}, err
		}
		return nil, Verdict{Kind: VerdictGrant}, nil
	})
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeBalanced, Location: time.UTC},
		Admission: steal,
		Executor:  &fakeExecutor{},
	})
	store = h.store
	owner = "job"
	h.submit(t, jobSpec("job", "embed", "owner"))

	result := h.mustTick(t)
	if result.Ran != 0 {
		t.Fatalf("tick = %+v, want no work after losing the lease race", result)
	}
	if result.Gate != "lease: contended" {
		t.Fatalf("gate = %q, want lease: contended", result.Gate)
	}
	if job := h.job(t, "job"); job.State != StateRunning {
		t.Fatalf("job state = %q, want the competing runner to hold it", job.State)
	}
}

func TestSchedulerAccessors(t *testing.T) {
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeIdle, Location: time.UTC},
		Admission: &fakeAdmission{},
		Executor:  &fakeExecutor{},
		Observer:  &fakeObserver{state: HostState{}},
	})
	if h.scheduler.Policy().Mode != ModeIdle {
		t.Fatalf("policy mode = %q, want idle", h.scheduler.Policy().Mode)
	}
	if h.scheduler.LeaseOwner() != "test-runner" {
		t.Fatalf("lease owner = %q", h.scheduler.LeaseOwner())
	}
}

func TestFairShareAcrossOwnersAtRunnerLevel(t *testing.T) {
	h := newGrantHarness(t, Policy{
		Mode:             ModeBalanced,
		Location:         time.UTC,
		MaxSharePerOwner: 0.5,
	})
	single := func(id, owner, batch string) Job {
		return jobSpec(id, "embed", owner, func(j *Job) { j.MaxBatch = 1; j.BatchKey = batch })
	}
	h.submit(t, single("a1", "alpha", "early"), single("a2", "alpha", "late"), single("b1", "beta", "early"))

	// Alpha arrives first, but once it is at its share the tick yields to beta
	// instead of running alpha's second job.
	first := h.mustTick(t)
	if len(first.Executed) != 2 || first.Executed[0] != "a1" || first.Executed[1] != "b1" {
		t.Fatalf("first tick executed %v, want a1 then b1 (fair share)", first.Executed)
	}
	if !containsSubstring(first.Warnings, "fair-share") {
		t.Fatalf("warnings = %v, want the yield recorded", first.Warnings)
	}
	second := h.mustTick(t)
	if second.Ran != 1 || second.Executed[0] != "a2" {
		t.Fatalf("second tick = %+v, want alpha's remaining job", second)
	}
}
