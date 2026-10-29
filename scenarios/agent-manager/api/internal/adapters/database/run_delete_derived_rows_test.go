package database

import (
	"context"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// seedDerivedRows gives a run the rows other owners derive from it.
func seedDerivedRows(t *testing.T, db *DB, runID string) {
	t.Helper()
	ctx := context.Background()
	for _, stmt := range []string{
		`INSERT INTO invocation_read_model_runs (run_id, occurred_at, created_at, status) VALUES (?, '2026-10-29T00:00:00Z', '2026-10-29T00:00:00Z', 'complete')`,
		`INSERT INTO invocation_read_model_facts (run_id, call_event_id, occurred_at, tool_name, ownership, outcome, fingerprint, availability, classifier_version, run_status) VALUES (?, 'call-1', '2026-10-29T00:00:00Z', 'Bash', 'agent', 'ok', 'fp', 'available', 'v1', 'complete')`,
		`INSERT INTO transcript_files (file_key, run_id, path, root_kind, root_path, harness, first_seen_at) VALUES ('key-' || ?1, ?1, '/tmp/x.jsonl', 'codex', '/tmp', 'codex', '2026-10-29T00:00:00Z')`,
	} {
		if _, err := db.DB.ExecContext(ctx, stmt, runID); err != nil {
			t.Fatalf("seed %q: %v", stmt, err)
		}
	}
}

func derivedRowCount(t *testing.T, db *DB, runID string) int {
	t.Helper()
	total := 0
	for _, table := range append(append([]string(nil), runProjectionTables...), "transcript_files") {
		var n int
		if err := db.DB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM `+table+` WHERE run_id = ?`, runID).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		total += n
	}
	return total
}

func TestDeletingARunOrItsTaskRemovesItsDerivedRows(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	task := &domain.Task{ID: uuid.New(), Title: "t", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	deleted, kept := pendingTaskRun(task.ID), pendingTaskRun(task.ID)
	deleted.Status, kept.Status = domain.RunStatusComplete, domain.RunStatusComplete
	for _, run := range []*domain.Run{deleted, kept} {
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		seedDerivedRows(t, db, run.ID.String())
	}
	stored, err := repos.Runs.Get(ctx, deleted.ID)
	if err != nil || stored == nil {
		t.Fatal(err)
	}
	if err := repos.Runs.Delete(ctx, deleted.ID, stored.Status, stored.LifecycleVersion); err != nil {
		t.Fatal(err)
	}
	if n := derivedRowCount(t, db, deleted.ID.String()); n != 0 {
		t.Fatalf("deleted run kept %d derived rows", n)
	}
	if n := derivedRowCount(t, db, kept.ID.String()); n != 3 {
		t.Fatalf("other run lost derived rows: %d left", n)
	}

	if _, err := db.DB.ExecContext(ctx, `UPDATE tasks SET status = ? WHERE id = ?`, string(domain.TaskStatusCancelled), task.ID.String()); err != nil {
		t.Fatal(err)
	}
	if err := repos.Tasks.Delete(ctx, task.ID); err != nil {
		t.Fatal(err)
	}
	if n := derivedRowCount(t, db, kept.ID.String()); n != 0 {
		t.Fatalf("task delete cascaded the run but kept %d derived rows", n)
	}
}

func TestOrphanRepairRemovesOnlyDeletedRunsRowsAndCompletes(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "t", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	live := pendingTaskRun(task.ID)
	if err := repos.Runs.Create(ctx, live); err != nil {
		t.Fatal(err)
	}
	seedDerivedRows(t, db, live.ID.String())
	orphan := uuid.NewString() // rows left by a run deleted before the fix
	seedDerivedRows(t, db, orphan)

	if err := db.RepairOrphanRunProjections(ctx); err != nil {
		t.Fatal(err)
	}
	if n := derivedRowCount(t, db, orphan); n != 0 {
		t.Fatalf("orphan kept %d rows", n)
	}
	if n := derivedRowCount(t, db, live.ID.String()); n != 3 {
		t.Fatalf("live run lost rows: %d left", n)
	}
	if done, err := db.duplicateRepairDone(ctx, migrationOrphanRunProjections); err != nil || !done {
		t.Fatalf("marker not written: done=%v err=%v", done, err)
	}
}
