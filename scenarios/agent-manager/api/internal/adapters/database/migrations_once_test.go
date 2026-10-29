package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func newMigrationTestDB(t *testing.T) *DB {
	t.Helper()
	t.Setenv("VROOLI_STORAGE_ROOT", t.TempDir())
	db, err := NewConnection(logrus.New())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func countWhere(t *testing.T, db *DB, query string, args ...any) int {
	t.Helper()
	var n int
	if err := db.GetContext(context.Background(), &n, query, args...); err != nil {
		t.Fatal(err)
	}
	return n
}

const legacyQuotaFrame = `{"payloadKind":"quota","limitType":"five_hour","provider":"anthropic"}`

func insertLegacyQuotaEvent(t *testing.T, db *DB) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO run_events (id, run_id, sequence, event_type, data) VALUES (?, ?, 1, 'error', ?)`, id, uuid.NewString(), legacyQuotaFrame); err != nil {
		t.Fatal(err)
	}
	return id
}

func insertFalselyFailedAttachedRun(t *testing.T, db *DB) string {
	t.Helper()
	id := uuid.NewString()
	if _, err := db.Exec(`INSERT INTO runs (id, execution_mode, status, error_msg) VALUES (?, 'attached', 'failed', ?)`, id, attachedExitReason); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestQuotaEventMigrationRunsOncePerDatabase(t *testing.T) {
	db := newMigrationTestDB(t)
	ctx := context.Background()
	first := insertLegacyQuotaEvent(t, db)
	if _, err := db.Exec(`DELETE FROM schema_migrations_done WHERE name = ?`, migrationQuotaEvents); err != nil {
		t.Fatal(err)
	}
	if err := db.runOnceMigrations(ctx); err != nil {
		t.Fatal(err)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM run_events WHERE id = ? AND event_type = 'quota'`, first); got != 1 {
		t.Fatalf("first run did not retype the legacy frame: %d", got)
	}
	// A legacy-looking row written after the marker is not scanned again.
	second := insertLegacyQuotaEvent(t, db)
	if err := db.InitializeSchema(); err != nil {
		t.Fatal(err)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM run_events WHERE id = ? AND event_type = 'error'`, second); got != 1 {
		t.Fatalf("second start rescanned run_events: %d", got)
	}
}

func TestAttachedExitStatusMigrationRunsOncePerDatabase(t *testing.T) {
	db := newMigrationTestDB(t)
	ctx := context.Background()
	first := insertFalselyFailedAttachedRun(t, db)
	if _, err := db.Exec(`DELETE FROM schema_migrations_done WHERE name = ?`, migrationAttachedExitStatus); err != nil {
		t.Fatal(err)
	}
	if err := db.runOnceMigrations(ctx); err != nil {
		t.Fatal(err)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM runs WHERE id = ? AND status = 'complete'`, first); got != 1 {
		t.Fatalf("first run did not complete the attached run: %d", got)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM invocation_read_model_pending WHERE run_id = ?`, first); got != 1 {
		t.Fatalf("corrected run was not queued for re-projection: %d", got)
	}
	second := insertFalselyFailedAttachedRun(t, db)
	if err := db.InitializeSchema(); err != nil {
		t.Fatal(err)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM runs WHERE id = ? AND status = 'failed'`, second); got != 1 {
		t.Fatalf("second start rescanned runs: %d", got)
	}
}

func TestRetiredBookkeepingTablesAreDroppedOnce(t *testing.T) {
	db := newMigrationTestDB(t)
	for _, table := range []string{"stats_checkpoint", "transcript_usage_repairs"} {
		if _, err := db.Exec(`CREATE TABLE ` + table + ` (name TEXT)`); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`DELETE FROM schema_migrations_done WHERE name = ?`, migrationDropRetiredBookkeep); err != nil {
		t.Fatal(err)
	}
	if err := db.InitializeSchema(); err != nil {
		t.Fatal(err)
	}
	if got := countWhere(t, db, `SELECT COUNT(*) FROM sqlite_master WHERE name IN ('stats_checkpoint','transcript_usage_repairs')`); got != 0 {
		t.Fatalf("retired tables remain: %d", got)
	}
}
