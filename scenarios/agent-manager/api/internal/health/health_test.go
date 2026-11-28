package health_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/fallback"
	"agent-manager/internal/health"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func newTestStore(t *testing.T) *health.Store {
	t.Helper()
	dir := t.TempDir()
	dsn := "file:" + filepath.Join(dir, "test.db") + "?_pragma=foreign_keys(ON)"
	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
CREATE TABLE model_health_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL,
    runner_type TEXT NOT NULL,
    model_id TEXT NOT NULL,
    status TEXT NOT NULL,
    reason TEXT,
    message TEXT,
    triggered_by TEXT NOT NULL
);
CREATE TABLE runner_health_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL,
    runner_type TEXT NOT NULL,
    status TEXT NOT NULL,
    reason TEXT,
    message TEXT,
    triggered_by TEXT NOT NULL
);
`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return health.NewStore(db)
}

func TestStore_RecordAndSnapshot(t *testing.T) {
	store := newTestStore(t)
	store.RegisterRunners([]string{"claude-code", "codex", "opencode"})

	ctx := context.Background()
	if err := store.RecordModel(ctx, "claude-code", "sonnet-4.5", health.StatusOK, "", "", "probe"); err != nil {
		t.Fatalf("RecordModel ok: %v", err)
	}
	if err := store.RecordModel(ctx, "claude-code", "sonnet-4.5", health.StatusFailed, string(fallback.ReasonRateLimit), "rate limit hit", "run-123"); err != nil {
		t.Fatalf("RecordModel failed: %v", err)
	}
	if err := store.RecordRunner(ctx, "codex", health.StatusFailed, string(fallback.ReasonBinaryMissing), "codex binary missing", "probe"); err != nil {
		t.Fatalf("RecordRunner: %v", err)
	}

	snap, err := store.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	entry, ok := snap.Models["claude-code"]["sonnet-4.5"]
	if !ok {
		t.Fatalf("missing model entry; got %+v", snap)
	}
	// Latest observation wins (failed).
	if entry.Status != health.StatusFailed {
		t.Errorf("Status = %q, want failed", entry.Status)
	}
	if entry.Reason != string(fallback.ReasonRateLimit) {
		t.Errorf("Reason = %q", entry.Reason)
	}
	if entry.Message != "rate limit hit" {
		t.Errorf("Message = %q", entry.Message)
	}
	if entry.LastChecked.IsZero() {
		t.Error("LastChecked should be populated")
	}

	if snap.Runners["codex"].Status != health.StatusFailed {
		t.Errorf("codex runner Status = %q", snap.Runners["codex"].Status)
	}
	// Registered but never observed → unknown
	if snap.Runners["claude-code"].Status != health.StatusUnknown {
		t.Errorf("claude-code runner Status = %q, want unknown", snap.Runners["claude-code"].Status)
	}
	// Registered runners with no models still get an empty inner map
	if _, ok := snap.Models["opencode"]; !ok {
		t.Error("opencode should have an empty model map")
	}
}

func TestStore_LatestModelStatus(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.RecordModel(ctx, "claude-code", "haiku", health.StatusOK, "", "", "probe"); err != nil {
		t.Fatalf("record: %v", err)
	}
	got, err := store.LatestModelStatus(ctx, "claude-code", "haiku")
	if err != nil {
		t.Fatalf("LatestModelStatus: %v", err)
	}
	if got.Status != health.StatusOK {
		t.Errorf("Status = %q", got.Status)
	}

	// Missing pair returns Unknown without error.
	missing, err := store.LatestModelStatus(ctx, "codex", "no-such-model")
	if err != nil {
		t.Fatalf("LatestModelStatus missing: %v", err)
	}
	if missing.Status != health.StatusUnknown {
		t.Errorf("missing pair Status = %q, want unknown", missing.Status)
	}
}

func TestStore_QueryAuditFilters(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_ = store.RecordModel(ctx, "claude-code", "sonnet", health.StatusOK, "", "", "probe")
	}
	_ = store.RecordModel(ctx, "claude-code", "sonnet", health.StatusFailed, string(fallback.ReasonAuthFailure), "401", "run-1")
	_ = store.RecordModel(ctx, "codex", "gpt", health.StatusOK, "", "", "probe")

	rows, err := store.QueryModelAudit(ctx, health.AuditQuery{RunnerType: "claude-code"})
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if len(rows) != 6 {
		t.Errorf("filtered claude-code rows = %d, want 6", len(rows))
	}

	failedRows, err := store.QueryModelAudit(ctx, health.AuditQuery{Status: health.StatusFailed})
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if len(failedRows) != 1 || failedRows[0].Reason != string(fallback.ReasonAuthFailure) {
		t.Errorf("failed query = %+v", failedRows)
	}
}

func TestStore_NilSafety(t *testing.T) {
	var store *health.Store
	store.RegisterRunners([]string{"x"})
	if got := store.RegisteredRunners(); got != nil {
		t.Errorf("nil RegisteredRunners = %v", got)
	}
	if err := store.RecordModel(context.Background(), "x", "y", health.StatusOK, "", "", "probe"); err != nil {
		t.Errorf("nil RecordModel = %v", err)
	}
	snap, err := store.Snapshot(context.Background())
	if err != nil {
		t.Errorf("nil Snapshot err = %v", err)
	}
	if snap.Models == nil || snap.Runners == nil {
		t.Error("nil Snapshot should still return initialised maps")
	}
}

func TestEvictBefore(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	_ = store.RecordModel(ctx, "claude-code", "sonnet", health.StatusOK, "", "", "probe")
	_ = store.RecordRunner(ctx, "claude-code", health.StatusOK, "", "", "probe")

	// Cutoff in the future evicts everything.
	n, err := store.EvictBefore(ctx, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("EvictBefore: %v", err)
	}
	if n != 2 {
		t.Errorf("evicted %d, want 2", n)
	}

	// Cutoff in the past evicts nothing.
	_ = store.RecordModel(ctx, "claude-code", "sonnet", health.StatusOK, "", "", "probe")
	n, _ = store.EvictBefore(ctx, time.Now().Add(-time.Hour))
	if n != 0 {
		t.Errorf("past cutoff evicted %d, want 0", n)
	}

	n, err = store.EvictByRetention(ctx, 0)
	if err != nil || n != 0 {
		t.Errorf("EvictByRetention(0) = %d, %v", n, err)
	}
}
