package schedule

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func countRuns(t *testing.T, store *Store, jobID string) map[string]int {
	t.Helper()
	rows, err := store.db.Query(`SELECT outcome, COUNT(*) FROM schedule_runs WHERE job_id = ? GROUP BY outcome`, jobID)
	if err != nil {
		t.Fatalf("count runs: %v", err)
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var outcome string
		var count int
		if err := rows.Scan(&outcome, &count); err != nil {
			t.Fatalf("scan run count: %v", err)
		}
		counts[outcome] = count
	}
	return counts
}

func TestCrashRecoveryRequeuesAndRuns(t *testing.T) {
	h := newGrantHarness(t, Policy{Mode: ModeBalanced, Location: time.UTC})
	h.submit(t, jobSpec("job", "embed", "owner"))

	// Simulate a runner that leased the job and died without completing it.
	if _, err := h.store.Lease(context.Background(), LeaseOptions{
		JobIDs: []string{"job"},
		Owner:  "dead-runner",
		TTL:    time.Minute,
	}); err != nil {
		t.Fatalf("lease: %v", err)
	}
	h.clock.Advance(2 * time.Minute)

	result := h.mustTick(t)
	if result.Recovered != 1 {
		t.Fatalf("recovered = %d, want 1", result.Recovered)
	}
	if result.Ran != 1 {
		t.Fatalf("ran = %d, want the requeued job to run in the same tick", result.Ran)
	}
	job := h.job(t, "job")
	if job.State != StateCompleted {
		t.Fatalf("job state = %q, want completed", job.State)
	}
	if job.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2 (the crashed attempt plus the retry)", job.Attempts)
	}
	outcomes := countRuns(t, h.store, "job")
	if outcomes[runOutcomeExpired] != 1 || outcomes[runOutcomeCompleted] != 1 {
		t.Fatalf("run outcomes = %v, want one expired and one completed", outcomes)
	}
}

func TestCrashRecoveryFailsJobThatExhaustedAttempts(t *testing.T) {
	h := newGrantHarness(t, Policy{Mode: ModeBalanced, Location: time.UTC})
	h.submit(t, jobSpec("job", "embed", "owner", func(j *Job) { j.MaxAttempts = 1 }))
	if _, err := h.store.Lease(context.Background(), LeaseOptions{
		JobIDs: []string{"job"},
		Owner:  "dead-runner",
		TTL:    time.Minute,
	}); err != nil {
		t.Fatalf("lease: %v", err)
	}
	h.clock.Advance(2 * time.Minute)

	result := h.mustTick(t)
	if result.Recovered != 1 || result.Ran != 0 {
		t.Fatalf("tick = %+v, want a recovered job that does not run", result)
	}
	job := h.job(t, "job")
	if job.State != StateFailed {
		t.Fatalf("job state = %q, want failed", job.State)
	}
	if job.LastReason == "" || job.Attempts != 1 {
		t.Fatalf("job = %+v, want a recorded reason and one attempt", job)
	}
}

func TestRecoveryLeavesLiveLeaseAlone(t *testing.T) {
	h := newGrantHarness(t, Policy{Mode: ModeBalanced, Location: time.UTC})
	h.submit(t, jobSpec("job", "embed", "owner"))
	if _, err := h.store.Lease(context.Background(), LeaseOptions{
		JobIDs: []string{"job"},
		Owner:  "live-runner",
		TTL:    time.Hour,
	}); err != nil {
		t.Fatalf("lease: %v", err)
	}
	h.clock.Advance(time.Minute)
	result := h.mustTick(t)
	if result.Recovered != 0 {
		t.Fatalf("recovered = %d, want a live lease untouched", result.Recovered)
	}
	if job := h.job(t, "job"); job.State != StateRunning {
		t.Fatalf("job state = %q, want running", job.State)
	}
}

type resumingExecutor struct {
	mu              sync.Mutex
	failures        int
	attempts        int
	checkpointsSeen []string
}

func (e *resumingExecutor) Execute(ctx context.Context, job Job) (Result, error) {
	return e.ExecuteWithCheckpoint(ctx, job, nopCheckpointer{})
}

func (e *resumingExecutor) ExecuteWithCheckpoint(ctx context.Context, job Job, checkpoint Checkpointer) (Result, error) {
	e.mu.Lock()
	e.attempts++
	attempt := e.attempts
	e.checkpointsSeen = append(e.checkpointsSeen, job.Checkpoint)
	e.mu.Unlock()
	if attempt <= e.failures {
		next := fmt.Sprintf(`{"offset":%d}`, attempt*8)
		if err := checkpoint.Save(ctx, next); err != nil {
			return Result{}, err
		}
		return Result{ActualSeconds: 2, Items: attempt * 8}, errTransient
	}
	return Result{ActualSeconds: 1, Items: 8, Checkpoint: `{"offset":16}`}, nil
}

func (e *resumingExecutor) seen() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.checkpointsSeen...)
}

func TestCheckpointResumeAfterFailure(t *testing.T) {
	executor := &resumingExecutor{failures: 1}
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeBalanced, Location: time.UTC},
		Admission: &fakeAdmission{},
		Executor:  executor,
	})
	h.submit(t, jobSpec("job", "embed", "owner"))

	if result := h.mustTick(t); result.Deferred != 1 {
		t.Fatalf("tick = %+v, want the first attempt deferred", result)
	}
	job := h.job(t, "job")
	if job.Checkpoint != `{"offset":8}` {
		t.Fatalf("checkpoint = %q, want the mid-run save", job.Checkpoint)
	}

	h.clock.Advance(DefaultDeferBase)
	if result := h.mustTick(t); result.Ran != 1 {
		t.Fatalf("tick = %+v, want the resumed attempt to succeed", result)
	}
	seen := executor.seen()
	if len(seen) != 2 || seen[1] != `{"offset":8}` {
		t.Fatalf("checkpoints seen = %v, want the second attempt to resume from offset 8", seen)
	}
	job = h.job(t, "job")
	if job.State != StateCompleted || job.Checkpoint != `{"offset":16}` {
		t.Fatalf("job = %q checkpoint %q, want completed/offset 16", job.State, job.Checkpoint)
	}
}

func TestCrashRecoveryPreservesCheckpoint(t *testing.T) {
	executor := &fakeExecutor{}
	h := newHarness(t, harnessOptions{
		Policy:    Policy{Mode: ModeBalanced, Location: time.UTC},
		Admission: &fakeAdmission{},
		Executor:  executor,
	})
	h.submit(t, jobSpec("job", "embed", "owner"))
	if _, err := h.store.Lease(context.Background(), LeaseOptions{
		JobIDs: []string{"job"},
		Owner:  "dead-runner",
		TTL:    time.Minute,
	}); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if err := h.store.Checkpoint(context.Background(), "job", `{"offset":64}`); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	h.clock.Advance(2 * time.Minute)
	if result := h.mustTick(t); result.Ran != 1 {
		t.Fatalf("tick = %+v, want the job to resume", result)
	}
	calls := executor.seen()
	if len(calls) != 1 || calls[0].Checkpoint != `{"offset":64}` {
		t.Fatalf("executor saw checkpoints %v, want offset 64", calls)
	}
}
