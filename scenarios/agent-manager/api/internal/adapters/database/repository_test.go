package database

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestDB creates a fresh SQLite database for testing.
// Returns the DB wrapper and a cleanup function.
func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	// Unit repositories need isolated durable semantics, not host disk fsyncs.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared&_pragma=foreign_keys(ON)&_pragma=busy_timeout(10000)", uuid.NewString())

	sqlDB, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}

	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(logrus.PanicLevel) // Suppress logs during tests

	wrapped := &DB{
		DB:  sqlDB,
		log: log,
	}
	if err := wrapped.initSchema(); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("init schema: %v", err)
	}

	return wrapped, func() {
		_ = sqlDB.Close()
	}
}

func TestInitSchemaCreatesRoleOnlyProfileTable(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	var roleRefColumns int
	if err := db.Get(&roleRefColumns, "SELECT COUNT(*) FROM pragma_table_info('agent_profiles') WHERE name = 'role_ref'"); err != nil {
		t.Fatalf("inspect role_ref: %v", err)
	}
	if roleRefColumns != 1 {
		t.Fatalf("role_ref columns = %d, want 1", roleRefColumns)
	}

	for _, legacyColumn := range []string{"runner_type", "model", "policy_ref", "model_preset", "fallback_runner_types"} {
		var count int
		if err := db.Get(&count, "SELECT COUNT(*) FROM pragma_table_info('agent_profiles') WHERE name = ?", legacyColumn); err != nil {
			t.Fatalf("inspect %s: %v", legacyColumn, err)
		}
		if count != 0 {
			t.Fatalf("legacy column %s unexpectedly present", legacyColumn)
		}
	}
}

func TestInitSchemaMigratesExistingInvocationReadModelRunColumnsBeforeValidation(t *testing.T) {
	tmpDir := t.TempDir()
	sqlDB, err := sqlx.Connect("sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)", filepath.Join(tmpDir, "legacy.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	// This is the exact pre-cost-attribution projection shape. The schema
	// validator must not reject it before the additive migration has a chance
	// to add the new columns.
	if _, err := sqlDB.Exec(`CREATE TABLE invocation_read_model_runs (
		run_id TEXT PRIMARY KEY, occurred_at TEXT NOT NULL, created_at TEXT NOT NULL,
		started_at TEXT, ended_at TEXT, duration_ms INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL, profile_id TEXT NOT NULL DEFAULT 'unknown',
		runner_type TEXT NOT NULL DEFAULT 'unknown', model TEXT NOT NULL DEFAULT 'unknown',
		tag TEXT NOT NULL DEFAULT 'unknown', total_cost_usd REAL NOT NULL DEFAULT 0,
		total_tokens INTEGER NOT NULL DEFAULT 0, cost_time_basis TEXT NOT NULL DEFAULT 'terminal_projection',
		projected_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		t.Fatal(err)
	}
	wrapped := NewDB(sqlDB, logrus.New())
	if err := wrapped.InitializeSchema(); err != nil {
		t.Fatalf("InitializeSchema: %v", err)
	}
	columns, err := wrapped.tableColumns(context.Background(), "invocation_read_model_runs")
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"input_cost_usd", "output_cost_usd", "cache_read_cost_usd", "cache_creation_cost_usd", "input_tokens", "output_tokens", "cache_read_tokens", "cache_creation_tokens", "preamble_injected_tokens", "preamble_fixed_tokens", "preamble_token_basis", "unattributed_tokens", "unattributed_reason"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("migration did not add %s; columns=%v", column, columns)
		}
	}
}

func TestInitSchemaMigratesExistingFindingColumnsBeforeValidation(t *testing.T) {
	tmpDir := t.TempDir()
	sqlDB, err := sqlx.Connect("sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)", filepath.Join(tmpDir, "legacy-findings.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec(`CREATE TABLE run_findings (
		id TEXT PRIMARY KEY, run_id TEXT NOT NULL, investigation_run_id TEXT NOT NULL,
		category TEXT NOT NULL, severity TEXT NOT NULL, recommendation_text TEXT NOT NULL,
		evidence TEXT NOT NULL DEFAULT '', target_path TEXT NOT NULL DEFAULT '',
		fingerprint TEXT NOT NULL, operator_decision TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := sqlDB.Exec(`INSERT INTO run_findings
		(id,run_id,investigation_run_id,category,severity,recommendation_text,evidence,target_path,fingerprint,operator_decision,created_at)
		VALUES ('legacy-finding','run-a','investigation-a','tooling','warning','Inspect the command','command outcome owner','api/run.go','legacy-fingerprint','','2026-09-06T12:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	wrapped := NewDB(sqlDB, logrus.New())
	if err := wrapped.InitializeSchema(); err != nil {
		t.Fatalf("InitializeSchema: %v", err)
	}
	columns, err := wrapped.tableColumns(context.Background(), "run_findings")
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"proposed_workaround", "acceptance_impact", "owner_hint", "target_measure", "before_value", "after_value", "effectiveness", "friction_topic"} {
		if _, ok := columns[column]; !ok {
			t.Fatalf("migration did not add %s; columns=%v", column, columns)
		}
	}
	var quality string
	var resolved, outcome, owner bool
	if err := sqlDB.QueryRow(`SELECT quality_signal,cites_resolved_commands,cites_real_outcome,cites_attributed_owner FROM run_findings WHERE id='legacy-finding'`).Scan(&quality, &resolved, &outcome, &owner); err != nil {
		t.Fatal(err)
	}
	if quality != "unavailable" || resolved || outcome || owner {
		t.Fatalf("legacy keyword-shaped finding was upgraded during migration: quality=%q resolved=%v outcome=%v owner=%v", quality, resolved, outcome, owner)
	}
}

func TestInitSchemaMigratesLegacyWatchActionStatusBeforeIndexCreation(t *testing.T) {
	tmpDir := t.TempDir()
	sqlDB, err := sqlx.Connect("sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)", filepath.Join(tmpDir, "legacy-watch-actions.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec(`CREATE TABLE cohort_watch_actions (
		action_id TEXT PRIMARY KEY, watch_id TEXT NOT NULL, decision_id TEXT,
		idempotency_key TEXT NOT NULL UNIQUE, kind INTEGER NOT NULL,
		target_run_id TEXT NOT NULL DEFAULT '', status TEXT NOT NULL,
		action_json TEXT NOT NULL, cooldown_until TEXT, created_at TEXT NOT NULL,
		acknowledged_at TEXT
	); INSERT INTO cohort_watch_actions
		(action_id,watch_id,idempotency_key,kind,status,action_json,created_at)
		VALUES ('action-1','watch-1','key-1',1,'requested','{}','2026-09-04T20:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	wrapper := NewDB(sqlDB, logrus.New())
	if err := wrapper.InitializeSchema(); err != nil {
		t.Fatalf("InitializeSchema: %v", err)
	}
	var state int
	if err := sqlDB.Get(&state, `SELECT state FROM cohort_watch_actions WHERE action_id='action-1'`); err != nil {
		t.Fatal(err)
	}
	if state != 1 {
		t.Fatalf("migrated state=%d, want requested(1)", state)
	}
}

func TestInitSchemaMigratesExistingInvestigationReceiptTimingColumn(t *testing.T) {
	tmpDir := t.TempDir()
	sqlDB, err := sqlx.Connect("sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)", filepath.Join(tmpDir, "legacy-investigation.db")))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec(`CREATE TABLE investigation_cross_scenario_calls (
		run_id TEXT NOT NULL, receipt_event_id TEXT NOT NULL, target_scenario TEXT NOT NULL,
		operation TEXT NOT NULL, outcome TEXT NOT NULL, status_code INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL, verified INTEGER NOT NULL, projection TEXT NOT NULL,
		ledger_availability TEXT NOT NULL, PRIMARY KEY (run_id, receipt_event_id)
	)`); err != nil {
		t.Fatal(err)
	}
	wrapped := NewDB(sqlDB, logrus.New())
	if err := wrapped.InitializeSchema(); err != nil {
		t.Fatalf("initialize legacy investigation schema: %v", err)
	}
	var count int
	if err := sqlDB.Get(&count, "SELECT COUNT(*) FROM pragma_table_info('investigation_cross_scenario_calls') WHERE name = 'occurred_at'"); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("occurred_at column count = %d, want 1", count)
	}
}

func TestDataDirPrefersCanonicalStorageOverLegacyFallbackEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SQLITE_DATABASE_PATH", filepath.Join(home, "legacy-sqlite-root"))
	t.Setenv("VROOLI_DATA", filepath.Join(home, "legacy-vrooli-data"))
	t.Setenv("AM_SQLITE_PATH", "")
	// Exercise the HOME-derived default, not a host-level canonical override
	// inherited by the test process (for example from web-console).
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("VROOLI_DATA_ROOT", "")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_SCENARIO", "")
	t.Setenv("VROOLI_VARIANT", "")

	got := DataDir()
	want := filepath.Join(home, ".vrooli", "data", "vrooli", "agent-manager")
	if got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestSQLiteDSNUsesCanonicalPathWithoutLegacyMigration(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AM_SQLITE_PATH", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SQLITE_DATABASE_PATH", filepath.Join(home, "legacy-sqlite-root"))
	t.Setenv("VROOLI_DATA", filepath.Join(home, "legacy-vrooli-data"))
	t.Setenv("VROOLI_STORAGE_ROOT", "")
	t.Setenv("VROOLI_DATA_ROOT", "")
	t.Setenv("VROOLI_STORAGE_NAMESPACE", "")
	t.Setenv("VROOLI_SCENARIO", "")
	t.Setenv("VROOLI_VARIANT", "")

	dsn, err := SQLiteDSN(nil)
	if err != nil {
		t.Fatalf("SQLiteDSN() error = %v", err)
	}
	wantPath := filepath.Join(home, ".vrooli", "data", "vrooli", "agent-manager", "agent-manager.db")
	if !strings.Contains(dsn, wantPath) {
		t.Fatalf("SQLiteDSN() = %q, want path containing %q", dsn, wantPath)
	}
	if _, err := os.Stat(filepath.Dir(wantPath)); err != nil {
		t.Fatalf("expected canonical sqlite dir at %s: %v", filepath.Dir(wantPath), err)
	}
}

// ============================================================================
// Profile Repository Tests
// ============================================================================
