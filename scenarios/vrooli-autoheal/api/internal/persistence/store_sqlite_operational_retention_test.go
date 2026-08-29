package persistence

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	apiretention "github.com/vrooli/api-core/retention"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/userconfig"
)

func TestPruneOperationalHistorySQLiteBoundsEachDataClass(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	old := time.Now().Add(-48 * time.Hour).UTC().Format(time.RFC3339Nano)
	fresh := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339Nano)
	for _, query := range []string{
		"INSERT INTO health_results (check_id, status, message, details, duration_ms, created_at) VALUES ('a', 'ok', '', '{}', 0, '" + old + "')",
		"INSERT INTO health_results (check_id, status, message, details, duration_ms, created_at) VALUES ('fresh', 'ok', '', '{}', 0, '" + fresh + "')",
		"INSERT INTO action_logs (check_id, action_id, message, created_at) VALUES ('a', 'restart', '', '" + old + "')",
		"INSERT INTO action_logs (check_id, action_id, message, created_at) VALUES ('fresh', 'restart', '', '" + fresh + "')",
		"INSERT INTO system_events (fingerprint, occurred_at, ingested_at, source, platform, category, severity, title, summary, boot_id, details_json) VALUES ('retention-test', '" + old + "', '" + old + "', 'test', 'linux', 'test', 'info', 'old', '', '', '{}')",
		"INSERT INTO system_events (fingerprint, occurred_at, ingested_at, source, platform, category, severity, title, summary, boot_id, details_json) VALUES ('retention-fresh', '" + fresh + "', '" + fresh + "', 'test', 'linux', 'test', 'info', 'fresh', '', '', '{}')",
	} {
		if _, err := db.Exec(query); err != nil {
			t.Fatalf("seed retention row: %v", err)
		}
	}
	result, err := store.PruneOperationalHistory(context.Background(), time.Now().Add(-24*time.Hour), 1)
	if err != nil {
		t.Fatalf("PruneOperationalHistory: %v", err)
	}
	if result.HealthResults != 1 || result.ActionLogs != 1 || result.SystemEvents != 1 {
		t.Fatalf("retention result = %#v", result)
	}
	for _, table := range []string{"health_results", "action_logs", "system_events"} {
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatalf("count retained %s rows: %v", table, err)
		}
		if count != 1 {
			t.Errorf("retained %s rows = %d, want the one fresh row", table, count)
		}
	}
}

func TestManifestEnforcesNamedOperationalHistoryWindow(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "..", ".vrooli", "service.json")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read retention manifest: %v", err)
	}
	specs, err := apiretention.ParseManifest(manifest)
	if err != nil {
		t.Fatalf("parse retention manifest: %v", err)
	}
	want := time.Duration(userconfig.DefaultHistoryRetentionHours) * time.Hour
	found := map[string]bool{"health_results": false, "action_logs": false}
	for _, spec := range specs {
		if _, required := found[spec.Budget.Name]; !required {
			continue
		}
		found[spec.Budget.Name] = true
		if spec.Budget.MaxAge != want {
			t.Errorf("%s max_age = %s, want named window %s", spec.Budget.Name, spec.Budget.MaxAge, want)
		}
		if !spec.Budget.HasByteBound() {
			t.Errorf("%s has no byte ceiling", spec.Budget.Name)
		}
	}
	for name, ok := range found {
		if !ok {
			t.Errorf("manifest has no enforced %s budget", name)
		}
	}
}

func TestOperationalRetentionStatusReportsDatabaseAndRange(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	now := time.Now().UTC().Truncate(time.Second)
	if err := store.SaveResult(context.Background(), checks.Result{CheckID: "range", Status: checks.StatusOK, Timestamp: now}); err != nil {
		t.Fatalf("save result: %v", err)
	}
	status, err := store.OperationalRetentionStatus(context.Background())
	if err != nil {
		t.Fatalf("OperationalRetentionStatus: %v", err)
	}
	if status.DatabaseBytes <= 0 || status.OldestAt == nil || status.NewestAt == nil {
		t.Fatalf("retention status = %#v", status)
	}
}

func TestSaveActionLogSQLiteBoundsOversizedText(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	if err := store.SaveActionLog(context.Background(), "a", "restart", false, false, "message", strings.Repeat("x", maxActionLogTextBytes*2), "", 0); err != nil {
		t.Fatalf("SaveActionLog: %v", err)
	}
	logs, err := store.GetActionLogs(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetActionLogs: %v", err)
	}
	if len(logs.Logs) != 1 || len(logs.Logs[0].Output) > maxActionLogTextBytes || !strings.Contains(logs.Logs[0].Output, "truncated") {
		t.Fatalf("bounded action logs = %#v", logs)
	}
}
