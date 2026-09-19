package fleet

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

// newStoreDB opens a file-backed routed SQLite handle with the fleet schema
// applied, pool capped at one to mirror production.
func newStoreDB(t *testing.T) *apidb.RoutedDB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fleet.db")
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)"
	db, err := apidb.Open(context.Background(), apidb.Config{
		Driver:       apidb.DriverSQLite,
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return db
}

func TestSQLStoreRoundTrip(t *testing.T) {
	db := newStoreDB(t)
	store := NewSQLStore(db)

	now := time.Unix(1700000000, 0).UTC()
	in := Result{
		ScannedAt: now,
		Entries: []ScenarioEntry{
			{Scenario: "alpha", Engines: []string{"postgres", "redis"}, PrimaryEngine: "postgres", Language: "go", StorageStage: "production", IsolationReady: false, IsolationReason: "seams unwired", NamespaceAdopted: true, HasBackupTarget: false, FindingCount: 2, ErrorCount: 1, AutofixableCount: 1, DataDirBytes: 12, DataDirBudget: 10, DataDirUtil: 1.2, DataDirOverBudget: true, DataDirSeverity: "warning", DataDirPaths: []string{"/tmp/alpha/data"}},
			{Scenario: "beta", Engines: []string{"sqlite"}, PrimaryEngine: "sqlite", Language: "go", StorageStage: "greenfield", IsolationReady: true, NamespaceAdopted: true, HasBackupTarget: true},
		},
	}
	if err := store.Save(context.Background(), in); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.ScenarioCount != 2 {
		t.Fatalf("scenario_count: got %d want 2", got.ScenarioCount)
	}
	if !got.ScannedAt.Equal(now) {
		t.Fatalf("scanned_at: got %v want %v", got.ScannedAt, now)
	}
	// Recomputed aggregates: alpha (postgres, no backup) counts; isolation unready 1.
	if got.NoBackupCount != 1 {
		t.Fatalf("no_backup: got %d want 1", got.NoBackupCount)
	}
	if got.IsolationUnreadyCount != 1 {
		t.Fatalf("isolation_unready: got %d want 1", got.IsolationUnreadyCount)
	}
	if got.FindingCount != 2 {
		t.Fatalf("finding_count: got %d want 2", got.FindingCount)
	}
	if got.DataDirOverBudgetCount != 1 {
		t.Fatalf("data_dir_over_budget_count: got %d want 1", got.DataDirOverBudgetCount)
	}
	alpha := got.Entries[0]
	if alpha.Scenario != "alpha" || len(alpha.Engines) != 2 || alpha.IsolationReady {
		t.Fatalf("alpha round-trip mismatch: %+v", alpha)
	}
	if !alpha.DataDirOverBudget || alpha.DataDirBytes != 12 || alpha.DataDirBudget != 10 || alpha.DataDirSeverity != "warning" || len(alpha.DataDirPaths) != 1 {
		t.Fatalf("alpha data-dir budget round-trip mismatch: %+v", alpha)
	}
}

func TestMigrateSchemaAddsBudgetColumnsToLegacyFleetTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fleet.db")
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)")
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), `CREATE TABLE fleet_entries (
		scenario TEXT PRIMARY KEY,
		engines TEXT NOT NULL DEFAULT '',
		primary_engine TEXT NOT NULL DEFAULT '',
		language TEXT NOT NULL DEFAULT '',
		storage_stage TEXT NOT NULL DEFAULT '',
		isolation_ready INTEGER NOT NULL DEFAULT 0,
		isolation_reason TEXT NOT NULL DEFAULT '',
		namespace_adopted INTEGER NOT NULL DEFAULT 0,
		has_backup_target INTEGER NOT NULL DEFAULT 0,
		finding_count INTEGER NOT NULL DEFAULT 0,
		error_count INTEGER NOT NULL DEFAULT 0,
		autofixable_count INTEGER NOT NULL DEFAULT 0,
		scanned_at TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatalf("seed legacy table: %v", err)
	}
	if _, err := db.ExecContext(context.Background(), `INSERT INTO fleet_entries (scenario, scanned_at) VALUES ('legacy', '2026-07-09T00:00:00Z')`); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	if err := MigrateSchema(context.Background(), db); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := MigrateSchema(context.Background(), db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("ensure after migrate: %v", err)
	}

	var bytes int64
	if err := db.QueryRowContext(context.Background(), `SELECT data_dir_bytes FROM fleet_entries WHERE scenario = 'legacy'`).Scan(&bytes); err != nil {
		t.Fatalf("scan migrated column: %v", err)
	}
	if bytes != 0 {
		t.Fatalf("data_dir_bytes = %d, want default 0", bytes)
	}
}

func TestSQLStoreSaveReplacesSnapshot(t *testing.T) {
	db := newStoreDB(t)
	store := NewSQLStore(db)

	if err := store.Save(context.Background(), Result{Entries: []ScenarioEntry{{Scenario: "old"}}}); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := store.Save(context.Background(), Result{Entries: []ScenarioEntry{{Scenario: "new"}}}); err != nil {
		t.Fatalf("second save: %v", err)
	}
	got, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.ScenarioCount != 1 || got.Entries[0].Scenario != "new" {
		t.Fatalf("snapshot not replaced: %+v", got.Entries)
	}
}

func TestNilStoreIsSafe(t *testing.T) {
	var store *SQLStore
	if err := store.Save(context.Background(), Result{}); err != nil {
		t.Fatalf("nil save: %v", err)
	}
	if _, err := store.Load(context.Background()); err != nil {
		t.Fatalf("nil load: %v", err)
	}
}
