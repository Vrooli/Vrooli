package schedule

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) (*Store, *fakeClock) {
	t.Helper()
	clock := &fakeClock{t: testEpoch}
	store, err := NewStore(context.Background(), Config{
		Path:  filepath.Join(t.TempDir(), "schedule.db"),
		Clock: clock.Now,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, clock
}

func TestSubmitNormalizesAndRejectsDuplicates(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	job, err := store.Submit(ctx, Job{Kind: "embed", Owner: "agent-manager", Value: 2})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}
	if job.ID == "" {
		t.Fatal("submit must assign an id")
	}
	if job.State != StateQueued {
		t.Fatalf("state = %q, want queued", job.State)
	}
	if job.Priority != PriorityBatch || job.ResourceClass != ResourceClassCompute {
		t.Fatalf("defaults = %q/%q, want batch/compute", job.Priority, job.ResourceClass)
	}
	if job.MaxAttempts != DefaultMaxAttempts {
		t.Fatalf("max attempts = %d, want %d", job.MaxAttempts, DefaultMaxAttempts)
	}
	if !job.NotBefore.Equal(testEpoch) {
		t.Fatalf("not before = %s, want %s", job.NotBefore, testEpoch)
	}

	if _, err := store.Submit(ctx, job); !errors.Is(err, ErrJobExists) {
		t.Fatalf("duplicate submit error = %v, want ErrJobExists", err)
	}
}

func TestSubmitRejectsInvalidJobs(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	cases := map[string]Job{
		"no kind":           {Owner: "owner"},
		"no owner":          {Kind: "embed"},
		"negative value":    {Kind: "embed", Owner: "owner", Value: -1},
		"negative estimate": {Kind: "embed", Owner: "owner", Estimate: Estimate{Seconds: -1}},
		"unknown priority":  {Kind: "embed", Owner: "owner", Priority: "urgent"},
		"unknown class":     {Kind: "embed", Owner: "owner", ResourceClass: "disk"},
	}
	for name, job := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := store.Submit(ctx, job); !errors.Is(err, ErrInvalidJob) {
				t.Fatalf("submit error = %v, want ErrInvalidJob", err)
			}
		})
	}
	deadline := testEpoch.Add(-time.Minute)
	if _, err := store.Submit(ctx, Job{Kind: "embed", Owner: "owner", Deadline: &deadline}); !errors.Is(err, ErrInvalidJob) {
		t.Fatalf("deadline before not_before error = %v, want ErrInvalidJob", err)
	}
}

func TestSubmitBatchIsAtomic(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	_, err := store.SubmitBatch(ctx, []Job{
		jobSpec("ok", "embed", "owner"),
		{Kind: "", Owner: "owner"},
	})
	if err == nil {
		t.Fatal("expected the batch to fail")
	}
	if _, getErr := store.Get(ctx, "ok"); !errors.Is(getErr, ErrJobNotFound) {
		t.Fatalf("partial batch persisted: %v", getErr)
	}
}

func TestReadyOrdersByDueTimeAndSkipsFuture(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	later := jobSpec("later", "embed", "owner", func(j *Job) { j.NotBefore = testEpoch.Add(10 * time.Minute) })
	laterNotBefore := later.NotBefore
	second := jobSpec("second", "embed", "owner", func(j *Job) { j.NotBefore = testEpoch.Add(2 * time.Minute) })
	secondNotBefore := second.NotBefore
	first := jobSpec("first", "embed", "owner")
	if _, err := store.SubmitBatch(ctx, []Job{later, second, first}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	ready, err := store.Ready(ctx, 10)
	if err != nil {
		t.Fatalf("ready: %v", err)
	}
	if len(ready) != 1 || ready[0].ID != "first" {
		t.Fatalf("ready = %v, want only first", describeJobs(ready))
	}
	clock.Set(testEpoch.Add(3 * time.Minute))
	ready, err = store.Ready(ctx, 10)
	if err != nil {
		t.Fatalf("ready: %v", err)
	}
	if len(ready) != 2 || ready[0].ID != "first" || ready[1].ID != "second" {
		t.Fatalf("ready = %v, want first then second", describeJobs(ready))
	}
	if !laterNotBefore.After(testEpoch) || !secondNotBefore.After(testEpoch) {
		t.Fatal("test setup error: future jobs must be in the future")
	}
}

func TestLeaseCountsOneModelLoadInsideDwell(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	if _, err := store.SubmitBatch(ctx, []Job{
		jobSpec("a", "embed", "owner"),
		jobSpec("b", "embed", "owner"),
	}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	leased, err := store.Lease(ctx, LeaseOptions{
		JobIDs:     []string{"a", "b"},
		Owner:      "runner",
		TTL:        time.Minute,
		ModelDwell: 10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("lease: %v", err)
	}
	if len(leased) != 2 {
		t.Fatalf("leased %d jobs, want 2", len(leased))
	}
	if !leased[0].ModelLoad {
		t.Fatal("the first job on a cold model must pay for the load")
	}
	if leased[1].ModelLoad {
		t.Fatal("the second job inside the dwell must reuse the loaded model")
	}
	for _, item := range leased {
		if item.Job.State != StateRunning {
			t.Fatalf("job %s state = %q, want running", item.Job.ID, item.Job.State)
		}
		if item.Job.Attempts != 1 {
			t.Fatalf("job %s attempts = %d, want 1", item.Job.ID, item.Job.Attempts)
		}
		if item.RunID == "" {
			t.Fatalf("job %s has no run id", item.Job.ID)
		}
	}

	hot, err := store.HotModels(ctx, 10*time.Minute)
	if err != nil {
		t.Fatalf("hot models: %v", err)
	}
	if _, ok := hot["nomic-embed-text"]; !ok {
		t.Fatal("expected the model to be hot inside the dwell")
	}
}

func TestLeaseRejectsNonDueAndFutureJobs(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	future := jobSpec("future", "embed", "owner", func(j *Job) { j.NotBefore = testEpoch.Add(time.Hour) })
	futureNotBefore := future.NotBefore
	if _, err := store.Submit(ctx, future); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := store.Lease(ctx, LeaseOptions{JobIDs: []string{"future"}, Owner: "runner"}); !errors.Is(err, ErrNotLeasable) {
		t.Fatalf("lease error = %v, want ErrNotLeasable", err)
	}
	clock.Set(futureNotBefore.Add(time.Second))
	if _, err := store.Lease(ctx, LeaseOptions{JobIDs: []string{"future"}, Owner: "runner", TTL: time.Minute}); err != nil {
		t.Fatalf("lease after due: %v", err)
	}
}

func TestCompleteRecordsActualCost(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	if _, err := store.Submit(ctx, jobSpec("job", "embed", "owner")); err != nil {
		t.Fatalf("submit: %v", err)
	}
	leased, err := store.Lease(ctx, LeaseOptions{JobIDs: []string{"job"}, Owner: "runner", TTL: time.Minute})
	if err != nil {
		t.Fatalf("lease: %v", err)
	}
	job, err := store.Complete(ctx, Completion{
		JobID:         "job",
		Outcome:       StateCompleted,
		ActualSeconds: 4.5,
		Items:         7,
		Tokens:        900,
		Checkpoint:    "done",
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if job.State != StateCompleted || job.FinishedAt == nil {
		t.Fatalf("job = %+v, want completed with a finish time", job)
	}
	if job.Checkpoint != "done" {
		t.Fatalf("checkpoint = %q, want done", job.Checkpoint)
	}
	if job.LeaseOwner != "" || job.LeaseExpiresAt != nil {
		t.Fatalf("lease not cleared: %+v", job)
	}
	var actual, estimated float64
	var items, tokens int
	if err := store.db.QueryRow(`SELECT actual_seconds, estimated_seconds, items, tokens FROM schedule_runs WHERE run_id = ?`, leased[0].RunID).
		Scan(&actual, &estimated, &items, &tokens); err != nil {
		t.Fatalf("read run: %v", err)
	}
	if actual != 4.5 || estimated != 10 || items != 7 || tokens != 900 {
		t.Fatalf("run = actual %v estimated %v items %d tokens %d", actual, estimated, items, tokens)
	}
	if _, err := store.Complete(ctx, Completion{JobID: "job", Outcome: StateCompleted}); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("second complete error = %v, want ErrLeaseLost", err)
	}
}

func TestDeferBacksOff(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	if _, err := store.Submit(ctx, jobSpec("job", "embed", "owner")); err != nil {
		t.Fatalf("submit: %v", err)
	}
	policy, err := Policy{}.Normalized()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	job, err := store.Defer(ctx, DeferOptions{JobID: "job", Reason: "capacity denied", Backoff: policy.backoff})
	if err != nil {
		t.Fatalf("defer: %v", err)
	}
	if job.State != StateDeferred || job.Deferrals != 1 {
		t.Fatalf("job = %q/%d, want deferred/1", job.State, job.Deferrals)
	}
	if !job.NotBefore.Equal(testEpoch.Add(DefaultDeferBase)) {
		t.Fatalf("not before = %s, want %s", job.NotBefore, testEpoch.Add(DefaultDeferBase))
	}
	if job.LastReason != "capacity denied" {
		t.Fatalf("reason = %q", job.LastReason)
	}
	clock.Advance(DefaultDeferBase)
	ready, err := store.Ready(ctx, 0)
	if err != nil {
		t.Fatalf("ready: %v", err)
	}
	if len(ready) != 1 {
		t.Fatalf("ready = %d jobs, want 1", len(ready))
	}
	if _, err := store.Defer(ctx, DeferOptions{JobID: "job", Reason: "still denied", Backoff: policy.backoff}); err != nil {
		t.Fatalf("second defer: %v", err)
	}
	job = mustGet(t, store, "job")
	if !job.NotBefore.Equal(testEpoch.Add(DefaultDeferBase).Add(2 * DefaultDeferBase)) {
		t.Fatalf("second backoff = %s, want 2x base after the first due time", job.NotBefore)
	}
}

func TestCancelClosesRunningJobAndChargesElapsed(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	if _, err := store.Submit(ctx, jobSpec("job", "embed", "owner")); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := store.Lease(ctx, LeaseOptions{JobIDs: []string{"job"}, Owner: "runner", TTL: time.Hour}); err != nil {
		t.Fatalf("lease: %v", err)
	}
	clock.Advance(90 * time.Second)
	job, err := store.Cancel(ctx, "job", "operator cancelled")
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if job.State != StateCanceled || job.LastReason != "operator cancelled" {
		t.Fatalf("job = %+v, want canceled with reason", job)
	}
	var actual float64
	var outcome string
	if err := store.db.QueryRow(`SELECT actual_seconds, outcome FROM schedule_runs WHERE job_id = 'job'`).Scan(&actual, &outcome); err != nil {
		t.Fatalf("read run: %v", err)
	}
	if outcome != runOutcomeCanceled {
		t.Fatalf("run outcome = %q, want canceled", outcome)
	}
	if actual < 89 || actual > 91 {
		t.Fatalf("actual = %v, want about 90", actual)
	}
}

func mustGet(t *testing.T, store *Store, id string) Job {
	t.Helper()
	job, err := store.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("get %s: %v", id, err)
	}
	return job
}

func TestCheckpointSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "schedule.db")
	clock := &fakeClock{t: testEpoch}
	store, err := NewStore(context.Background(), Config{Path: path, Clock: clock.Now})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ctx := context.Background()
	if _, err := store.Submit(ctx, jobSpec("job", "embed", "owner")); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := store.Checkpoint(ctx, "job", `{"offset":128}`); err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	reopened, err := NewStore(context.Background(), Config{Path: path, Clock: clock.Now})
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	job, err := reopened.Get(ctx, "job")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if job.Checkpoint != `{"offset":128}` {
		t.Fatalf("checkpoint = %q, want persisted value", job.Checkpoint)
	}
	if job.State != StateQueued {
		t.Fatalf("state = %q, want queued", job.State)
	}
}

func TestExpireOverdueFailsLateJobs(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	deadline := testEpoch.Add(5 * time.Minute)
	if _, err := store.Submit(ctx, jobSpec("job", "embed", "owner", func(j *Job) { j.Deadline = &deadline })); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if expired, err := store.ExpireOverdue(ctx); err != nil || expired != 0 {
		t.Fatalf("expired = %d, err = %v, want 0", expired, err)
	}
	clock.Set(deadline.Add(time.Second))
	expired, err := store.ExpireOverdue(ctx)
	if err != nil {
		t.Fatalf("expire: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expired = %d, want 1", expired)
	}
	job := mustGet(t, store, "job")
	if job.State != StateFailed || job.LastReason != "deadline exceeded" {
		t.Fatalf("job = %q/%q, want failed/deadline exceeded", job.State, job.LastReason)
	}
}

func TestResolveConfiguredPathRefusesImplicitTestDatabase(t *testing.T) {
	if _, err := ResolveConfiguredPath(Config{}); err == nil {
		t.Fatal("expected an implicit path to be refused under go test")
	}
	path := filepath.Join(t.TempDir(), "explicit.db")
	got, err := ResolveConfiguredPath(Config{Path: path})
	if err != nil {
		t.Fatalf("resolve explicit path: %v", err)
	}
	if got != path {
		t.Fatalf("path = %q, want %q", got, path)
	}
}

func TestStoreRejectsUnknownSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.db")
	db, err := sql.Open("sqlite", buildDSN(path))
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	if _, err := db.Exec(`PRAGMA user_version = 99`); err != nil {
		t.Fatalf("stamp version: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close raw: %v", err)
	}
	if _, err := NewStore(context.Background(), Config{Path: path}); err == nil {
		t.Fatal("expected a newer schema version to be refused")
	}
}

func TestListFilters(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	if _, err := store.SubmitBatch(ctx, []Job{
		jobSpec("embed", "embed", "alpha"),
		jobSpec("annotate", "annotate", "beta", func(j *Job) { j.ModelAffinity = "winnow:e4b" }),
	}); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := store.Cancel(ctx, "annotate", "withdrawn"); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	cases := []struct {
		name   string
		filter ListFilter
		want   []string
	}{
		{"all", ListFilter{}, []string{"embed", "annotate"}},
		{"by kind", ListFilter{Kinds: []string{"embed"}}, []string{"embed"}},
		{"by owner", ListFilter{Owners: []string{"beta"}}, []string{"annotate"}},
		{"by model", ListFilter{Models: []string{"winnow:e4b"}}, []string{"annotate"}},
		{"by state", ListFilter{States: []State{StateCanceled}}, []string{"annotate"}},
		{"limit", ListFilter{Limit: 1}, []string{"embed"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			jobs, err := store.List(ctx, test.filter)
			if err != nil {
				t.Fatalf("list: %v", err)
			}
			got := make([]string, len(jobs))
			for i, job := range jobs {
				got[i] = job.ID
			}
			if len(got) != len(test.want) {
				t.Fatalf("list = %v, want %v", got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Fatalf("list = %v, want %v", got, test.want)
				}
			}
		})
	}
}

func TestHeartbeatLeaseExtendsAndRejectsReleasedLeases(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	if _, err := store.Submit(ctx, jobSpec("job", "embed", "owner")); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := store.Lease(ctx, LeaseOptions{JobIDs: []string{"job"}, Owner: "runner", TTL: time.Minute}); err != nil {
		t.Fatalf("lease: %v", err)
	}
	expires, err := store.HeartbeatLease(ctx, "job", 10*time.Minute)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	if !expires.Equal(testEpoch.Add(10 * time.Minute)) {
		t.Fatalf("expiry = %s, want the extended deadline", expires)
	}
	clock.Advance(2 * time.Minute)
	recovered, err := store.ReapExpiredLeases(ctx)
	if err != nil {
		t.Fatalf("reap: %v", err)
	}
	if len(recovered) != 0 {
		t.Fatalf("reaped %d jobs, want the renewed lease kept", len(recovered))
	}
	if _, err := store.Complete(ctx, Completion{JobID: "job", Outcome: StateCompleted, ActualSeconds: 1}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if _, err := store.HeartbeatLease(ctx, "job", time.Minute); !errors.Is(err, ErrLeaseLost) {
		t.Fatalf("heartbeat error = %v, want ErrLeaseLost", err)
	}
}

func TestBudgetResetsAfterTheRollingHour(t *testing.T) {
	store, clock := newTestStore(t)
	ctx := context.Background()
	if _, err := store.Submit(ctx, jobSpec("job", "embed", "owner")); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if _, err := store.Lease(ctx, LeaseOptions{JobIDs: []string{"job"}, Owner: "runner", TTL: time.Minute}); err != nil {
		t.Fatalf("lease: %v", err)
	}
	if _, err := store.Complete(ctx, Completion{JobID: "job", Outcome: StateCompleted, ActualSeconds: 7, Items: 1}); err != nil {
		t.Fatalf("complete: %v", err)
	}
	budget, err := store.Budget(ctx)
	if err != nil {
		t.Fatalf("budget: %v", err)
	}
	if budget.SecondsUsed != 7 || budget.JobsUsed != 1 || budget.ModelLoads != 1 {
		t.Fatalf("budget = %+v, want one 7s run with one load", budget)
	}
	clock.Advance(2 * time.Hour)
	budget, err = store.Budget(ctx)
	if err != nil {
		t.Fatalf("budget: %v", err)
	}
	if budget.SecondsUsed != 0 || budget.JobsUsed != 0 || budget.ModelLoads != 0 {
		t.Fatalf("budget = %+v, want an empty rolling hour", budget)
	}
}

func TestStorePathIsAbsolute(t *testing.T) {
	store, _ := newTestStore(t)
	if path := store.Path(); !filepath.IsAbs(path) {
		t.Fatalf("path = %q, want an absolute path", path)
	}
}
