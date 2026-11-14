package tasks

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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

func TestTaskResultRetainsLargeIntegersAcrossCheckpointReplay(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "tasks.db")))
	_, _, err := s.Begin(ctx, seedRecord())
	require.NoError(t, err)
	_, err = s.Update(ctx, "attempt", func(r *Record) error {
		r.State = "completed"
		r.Result = DomainResult{"status": "ok", "signals": map[string]any{"count": json.Number("9007199254740993")}}
		return nil
	})
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		r, err := s.Get(ctx, "attempt")
		require.NoError(t, err)
		encoded, err := json.Marshal(r.Result)
		require.NoError(t, err)
		require.Contains(t, string(encoded), `"count":9007199254740993`)
	}
}

func TestTaskRecordSealsReceiptsAndMakesRestartLossExplicit(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	s := NewStore(db)
	r := seedRecord()
	_, _, err := s.Begin(ctx, r)
	require.NoError(t, err)
	token := "prt_resume_v1_" + strings.Repeat("A", 43)
	_, err = s.Update(ctx, r.AttemptID, func(record *Record) error {
		record.State, record.Delivery = "completed", "pending"
		record.Result = DomainResult{"evidence": []any{"artifact:1"}, "learning": map[string]any{"resume_token": token}}
		record.Prepare = map[string]any{"message": "receipt " + token, "literal": "prt_sealed_v1_" + strings.Repeat("A", 64)}
		record.FinishInputs = map[string]any{"note": token}
		return nil
	})
	require.NoError(t, err)
	var raw string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT record FROM learning_tasks WHERE attempt_id=?", r.AttemptID).Scan(&raw))
	require.NotContains(t, raw, token)
	require.NotContains(t, raw, "prt_resume_v1_")
	live, err := s.Get(ctx, r.AttemptID)
	require.NoError(t, err)
	require.Equal(t, token, live.Result["learning"].(map[string]any)["resume_token"])
	require.Equal(t, token, live.FinishInputs["note"])
	require.Equal(t, "prt_sealed_v1_"+strings.Repeat("A", 64), live.Prepare["literal"])
	pending, err := s.Pending(ctx, time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, token, pending[0].FinishInputs["note"])
	restarted, err := NewStore(db).Get(ctx, r.AttemptID)
	require.NoError(t, err)
	require.Equal(t, "[resume receipt unavailable after restart]", restarted.Result["learning"].(map[string]any)["resume_token"])
	require.Equal(t, []any{"artifact:1"}, restarted.Result["evidence"])
	require.Equal(t, "prt_sealed_v1_"+strings.Repeat("A", 64), restarted.Prepare["literal"])
	called := false
	drainer := Drainer{Store: NewStore(db), Deliver: func(context.Context, Record) (map[string]any, error) {
		called = true
		return nil, nil
	}}
	require.ErrorContains(t, drainer.DrainOnce(ctx, time.Now().Add(time.Hour)), "owner reconciliation required")
	require.False(t, called, "never deliver a substituted receipt")
	var after string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT record FROM learning_tasks WHERE attempt_id=?", r.AttemptID).Scan(&after))
	require.Equal(t, raw, after, "unavailable immutable intent must not be overwritten")
}

func TestLiteralReceiptMarkersDoNotBlockHealthyTask(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	s := NewStore(db)
	r := seedRecord()
	_, _, err := s.Begin(ctx, r)
	require.NoError(t, err)
	literal := "prt_sealed_v1_" + strings.Repeat("A", 64)
	marker := "[resume receipt unavailable after restart]"
	_, err = s.Update(ctx, r.AttemptID, func(record *Record) error {
		record.Result = DomainResult{"literal": literal, "message": marker}
		record.FinishInputs = map[string]any{"literal": literal, "message": marker}
		return nil
	})
	require.NoError(t, err)
	restarted := NewStore(db)
	got, err := restarted.Get(ctx, r.AttemptID)
	require.NoError(t, err)
	require.Equal(t, literal, got.Result["literal"])
	require.Equal(t, marker, got.Result["message"])
	_, err = restarted.Update(ctx, r.AttemptID, func(record *Record) error { record.State = "completed"; return nil })
	require.NoError(t, err)
	// Untagged legacy rows are plain JSON, including token-like user text.
	legacy, err := json.Marshal(Record{AttemptID: r.AttemptID, State: "prepared", Delivery: "waiting", Result: DomainResult{"literal": literal, "message": marker}})
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "UPDATE learning_tasks SET record=? WHERE attempt_id=?", string(legacy), r.AttemptID)
	require.NoError(t, err)
	got, err = restarted.Get(ctx, r.AttemptID)
	require.NoError(t, err)
	require.Equal(t, literal, got.Result["literal"])
	require.Equal(t, marker, got.Result["message"])
	_, err = restarted.Update(ctx, r.AttemptID, func(record *Record) error { record.State = "completed"; return nil })
	require.NoError(t, err)
}

func TestRecoveryContinuesPastReceiptUnavailable(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	s := NewStore(db)
	for _, id := range []string{"first", "second"} {
		r := seedRecord()
		r.TaskID, r.AttemptID = id, id
		_, _, err := s.Begin(ctx, r)
		require.NoError(t, err)
		_, err = s.Update(ctx, id, func(record *Record) error {
			record.State = "running"
			if id == "first" {
				record.Prepare = map[string]any{"resume_token": "prt_resume_v1_" + strings.Repeat("A", 43)}
			}
			return nil
		})
		require.NoError(t, err)
	}
	restarted := NewStore(db)
	require.ErrorContains(t, restarted.Recover(ctx, ""), "owner reconciliation required")
	first, err := restarted.Get(ctx, "first")
	require.NoError(t, err)
	require.Equal(t, "running", first.State)
	second, err := restarted.Get(ctx, "second")
	require.NoError(t, err)
	require.Equal(t, "uncertain", second.State)
}

func TestOutboxFairlyScansPastUnavailableReceipts(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	old := NewStore(db)
	now := time.Now().UTC()
	due := now.Add(-time.Minute).Format(time.RFC3339Nano)
	add := func(s *Store, id string, secret bool) {
		r := seedRecord()
		r.TaskID, r.AttemptID = id, id
		_, _, err := s.Begin(ctx, r)
		require.NoError(t, err)
		_, err = s.Update(ctx, id, func(record *Record) error {
			record.State, record.Delivery, record.NextAttemptAt = "completed", "pending", due
			record.FinishInputs = map[string]any{"id": id}
			if secret {
				record.FinishInputs["token"] = "prt_resume_v1_" + strings.Repeat("A", 43)
			}
			return nil
		})
		require.NoError(t, err)
	}
	for i := 0; i < 25; i++ {
		add(old, fmt.Sprintf("bad-%02d", i), true)
	}
	add(old, "z-good", false)
	var before string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT record FROM learning_tasks WHERE attempt_id='bad-00'").Scan(&before))
	restarted := NewStore(db)
	var delivered []string
	drainer := &Drainer{Store: restarted, Deliver: func(_ context.Context, r Record) (map[string]any, error) {
		delivered = append(delivered, r.AttemptID)
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
	}}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	drainGate <- struct{}{}
	cancelErr := drainer.DrainOnce(cancelled, now)
	<-drainGate
	require.ErrorIs(t, cancelErr, context.Canceled)
	require.Empty(t, delivered)
	require.ErrorContains(t, drainer.DrainOnce(ctx, now), "owner reconciliation required")
	require.Empty(t, delivered)
	// A replacement owner resumes the persisted cursor and warning throttle.
	replacement := &Drainer{Store: NewStore(db), Deliver: drainer.Deliver}
	// A newly due item before the cursor waits for the next bounded scan cycle.
	add(restarted, "a-new", false)
	// A newly inserted item inside the high-watermark range also waits.
	add(restarted, "mid-new", false)
	require.NoError(t, replacement.DrainOnce(ctx, now.Add(time.Second)))
	require.Equal(t, []string{"z-good"}, delivered)
	require.NoError(t, drainer.DrainOnce(ctx, now.Add(2*time.Second)))
	require.Equal(t, []string{"z-good", "a-new"}, delivered)
	require.NoError(t, drainer.DrainOnce(ctx, now.Add(3*time.Second)))
	require.Equal(t, []string{"z-good", "a-new", "mid-new"}, delivered)
	var after string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT record FROM learning_tasks WHERE attempt_id='bad-00'").Scan(&after))
	require.Equal(t, before, after)
}

func TestOutboxCheckpointSurvivesRepeatedOwnerRecreation(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	old := NewStore(db)
	now := time.Now().UTC()
	for i := 0; i < 42; i++ {
		id := fmt.Sprintf("item-%02d", i)
		r := seedRecord()
		r.TaskID, r.AttemptID = id, id
		_, _, err := old.Begin(ctx, r)
		require.NoError(t, err)
		_, err = old.Update(ctx, id, func(record *Record) error {
			record.State, record.Delivery = "completed", "pending"
			record.NextAttemptAt = now.Add(-time.Minute).Format(time.RFC3339Nano)
			if i < 41 {
				record.Result = DomainResult{"token": "prt_resume_v1_" + strings.Repeat("A", 43)}
			}
			return nil
		})
		require.NoError(t, err)
	}
	var delivered []string
	for i := 0; i < 3; i++ {
		d := &Drainer{Store: NewStore(db), Deliver: func(_ context.Context, r Record) (map[string]any, error) {
			delivered = append(delivered, r.AttemptID)
			return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
		}}
		err := d.DrainOnce(ctx, now.Add(time.Duration(i)*time.Second))
		if i == 0 {
			require.ErrorContains(t, err, "owner reconciliation required")
		} else {
			require.NoError(t, err)
		}
	}
	require.Equal(t, []string{"item-41"}, delivered)
}

func TestOutboxRetainsUnavailableObservationAcrossInterruptedPage(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	old := NewStore(db)
	now := time.Now().UTC()
	for _, id := range []string{"a-unavailable", "z-healthy"} {
		r := seedRecord()
		r.TaskID, r.AttemptID = id, id
		_, _, err := old.Begin(ctx, r)
		require.NoError(t, err)
		_, err = old.Update(ctx, id, func(record *Record) error {
			record.State, record.Delivery = "completed", "pending"
			record.NextAttemptAt = now.Add(-time.Minute).Format(time.RFC3339Nano)
			if id == "a-unavailable" {
				record.Result = DomainResult{"token": "prt_resume_v1_" + strings.Repeat("A", 43)}
			}
			return nil
		})
		require.NoError(t, err)
	}
	cancelCtx, cancel := context.WithCancel(ctx)
	first := &Drainer{Store: NewStore(db), Deliver: func(context.Context, Record) (map[string]any, error) {
		cancel()
		return nil, context.Canceled
	}}
	require.ErrorIs(t, first.DrainOnce(cancelCtx, now), context.Canceled)
	called := false
	restarted := &Drainer{Store: NewStore(db), Deliver: func(context.Context, Record) (map[string]any, error) {
		called = true
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
	}}
	require.ErrorContains(t, restarted.DrainOnce(ctx, now.Add(time.Second)), "receipt unavailable after restart")
	require.True(t, called, "healthy sibling remains deliverable after restart")
}

func TestOutboxRescheduledItemsReturnInLaterCycle(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "tasks.db")))
	r := seedRecord()
	_, _, err := s.Begin(ctx, r)
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = s.Update(ctx, r.AttemptID, func(record *Record) error {
		record.State, record.Delivery = "completed", "pending"
		record.NextAttemptAt = now.Add(-time.Second).Format(time.RFC3339Nano)
		return nil
	})
	require.NoError(t, err)
	calls := 0
	d := &Drainer{Store: s, Deliver: func(context.Context, Record) (map[string]any, error) {
		calls++
		if calls == 1 {
			return nil, errors.New("temporary")
		}
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
	}}
	require.NoError(t, d.DrainOnce(ctx, now))
	require.NoError(t, d.DrainOnce(ctx, now.Add(time.Second)))
	require.Equal(t, 1, calls)
	require.NoError(t, d.DrainOnce(ctx, now.Add(3*time.Second)))
	require.Equal(t, 2, calls)
}

func TestOutboxDueTimesOrderWholeAndFractionalSeconds(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "tasks.db")))
	base := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for _, item := range []struct{ id, timestamp string }{
		{"whole", "2026-10-30T12:00:00Z"},
		{"fraction", "2026-10-30T12:00:00.5Z"},
	} {
		r := seedRecord()
		r.TaskID, r.AttemptID = item.id, item.id
		_, _, err := s.Begin(ctx, r)
		require.NoError(t, err)
		_, err = s.Update(ctx, item.id, func(record *Record) error {
			record.State, record.Delivery, record.NextAttemptAt = "completed", "pending", item.timestamp
			return nil
		})
		require.NoError(t, err)
	}
	pending, err := s.Pending(ctx, base.Add(100*time.Millisecond))
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, "whole", pending[0].AttemptID)
	var delivered []string
	d := &Drainer{Store: s, Deliver: func(_ context.Context, r Record) (map[string]any, error) {
		delivered = append(delivered, r.AttemptID)
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
	}}
	require.NoError(t, d.DrainOnce(ctx, base.Add(100*time.Millisecond)))
	require.Equal(t, []string{"whole"}, delivered)
	require.NoError(t, d.DrainOnce(ctx, base.Add(600*time.Millisecond)))
	require.Equal(t, []string{"whole", "fraction"}, delivered)
}

func TestMalformedOrNonUTCDueTimesAreReportedWithoutDelivery(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "tasks.db")))
	now := time.Date(2026, 10, 30, 12, 0, 1, 0, time.UTC)
	for _, item := range []struct{ id, timestamp string }{
		{"good", "2026-10-30T12:00:00Z"},
		{"bad-date", "2026-02-30T12:00:00Z"},
		{"bad-month", "2026-99-30T12:00:00Z"},
		{"bad-fraction", "2026-10-30T12:00:00.xyzZ"},
		{"offset", "2026-10-30T11:00:00+01:00"},
	} {
		r := seedRecord()
		r.TaskID, r.AttemptID = item.id, item.id
		_, _, err := s.Begin(ctx, r)
		require.NoError(t, err)
		_, err = s.Update(ctx, item.id, func(record *Record) error {
			record.State, record.Delivery, record.NextAttemptAt = "completed", "pending", item.timestamp
			return nil
		})
		require.NoError(t, err)
	}
	var delivered []string
	d := &Drainer{Store: s, Deliver: func(_ context.Context, r Record) (map[string]any, error) {
		delivered = append(delivered, r.AttemptID)
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
	}}
	require.ErrorContains(t, d.DrainOnce(ctx, now), "4 pending task(s) and 0 feedback item(s) have invalid or non-UTC due times")
	require.Equal(t, []string{"good"}, delivered)
}

func TestDrainersSharingOwnerSerializeCheckpointAndDelivery(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "tasks.db")))
	r := seedRecord()
	_, _, err := s.Begin(ctx, r)
	require.NoError(t, err)
	now := time.Now().UTC()
	_, err = s.Update(ctx, r.AttemptID, func(record *Record) error {
		record.State, record.Delivery = "completed", "pending"
		record.NextAttemptAt = now.Add(-time.Second).Format(time.RFC3339Nano)
		return nil
	})
	require.NoError(t, err)
	var calls atomic.Int32
	deliver := func(context.Context, Record) (map[string]any, error) {
		calls.Add(1)
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
	}
	drainers := []*Drainer{{Store: s, Deliver: deliver}, {Store: NewStore(s.db), Deliver: deliver}}
	var wg sync.WaitGroup
	errs := make(chan error, len(drainers))
	for _, d := range drainers {
		wg.Add(1)
		go func(d *Drainer) { defer wg.Done(); errs <- d.DrainOnce(ctx, now) }(d)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, calls.Load())
}

func TestInvalidOutboxCheckpointFailsClosed(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	s := NewStore(db)
	_, err := db.ExecContext(ctx, `INSERT INTO learning_task_outbox_scan(singleton, cutoff, watermark_at, watermark_id, watermark_rowid, cursor_at, cursor_id) VALUES(1, '2026-10-30T12:00:00.000000000Z', '2026-10-30T12:00:00.000000000Z', 'a', 1, '2026-10-30T12:00:00.000000000Z', 'z')`)
	require.NoError(t, err)
	called := false
	d := &Drainer{Store: s, Deliver: func(context.Context, Record) (map[string]any, error) { called = true; return nil, nil }}
	require.ErrorContains(t, d.DrainOnce(ctx, time.Now()), "cursor exceeds watermark")
	require.False(t, called)
}

func TestOutboxCancellationStopsRemainingPage(t *testing.T) {
	ctx := context.Background()
	s := NewStore(taskDB(t, filepath.Join(t.TempDir(), "tasks.db")))
	now := time.Now().UTC()
	for _, id := range []string{"first", "second"} {
		r := seedRecord()
		r.TaskID, r.AttemptID = id, id
		_, _, err := s.Begin(ctx, r)
		require.NoError(t, err)
		_, err = s.Update(ctx, id, func(record *Record) error {
			record.State, record.Delivery = "completed", "pending"
			record.NextAttemptAt = now.Add(-time.Second).Format(time.RFC3339Nano)
			return nil
		})
		require.NoError(t, err)
	}
	cancellable, cancel := context.WithCancel(ctx)
	var calls []string
	d := &Drainer{Store: s, Deliver: func(_ context.Context, r Record) (map[string]any, error) {
		calls = append(calls, r.AttemptID)
		cancel()
		return nil, context.Canceled
	}}
	require.ErrorIs(t, d.DrainOnce(cancellable, now), context.Canceled)
	require.Equal(t, []string{"first"}, calls)
	second, err := s.Get(ctx, "second")
	require.NoError(t, err)
	require.Equal(t, "pending", second.Delivery)
}

func TestUnavailableRequeueDoesNotBlockHealthyOutbox(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	old := NewStore(db)
	now := time.Now().UTC()
	for _, id := range []string{"blocked", "healthy"} {
		r := seedRecord()
		r.TaskID, r.AttemptID = id, id
		_, _, err := old.Begin(ctx, r)
		require.NoError(t, err)
		_, err = old.Update(ctx, id, func(record *Record) error {
			record.State, record.NextAttemptAt = "completed", now.Add(-time.Second).Format(time.RFC3339Nano)
			record.FinishInputs = map[string]any{"scope": "fixture"}
			if id == "blocked" {
				record.Delivery, record.FinishDigest, record.LastError = "blocked", "", "memory_not_installed"
				record.Result = DomainResult{"resume_token": "prt_resume_v1_" + strings.Repeat("A", 43)}
			} else {
				record.Delivery = "pending"
			}
			return nil
		})
		require.NoError(t, err)
	}
	restarted := NewStore(db)
	var delivered []string
	d := &Drainer{Store: restarted, ResolveFinishDigest: func(context.Context) (string, error) { return "finish", nil },
		Deliver: func(_ context.Context, record Record) (map[string]any, error) {
			delivered = append(delivered, record.AttemptID)
			return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
		}}
	require.ErrorContains(t, d.DrainOnce(ctx, now), "owner reconciliation required")
	require.Equal(t, []string{"healthy"}, delivered)
	blocked, err := restarted.Get(ctx, "blocked")
	require.NoError(t, err)
	require.Equal(t, "blocked", blocked.Delivery)
}

func TestFindingRoutingFailureDoesNotBlockHealthyOutbox(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	s := NewStore(db)
	r := seedRecord()
	_, _, err := s.Begin(ctx, r)
	require.NoError(t, err)
	_, err = s.Update(ctx, r.AttemptID, func(record *Record) error {
		record.State, record.Delivery = "completed", "pending"
		record.NextAttemptAt = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano)
		return nil
	})
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, "DROP TABLE learning_findings")
	require.NoError(t, err)
	called := false
	d := &Drainer{Store: s, Deliver: func(context.Context, Record) (map[string]any, error) {
		called = true
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
	}}
	require.Error(t, d.DrainOnce(ctx, time.Now()))
	require.True(t, called)
}

func TestUnpinnedRequeueScansPastUnavailablePage(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	old := NewStore(db)
	for i := 0; i < 101; i++ {
		id := fmt.Sprintf("blocked-%03d", i)
		r := seedRecord()
		r.TaskID, r.AttemptID = id, id
		_, _, err := old.Begin(ctx, r)
		require.NoError(t, err)
		_, err = old.Update(ctx, id, func(record *Record) error {
			record.State, record.Delivery, record.FinishDigest, record.LastError = "completed", "blocked", "", "memory_not_installed"
			record.FinishInputs = map[string]any{"scope": "fixture"}
			if i < 100 {
				record.Result = DomainResult{"token": "prt_resume_v1_" + strings.Repeat("A", 43)}
			}
			return nil
		})
		require.NoError(t, err)
	}
	restarted := NewStore(db)
	cursor, err := restarted.requeueUnpinnedPage(ctx, "finish", "")
	require.ErrorContains(t, err, "owner reconciliation required")
	require.NotEmpty(t, cursor)
	cursor, err = restarted.requeueUnpinnedPage(ctx, "finish", cursor)
	require.NoError(t, err)
	require.Empty(t, cursor)
	got, err := restarted.Get(ctx, "blocked-100")
	require.NoError(t, err)
	require.Equal(t, "pending", got.Delivery)
	require.Equal(t, "finish", got.FinishDigest)
}

func TestDurableRequeueResumesPinnedDigestAfterOwnerRestart(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	old := NewStore(db)
	for i := 0; i < 101; i++ {
		id := fmt.Sprintf("item-%03d", i)
		r := seedRecord()
		r.TaskID, r.AttemptID = id, id
		_, _, err := old.Begin(ctx, r)
		require.NoError(t, err)
		_, err = old.Update(ctx, id, func(record *Record) error {
			record.State, record.Delivery, record.FinishDigest, record.LastError = "completed", "blocked", "", "memory_not_installed"
			record.FinishInputs = map[string]any{"scope": "fixture"}
			if i < 100 {
				record.Result = DomainResult{"token": "prt_resume_v1_" + strings.Repeat("A", 43)}
			}
			return nil
		})
		require.NoError(t, err)
	}
	var before string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT record FROM learning_tasks WHERE attempt_id='item-000'").Scan(&before))
	first := NewStore(db)
	now := time.Now().UTC()
	firstDrainer := &Drainer{Store: first, ResolveFinishDigest: func(context.Context) (string, error) { return "digest-a", nil }}
	require.ErrorContains(t, firstDrainer.DrainOnce(ctx, now), "owner reconciliation required")
	second := NewStore(db)
	resolverCalled := false
	drainer := &Drainer{Store: second, ResolveFinishDigest: func(context.Context) (string, error) {
		resolverCalled = true
		return "", errors.New("resolver unavailable")
	}, Deliver: func(context.Context, Record) (map[string]any, error) {
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
	}}
	require.NoError(t, drainer.DrainOnce(ctx, now.Add(time.Second)))
	stillBlocked, err := second.Get(ctx, "item-100")
	require.NoError(t, err)
	require.Equal(t, "blocked", stillBlocked.Delivery, "restart must not bypass the retry throttle")
	require.ErrorContains(t, drainer.DrainOnce(ctx, now.Add(time.Minute)), "owner reconciliation required")
	require.False(t, resolverCalled, "an active pinned cycle must not depend on resolving a new digest")
	got, err := second.Get(ctx, "item-100")
	require.NoError(t, err)
	require.Equal(t, "delivered", got.Delivery)
	require.Equal(t, "digest-a", got.FinishDigest, "restart must not replace the digest selected for the active cycle")
	var after string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT record FROM learning_tasks WHERE attempt_id='item-000'").Scan(&after))
	require.Equal(t, before, after)
}

func TestDurableRequeueSkipsMalformedRecordAndReportsIt(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	s := NewStore(db)
	for _, id := range []string{"malformed", "healthy"} {
		r := seedRecord()
		r.TaskID, r.AttemptID = id, id
		_, _, err := s.Begin(ctx, r)
		require.NoError(t, err)
		_, err = s.Update(ctx, id, func(record *Record) error {
			record.State, record.Delivery, record.FinishDigest, record.LastError = "completed", "blocked", "", "memory_not_installed"
			record.FinishInputs = map[string]any{"scope": "fixture"}
			return nil
		})
		require.NoError(t, err)
	}
	_, err := db.ExecContext(ctx, "UPDATE learning_tasks SET record='{' WHERE attempt_id='malformed'")
	require.NoError(t, err)
	require.ErrorContains(t, s.requeueDurable(ctx, "digest"), "owner reconciliation required")
	got, err := s.Get(ctx, "healthy")
	require.NoError(t, err)
	require.Equal(t, "pending", got.Delivery)
	var raw string
	require.NoError(t, db.QueryRowContext(ctx, "SELECT record FROM learning_tasks WHERE attempt_id='malformed'").Scan(&raw))
	require.Equal(t, "{", raw)
}

func TestDurableRequeueRetainsObservationAfterInterruptedPage(t *testing.T) {
	ctx := context.Background()
	db := taskDB(t, filepath.Join(t.TempDir(), "tasks.db"))
	old := NewStore(db)
	for _, id := range []string{"a-unavailable", "z-healthy"} {
		r := seedRecord()
		r.TaskID, r.AttemptID = id, id
		_, _, err := old.Begin(ctx, r)
		require.NoError(t, err)
		_, err = old.Update(ctx, id, func(record *Record) error {
			record.State, record.Delivery, record.FinishDigest, record.LastError = "completed", "blocked", "", "memory_not_installed"
			record.FinishInputs = map[string]any{"scope": "fixture"}
			if id == "a-unavailable" {
				record.Result = DomainResult{"token": "prt_resume_v1_" + strings.Repeat("A", 43)}
			}
			return nil
		})
		require.NoError(t, err)
	}
	var watermark int64
	require.NoError(t, db.QueryRowContext(ctx, "SELECT max(rowid) FROM learning_tasks").Scan(&watermark))
	// Checkpoint state after inspecting the first inaccessible row and before
	// emitting a warning, as if the owner stopped between those operations.
	require.NoError(t, old.saveRequeueScan(ctx, requeueScan{digest: "original", cursor: "a-unavailable", watermark: "z-healthy", rowid: watermark, unresolved: true, cycleUnresolved: true}))
	restarted := NewStore(db)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, restarted.requeueDurable(cancelled, "newer"), context.Canceled)
	checkpoint, err := restarted.loadRequeueScan(ctx)
	require.NoError(t, err)
	require.Equal(t, "a-unavailable", checkpoint.cursor)
	drainer := &Drainer{Store: restarted, Deliver: func(context.Context, Record) (map[string]any, error) {
		return map[string]any{"status": "ok", "signals": map[string]any{"capture_status": "complete", "entry_id": "ok"}}, nil
	}}
	require.ErrorContains(t, drainer.DrainOnce(ctx, time.Now().UTC()), "owner reconciliation required")
	got, err := restarted.Get(ctx, "z-healthy")
	require.NoError(t, err)
	require.Equal(t, "pending", got.Delivery)
	require.Equal(t, "original", got.FinishDigest)
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
