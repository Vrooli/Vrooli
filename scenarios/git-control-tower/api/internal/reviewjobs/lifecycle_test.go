package reviewjobs

import (
	"context"
	"testing"
)

func TestJobLifecycleTransitionMatrix(t *testing.T) {
	states := []State{Queued, Running, Partial, Succeeded, Failed, Cancelled, Interrupted, Cancelling}
	allowed := map[[2]State]bool{
		{Queued, Running}:       true,
		{Queued, Failed}:        true,
		{Queued, Cancelled}:     true,
		{Queued, Cancelling}:    true,
		{Queued, Interrupted}:   true,
		{Running, Queued}:       true,
		{Running, Partial}:      true,
		{Running, Succeeded}:    true,
		{Running, Failed}:       true,
		{Running, Cancelled}:    true,
		{Running, Cancelling}:   true,
		{Running, Interrupted}:  true,
		{Partial, Queued}:       true,
		{Partial, Running}:      true,
		{Partial, Succeeded}:    true,
		{Partial, Failed}:       true,
		{Partial, Cancelled}:    true,
		{Partial, Cancelling}:   true,
		{Partial, Interrupted}:  true,
		{Cancelling, Failed}:    true,
		{Cancelling, Cancelled}: true,
	}
	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				job := Job{State: from}
				err := job.Transition(to)
				if from == to {
					if err != nil || job.State != from {
						t.Fatalf("same-state transition: state=%q err=%v", job.State, err)
					}
					return
				}
				if allowed[[2]State{from, to}] {
					if err != nil || job.State != to {
						t.Fatalf("allowed transition %q -> %q: state=%q err=%v", from, to, job.State, err)
					}
					return
				}
				if err == nil || job.State != from {
					t.Fatalf("rejected transition mutated job: %q -> %q, current=%q err=%v", from, to, job.State, err)
				}
			})
		}
	}
}

func TestJobLifecycleRejectsUnknownStatesAndNil(t *testing.T) {
	job := Job{State: State("unknown")}
	if err := job.Transition(Failed); err == nil || job.State != State("unknown") {
		t.Fatalf("unknown source accepted or mutated: state=%q err=%v", job.State, err)
	}
	job.State = Queued
	if err := job.Transition(State("unknown")); err == nil || job.State != Queued {
		t.Fatalf("unknown target accepted or mutated: state=%q err=%v", job.State, err)
	}
	var nilJob *Job
	if err := nilJob.Transition(Queued); err == nil {
		t.Fatal("nil job transition accepted")
	}
}

func TestStoreRejectsUnknownLifecycleStates(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	if _, _, err := s.Create(ctx, Job{ID: "invalid-create", IdempotencyKey: "invalid-create", InputDigest: "sha256:invalid", State: "unknown"}); err == nil {
		t.Fatal("unknown create state accepted")
	}
	job, _, err := s.Create(ctx, Job{ID: "invalid-read", IdempotencyKey: "invalid-read", InputDigest: "sha256:read"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE gct_review_jobs SET state=? WHERE id=?`, "unknown", job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(ctx, job.ID); err == nil {
		t.Fatal("unknown persisted state accepted")
	}
}
