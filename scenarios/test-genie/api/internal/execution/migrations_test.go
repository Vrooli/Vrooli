package execution

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// oldSuiteExecutionsSchema is the pre-terminal_outcome table shape, used to
// simulate an existing populated database that predates the column.
const oldSuiteExecutionsSchema = `
CREATE TABLE suite_executions (
    id TEXT PRIMARY KEY,
    suite_request_id TEXT,
    scenario_name TEXT NOT NULL,
    preset_used TEXT,
    requested_preset TEXT,
    requested_phases TEXT NOT NULL DEFAULT '[]',
    requested_skip_phases TEXT NOT NULL DEFAULT '[]',
    planned_phases TEXT NOT NULL DEFAULT '[]',
    fail_fast INTEGER NOT NULL DEFAULT 0,
    success INTEGER NOT NULL,
    phases TEXT NOT NULL,
    started_at TEXT NOT NULL,
    completed_at TEXT NOT NULL
);`

func openOldSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(oldSuiteExecutionsSchema); err != nil {
		t.Fatalf("create old schema: %v", err)
	}
	return db
}

func TestMigrateAddsColumnAndBackfillsIdempotently(t *testing.T) {
	ctx := context.Background()
	db := openOldSchemaDB(t)

	// Seed a passing and a failing row on the old (column-less) table.
	seed := func(id string, success int) {
		if _, err := db.ExecContext(ctx, `
INSERT INTO suite_executions (id, scenario_name, success, phases, started_at, completed_at)
VALUES (?, ?, ?, '[]', '2026-01-01T00:00:00.000Z', '2026-01-01T00:01:00.000Z')`,
			id, "demo", success); err != nil {
			t.Fatalf("seed row %s: %v", id, err)
		}
	}
	seed("11111111-1111-1111-1111-111111111111", 1)
	seed("22222222-2222-2222-2222-222222222222", 0)

	// Column must not exist yet.
	if has, err := columnExists(ctx, db, "suite_executions", "terminal_outcome"); err != nil || has {
		t.Fatalf("precondition: column should be absent (has=%v err=%v)", has, err)
	}

	// Run twice — the guarded migration must be idempotent.
	for i := 0; i < 2; i++ {
		if err := Migrate(ctx, db); err != nil {
			t.Fatalf("Migrate run %d: %v", i+1, err)
		}
	}

	// Column added exactly once.
	if has, err := columnExists(ctx, db, "suite_executions", "terminal_outcome"); err != nil || !has {
		t.Fatalf("column should exist after migration (has=%v err=%v)", has, err)
	}

	// Rows preserved (the table was never recreated).
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM suite_executions`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 rows preserved, got %d", count)
	}

	// Backfill derived terminal_outcome from success.
	assertOutcome := func(id string, want TerminalOutcome) {
		var got string
		if err := db.QueryRowContext(ctx, `SELECT terminal_outcome FROM suite_executions WHERE id = ?`, id).Scan(&got); err != nil {
			t.Fatalf("read outcome %s: %v", id, err)
		}
		if TerminalOutcome(got) != want {
			t.Fatalf("row %s outcome = %q, want %q", id, got, want)
		}
	}
	assertOutcome("11111111-1111-1111-1111-111111111111", TerminalOutcomePassed)
	assertOutcome("22222222-2222-2222-2222-222222222222", TerminalOutcomeFailed)

	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'suite_execution_phases'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("normalized phase table missing after migration (count=%d err=%v)", count, err)
	}
	// The runtime hard cutover deliberately does not decode old JSON phase
	// documents. They are preserved only until the offline archive/rebuild
	// procedure retires the legacy store.
	record, err := NewSuiteExecutionRepository(db).GetByID(ctx, uuid.MustParse("11111111-1111-1111-1111-111111111111"))
	if err != nil {
		t.Fatalf("read migrated header: %v", err)
	}
	if len(record.Phases) != 0 {
		t.Fatalf("legacy phase JSON was read by runtime: %#v", record.Phases)
	}
}
