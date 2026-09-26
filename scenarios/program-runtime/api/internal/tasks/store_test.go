package tasks

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func taskDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(Schema())
	require.NoError(t, err)
	return db
}

func seedRecord() Record {
	return Record{TaskID: "task", AttemptID: "attempt", SessionID: "session", ProgramID: "program", Operation: "fixture.run", Digest: "digest", ResumeHash: "receipt", InputHash: "inputs", Scope: "fixture", ContextKey: "context", Provenance: "test", StartedAt: "2026-09-01T00:00:00Z", FinishDigest: "finish-digest"}
}

func TestTaskBeginIdentityAndOrdinal(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "tasks.db")))
	input := seedRecord()
	r, created, err := s.Begin(ctx, input)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, 1, r.AttemptNumber)
	_, created, err = s.Begin(ctx, input)
	require.NoError(t, err)
	require.False(t, created)
	for _, change := range []func(*Record){func(r *Record) { r.InputHash = "changed" }, func(r *Record) { r.ResumeHash = "wrong" }, func(r *Record) { r.Provenance = "operator" }, func(r *Record) { r.Digest = "changed" }} {
		altered := input
		change(&altered)
		_, _, err = s.Begin(ctx, altered)
		require.Error(t, err)
	}
	next := input
	next.AttemptID = "second"
	next.StartedAt = "2026-09-01T00:01:00Z"
	_, _, err = s.Begin(ctx, next)
	require.ErrorContains(t, err, "inspection")
	_, err = s.Update(ctx, input.AttemptID, func(r *Record) error { r.State = "completed"; return nil })
	require.NoError(t, err)
	r, created, err = s.Begin(ctx, next)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, 2, r.AttemptNumber)
	require.Equal(t, input.StartedAt, r.TaskStartedAt)
}

func TestFragmentCachePersistsVerifiedTraceWithoutCountingReads(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "fragments.db")))
	_, err := s.PutFragment(ctx, Fragment{StepKey: "program|key|find-account", Fragment: "def step(inputs, bindings):\n    return {'id': inputs['id']}", SourceProgramID: "program-1", Source: "learn.act('find-account')", StepName: "find-account", TraceInputs: map[string]any{"id": "42"}, TraceOutput: map[string]any{"id": "42"}}, 1, 0)
	require.NoError(t, err)
	got, err := s.GetFragment(ctx, "program|key|find-account")
	require.NoError(t, err)
	require.Equal(t, 1, got.Verified)
	require.Equal(t, 0, got.CachedRuns)
	require.Equal(t, map[string]any{"id": "42"}, got.TraceInputs)
	rows, err := s.ListFragments(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	_, err = s.PutFragment(ctx, Fragment{StepKey: got.StepKey, Fragment: got.Fragment, SourceProgramID: got.SourceProgramID}, 1, 1)
	require.NoError(t, err)
	_, err = s.GetFragment(ctx, got.StepKey)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestTaskRestartRecoveryAndDeliveryNeverReplayDomain(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "tasks.db")
	db := taskDB(t, path)
	s := NewStore(db)
	for _, id := range []string{"running", "prepared", "completed"} {
		r := seedRecord()
		r.TaskID = id
		r.AttemptID = id
		_, _, err := s.Begin(ctx, r)
		require.NoError(t, err)
		if id != "prepared" {
			_, err = s.Update(ctx, id, func(r *Record) error {
				r.State = id
				if id == "completed" {
					r.Outcome = "verified_success"
					r.Delivery = "pending"
					r.FinishInputs = map[string]any{"scope": "fixture", "attempt": map[string]any{"attempt_id": id}}
				}
				return nil
			})
			require.NoError(t, err)
		}
	}
	require.NoError(t, db.Close())
	s = NewStore(taskDB(t, path))
	require.NoError(t, s.Recover(ctx, ""))
	for _, id := range []string{"prepared", "running"} {
		r, err := s.Get(ctx, id)
		require.NoError(t, err)
		require.Equal(t, "uncertain", r.State)
		require.Equal(t, "unknown", r.Outcome)
		require.Equal(t, "blocked", r.Delivery)
	}
	var intents []map[string]any
	drainer := &Drainer{Store: s, Deliver: func(_ context.Context, r Record) (map[string]any, error) {
		require.Equal(t, "completed", r.AttemptID)
		require.Equal(t, "finish-digest", r.FinishDigest)
		intents = append(intents, r.FinishInputs)
		switch len(intents) {
		case 1:
			return nil, errors.New("temporary memory outage")
		case 2:
			return map[string]any{"status": "ok"}, nil
		}
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "entry"}}, nil
	}}
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	require.NoError(t, drainer.DrainOnce(ctx, now))
	r, err := s.Get(ctx, "completed")
	require.NoError(t, err)
	require.Equal(t, "pending", r.Delivery)
	require.NoError(t, drainer.DrainOnce(ctx, now.Add(time.Minute)))
	r, err = s.Get(ctx, "completed")
	require.NoError(t, err)
	require.Equal(t, "pending", r.Delivery, "kernel success without capture acknowledgement cannot deliver")
	require.NoError(t, drainer.DrainOnce(ctx, now.Add(2*time.Minute)))
	require.NoError(t, drainer.DrainOnce(ctx, now.Add(3*time.Minute)))
	require.Len(t, intents, 3)
	require.Equal(t, intents[0], intents[1])
	require.Equal(t, intents[0], intents[2])
	r, err = s.Get(ctx, "completed")
	require.NoError(t, err)
	require.Equal(t, "delivered", r.Delivery)
	require.Equal(t, "verified_success", r.Outcome)
}

func TestPermanentCaptureFailureBlocksWithoutRetryingDomainOrDroppingReceipts(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "tasks.db")))
	r := seedRecord()
	_, _, err := s.Begin(ctx, r)
	require.NoError(t, err)
	_, err = s.Update(ctx, r.AttemptID, func(r *Record) error {
		r.State, r.Outcome, r.Delivery = "completed", "verified_success", "pending"
		r.FinishInputs = map[string]any{"attempt": map[string]any{"attempt_id": r.AttemptID}}
		return nil
	})
	require.NoError(t, err)
	calls := 0
	partial := map[string]any{
		"status": "partial", "signals": map[string]any{"entry_id": "acknowledged-attempt"},
		"errors": []any{map[string]any{"class": "capture_failed", "cause": "immutable_conflict"}},
	}
	d := &Drainer{Store: s, Deliver: func(context.Context, Record) (map[string]any, error) { calls++; return partial, nil }}
	require.NoError(t, d.DrainOnce(ctx, time.Now()))
	require.NoError(t, d.DrainOnce(ctx, time.Now().Add(time.Hour)))
	got, err := s.Get(ctx, r.AttemptID)
	require.NoError(t, err)
	require.Equal(t, "blocked", got.Delivery)
	require.Equal(t, "verified_success", got.Outcome)
	require.Equal(t, partial, got.DeliveryResult)
	require.Equal(t, 1, calls)
}

// [REQ:LV-10] Duplicate receipts and identical inputs do not inflate qualification.
func TestFragmentEvidenceIsIdempotentDiverseAndDurable(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "evidence.db")
	db := taskDB(t, path)
	s := NewStore(db)
	f := Fragment{StepKey: "fragment-v2:fixture", Fragment: "def step(inputs, bindings):\n    return inputs", AttemptID: "one", InputDigest: "input-a", Compatibility: map[string]any{"verifier_revision": "v1"}, Evidence: []string{"postcondition:equal"}}
	_, err := s.PutFragment(ctx, f, 1, 0)
	require.NoError(t, err)
	_, err = s.PutFragment(ctx, f, 1, 0)
	require.NoError(t, err)
	got, err := s.BestFragment(ctx, f.StepKey)
	require.NoError(t, err)
	require.Equal(t, 1, got.Verified)
	require.Equal(t, 1, got.Contexts)
	f.AttemptID = "two"
	_, err = s.PutFragment(ctx, f, 1, 0)
	require.NoError(t, err)
	f.AttemptID = "three"
	f.InputDigest = "input-b"
	_, err = s.PutFragment(ctx, f, 1, 0)
	require.NoError(t, err)
	fresh := NewStore(taskDB(t, path))
	got, err = fresh.BestFragment(ctx, f.StepKey)
	require.NoError(t, err)
	require.Equal(t, 3, got.Verified)
	require.Equal(t, 2, got.Contexts)
	require.NotEmpty(t, got.LastVerifiedAt)
	require.Equal(t, f.Compatibility, got.Compatibility)
	f.AttemptID = "failed"
	_, err = fresh.PutFragment(ctx, f, 0, 1)
	require.NoError(t, err)
	_, err = fresh.BestFragment(ctx, f.StepKey)
	require.ErrorIs(t, err, sql.ErrNoRows)
	_, err = fresh.PutFragment(ctx, f, -1, 0)
	require.Error(t, err)
}
