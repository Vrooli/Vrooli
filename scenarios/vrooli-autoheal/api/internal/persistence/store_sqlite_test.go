package persistence

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"vrooli-autoheal/internal/checks"
)

func TestSQLiteStore_SaveAndReadHealthResults(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)

	result := checks.Result{
		CheckID:   "infra-network",
		Status:    checks.StatusOK,
		Message:   "Network connectivity OK",
		Details:   map[string]interface{}{"latencyMs": 8},
		Duration:  25 * time.Millisecond,
		Timestamp: time.Now().UTC(),
	}

	if err := store.SaveResult(context.Background(), result); err != nil {
		t.Fatalf("SaveResult() error = %v", err)
	}

	latest, err := store.GetLatestResultPerCheck(context.Background())
	if err != nil {
		t.Fatalf("GetLatestResultPerCheck() error = %v", err)
	}
	if len(latest) != 1 {
		t.Fatalf("latest count = %d, want 1", len(latest))
	}
	if latest[0].CheckID != "infra-network" {
		t.Fatalf("check id = %q, want infra-network", latest[0].CheckID)
	}

	stats, err := store.GetUptimeStats(context.Background(), 24)
	if err != nil {
		t.Fatalf("GetUptimeStats() error = %v", err)
	}
	if stats.TotalEvents != 1 {
		t.Fatalf("total events = %d, want 1", stats.TotalEvents)
	}
}

func TestSQLiteStore_ActionLogRoundTrip(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)

	err := store.SaveActionLog(
		context.Background(),
		"infra-network",
		"restart",
		true,
		"Restart executed",
		"ok",
		"",
		42,
	)
	if err != nil {
		t.Fatalf("SaveActionLog() error = %v", err)
	}

	logs, err := store.GetActionLogs(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetActionLogs() error = %v", err)
	}
	if logs.Total != 1 {
		t.Fatalf("logs total = %d, want 1", logs.Total)
	}
	if logs.Logs[0].CheckID != "infra-network" {
		t.Fatalf("check id = %q, want infra-network", logs.Logs[0].CheckID)
	}
}

func openSQLiteTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
	CREATE TABLE health_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		check_id TEXT NOT NULL,
		status TEXT NOT NULL,
		message TEXT NOT NULL,
		details TEXT NOT NULL DEFAULT '{}',
		duration_ms INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL
	);
	CREATE TABLE action_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		check_id TEXT NOT NULL,
		action_id TEXT NOT NULL,
		success INTEGER NOT NULL,
		message TEXT NOT NULL,
		output TEXT,
		error TEXT,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL
	);
	CREATE TABLE heal_trackers (
		check_id TEXT PRIMARY KEY,
		last_attempt TEXT,
		last_success TEXT,
		consecutive_failures INTEGER NOT NULL DEFAULT 0,
		total_attempts INTEGER NOT NULL DEFAULT 0,
		total_successes INTEGER NOT NULL DEFAULT 0,
		cooldown_until TEXT,
		updated_at TEXT NOT NULL
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("create schema error = %v", err)
	}

	return db
}
