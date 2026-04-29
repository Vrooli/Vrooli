package repository_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/testutil/mocks"
)

// openRawSQLite returns a connected handle WITHOUT applying any
// schema. The schema_version tests need to drive the version state
// directly — using db.NewSQLite (which calls EnsureSchema) would mask
// the cases under test.
func openRawSQLite(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	dsn := path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)"
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	d.SetMaxOpenConns(1)
	t.Cleanup(func() { d.Close() })
	return d
}

func TestEnsureSchema_FreshInitWritesExpectedVersion(t *testing.T) {
	db := openRawSQLite(t)
	clk := mocks.NewFakeClock(time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC))

	if err := repository.EnsureSchema(context.Background(), db, clk); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	var v int
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&v); err != nil {
		t.Fatalf("read version: %v", err)
	}
	if v != repository.ExpectedSchemaVersion {
		t.Errorf("version = %d, want %d", v, repository.ExpectedSchemaVersion)
	}

	var appliedAt string
	if err := db.QueryRow(`SELECT applied_at FROM schema_version WHERE version = ?`, v).Scan(&appliedAt); err != nil {
		t.Fatalf("read applied_at: %v", err)
	}
	if !strings.Contains(appliedAt, "2026-04-29") {
		t.Errorf("applied_at = %q, want it to reflect the fake clock (2026-04-29)", appliedAt)
	}
}

func TestEnsureSchema_IdempotentReinit(t *testing.T) {
	db := openRawSQLite(t)
	clk := mocks.NewFakeClock(time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC))

	// First init writes the version.
	if err := repository.EnsureSchema(context.Background(), db, clk); err != nil {
		t.Fatalf("first EnsureSchema: %v", err)
	}
	// Second init must be a no-op — same row, same value.
	if err := repository.EnsureSchema(context.Background(), db, clk); err != nil {
		t.Fatalf("second EnsureSchema: %v", err)
	}

	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_version`).Scan(&rows); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if rows != 1 {
		t.Errorf("schema_version row count = %d, want 1 (idempotent)", rows)
	}
}

func TestEnsureSchema_RefusesOlderVersionThanExpected(t *testing.T) {
	if repository.ExpectedSchemaVersion < 2 {
		t.Skip("requires ExpectedSchemaVersion >= 2 to simulate forward-only drift")
	}
	db := openRawSQLite(t)
	clk := clock.System{}

	// Apply DDL only, then stamp an older version manually.
	if _, err := db.Exec(repository.SchemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_version(version, applied_at) VALUES (?, ?)`,
		repository.ExpectedSchemaVersion-1, time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		t.Fatalf("seed older version: %v", err)
	}

	err := repository.EnsureSchema(context.Background(), db, clk)
	if err == nil {
		t.Fatal("expected error for older schema version, got nil")
	}
	if !strings.Contains(err.Error(), "forward-only migration missing") {
		t.Errorf("error = %v, want it to mention forward-only migration", err)
	}
}

func TestEnsureSchema_RefusesNewerVersionThanExpected(t *testing.T) {
	db := openRawSQLite(t)
	clk := clock.System{}

	if _, err := db.Exec(repository.SchemaSQL); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_version(version, applied_at) VALUES (?, ?)`,
		repository.ExpectedSchemaVersion+1, time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		t.Fatalf("seed newer version: %v", err)
	}

	err := repository.EnsureSchema(context.Background(), db, clk)
	if err == nil {
		t.Fatal("expected error for newer schema version, got nil")
	}
	if !strings.Contains(err.Error(), "binary is older than database") {
		t.Errorf("error = %v, want it to mention binary older than database", err)
	}
}

func TestEnsureSchema_RejectsNilDeps(t *testing.T) {
	if err := repository.EnsureSchema(context.Background(), nil, clock.System{}); err == nil {
		t.Error("expected error for nil db")
	}
	db := openRawSQLite(t)
	if err := repository.EnsureSchema(context.Background(), db, nil); err == nil {
		t.Error("expected error for nil clock")
	}
}

func TestEnsureSchema_AppliesLegacyMigrations(t *testing.T) {
	db := openRawSQLite(t)
	clk := clock.System{}

	// Simulate a pre-2026 database: full sandboxes schema as it existed
	// before the home-overlay refactor — columns named `driver` (not
	// `driver_id`) and no `home_overlay_state` column. Indexes from the
	// production schema reference owner/reserved_path/idempotency_key,
	// so the seed includes those columns as well to keep CREATE INDEX
	// IF NOT EXISTS valid when EnsureSchema replays the DDL.
	if _, err := db.Exec(`
        CREATE TABLE sandboxes (
            id                  TEXT PRIMARY KEY,
            name                TEXT,
            scope_path          TEXT NOT NULL DEFAULT '/tmp',
            reserved_path       TEXT,
            reserved_paths      TEXT NOT NULL DEFAULT '[]',
            no_lock             INTEGER NOT NULL DEFAULT 0,
            project_root        TEXT NOT NULL DEFAULT '/tmp',
            owner               TEXT,
            owner_type          TEXT NOT NULL DEFAULT 'user',
            status              TEXT NOT NULL DEFAULT 'active',
            error_message       TEXT,
            created_at          TEXT NOT NULL DEFAULT '',
            last_used_at        TEXT NOT NULL DEFAULT '',
            stopped_at          TEXT,
            approved_at         TEXT,
            deleted_at          TEXT,
            driver              TEXT NOT NULL DEFAULT 'overlayfs',
            driver_version      TEXT NOT NULL DEFAULT '1.0',
            lower_dir           TEXT,
            upper_dir           TEXT,
            work_dir            TEXT,
            merged_dir          TEXT,
            size_bytes          INTEGER NOT NULL DEFAULT 0,
            file_count          INTEGER NOT NULL DEFAULT 0,
            active_pids         TEXT NOT NULL DEFAULT '[]',
            session_count       INTEGER NOT NULL DEFAULT 0,
            tags                TEXT NOT NULL DEFAULT '[]',
            metadata            TEXT NOT NULL DEFAULT '{}',
            behavior            TEXT NOT NULL DEFAULT '{}',
            idempotency_key     TEXT UNIQUE,
            version             INTEGER NOT NULL DEFAULT 1,
            updated_at          TEXT NOT NULL DEFAULT '',
            base_commit_hash    TEXT,
            CHECK (status IN ('creating', 'active', 'stopped', 'approved', 'rejected', 'deleted', 'error'))
        );
    `); err != nil {
		t.Fatalf("seed pre-2026 sandboxes table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sandboxes (id, created_at, last_used_at, updated_at) VALUES ('11111111-1111-1111-1111-111111111111', '', '', '')`); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	if err := repository.EnsureSchema(context.Background(), db, clk); err != nil {
		t.Fatalf("EnsureSchema: %v", err)
	}

	// driver column → driver_id rename
	var driverIDExists, driverExists, homeOverlayExists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sandboxes') WHERE name='driver_id'`).Scan(&driverIDExists); err != nil {
		t.Fatalf("probe driver_id: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sandboxes') WHERE name='driver'`).Scan(&driverExists); err != nil {
		t.Fatalf("probe driver: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sandboxes') WHERE name='home_overlay_state'`).Scan(&homeOverlayExists); err != nil {
		t.Fatalf("probe home_overlay_state: %v", err)
	}
	if driverIDExists != 1 {
		t.Errorf("driver_id column missing after migration")
	}
	if driverExists != 0 {
		t.Errorf("legacy driver column still present after rename")
	}
	if homeOverlayExists != 1 {
		t.Errorf("home_overlay_state column missing after migration")
	}

	// driver_id backfill: 'overlayfs' → 'overlayfs-userns'
	var driverID string
	if err := db.QueryRow(`SELECT driver_id FROM sandboxes WHERE id = '11111111-1111-1111-1111-111111111111'`).Scan(&driverID); err != nil {
		t.Fatalf("read driver_id: %v", err)
	}
	if driverID != "overlayfs-userns" {
		t.Errorf("driver_id = %q, want overlayfs-userns", driverID)
	}

	// schema_version row written
	var v int
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_version`).Scan(&v); err != nil {
		t.Fatalf("read version: %v", err)
	}
	if v != repository.ExpectedSchemaVersion {
		t.Errorf("version = %d, want %d", v, repository.ExpectedSchemaVersion)
	}
}
