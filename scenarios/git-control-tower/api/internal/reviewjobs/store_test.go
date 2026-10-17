package reviewjobs

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file:reviewjobs?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestCreateIsIdempotentByInputKey(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	original := Job{ID: "job-1", IdempotencyKey: "same", InputDigest: "sha256:input", PolicyVersion: "policy-1", Checks: []Check{{Name: "tests", Availability: "requested"}}}
	got, created, err := s.Create(ctx, original)
	if err != nil || !created || got.ID != "job-1" {
		t.Fatalf("first create = %#v created=%v err=%v", got, created, err)
	}
	got, created, err = s.Create(ctx, Job{ID: "job-2", IdempotencyKey: "same", InputDigest: "sha256:other"})
	if err != nil || created || got.ID != "job-1" {
		t.Fatalf("duplicate create = %#v created=%v err=%v", got, created, err)
	}
}

func TestResultAndListSurviveStoreReopen(t *testing.T) {
	s := testStore(t)
	job, created, err := s.Create(context.Background(), Job{ID: "job-result", IdempotencyKey: "result-key", InputDigest: "sha256:result", PolicyVersion: "p1", State: Running, ResultJSON: `{"readiness":"yellow"}`, Checks: []Check{{Name: "tests", Availability: "requested", Verdict: "pending"}}})
	if err != nil || !created || job.ResultJSON == "" {
		t.Fatalf("create result job = %#v created=%v err=%v", job, created, err)
	}
	if err := s.SetResult(context.Background(), job.ID, `{"readiness":"green"}`); err != nil {
		t.Fatal(err)
	}
	jobs, err := s.List(context.Background())
	if err != nil || len(jobs) != 1 || jobs[0].ResultJSON != `{"readiness":"green"}` {
		t.Fatalf("list = %#v err=%v", jobs, err)
	}
}

func TestListReleasesSinglePoolBeforeLoadingJobs(t *testing.T) {
	db, err := sql.Open("sqlite", "file:reviewjobs-single-pool?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	store, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Create(context.Background(), Job{
		ID: "job-single-pool", IdempotencyKey: "single-pool", InputDigest: "sha256:single-pool",
		PolicyVersion: "p1", State: Running, Checks: []Check{{Name: "tests", Availability: "requested"}},
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	jobs, err := store.List(ctx)
	if err != nil || len(jobs) != 1 || jobs[0].ID != "job-single-pool" {
		t.Fatalf("list = %#v err=%v", jobs, err)
	}
}

func TestEnsureSchemaAddsNewColumnsToLegacyTable(t *testing.T) {
	db, err := sql.Open("sqlite", "file:reviewjobs-legacy?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE gct_review_jobs (id TEXT PRIMARY KEY, idempotency_key TEXT NOT NULL UNIQUE, input_digest TEXT NOT NULL, policy_version TEXT NOT NULL, state TEXT NOT NULL, checks_json TEXT NOT NULL, result_ref TEXT NOT NULL DEFAULT '', error TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(db); err != nil {
		t.Fatalf("legacy migration: %v", err)
	}
}

func TestTransitionRequiresExpectedSourceAndPreservesFailedCheck(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, _, err := s.Create(ctx, Job{ID: "job-1", IdempotencyKey: "key", InputDigest: "sha256:x", Checks: []Check{{Name: "tests", Availability: "requested"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpdateCheck(ctx, "job-1", Check{Name: "tests", ExecutionID: "run-1", Availability: "available", Verdict: "failed", Detail: "assertion failed"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Transition(ctx, "job-1", []State{Queued}, Succeeded, "result-1", ""); err != nil {
		t.Fatal(err)
	}
	job, err := s.Get(ctx, "job-1")
	if err != nil {
		t.Fatal(err)
	}
	if job.Checks[0].Verdict != "failed" || job.ResultRef != "result-1" {
		t.Fatalf("job lost failed check: %#v", job)
	}
	if err := s.Transition(ctx, "job-1", []State{Running}, Failed, "", "replay"); err == nil {
		t.Fatal("invalid source transition accepted")
	}
}
