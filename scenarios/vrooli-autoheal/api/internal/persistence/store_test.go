// Package persistence provides tests for the database store
// [REQ:PERSIST-STORE-001] [REQ:PERSIST-QUERY-001] [REQ:PERSIST-QUERY-002]
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"

	"vrooli-autoheal/internal/checks"
)

// TestNewStore verifies store creation
func TestNewStore(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	store := NewStore(db)
	if store == nil {
		t.Error("NewStore should return a non-nil store")
	}
	if store.db != db {
		t.Error("Store should hold reference to provided db")
	}
}

// TestStorePing verifies database connectivity check
func TestStorePing(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectPing()

	store := NewStore(db)
	ctx := context.Background()

	if err := store.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreSaveResult verifies health check result persistence
// [REQ:PERSIST-STORE-001]
func TestStoreSaveResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	result := checks.Result{
		CheckID:   "infra-network",
		Status:    checks.StatusOK,
		Message:   "Network connectivity OK",
		Details:   map[string]interface{}{"responseTimeMs": 42},
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
	}

	mock.ExpectExec("INSERT INTO health_results").
		WithArgs(
			result.CheckID,
			result.Status,
			result.Message,
			sqlmock.AnyArg(), // details JSON
			result.Duration.Milliseconds(),
			result.Timestamp,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	store := NewStore(db)
	ctx := context.Background()

	if err := store.SaveResult(ctx, result); err != nil {
		t.Errorf("SaveResult failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreSaveResultNilDetails verifies handling of nil details
func TestStoreSaveResultNilDetails(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	result := checks.Result{
		CheckID:   "infra-network",
		Status:    checks.StatusOK,
		Message:   "Network connectivity OK",
		Details:   nil, // nil details
		Timestamp: time.Now(),
		Duration:  100 * time.Millisecond,
	}

	mock.ExpectExec("INSERT INTO health_results").
		WithArgs(
			result.CheckID,
			result.Status,
			result.Message,
			sqlmock.AnyArg(), // should be "{}" for nil details
			result.Duration.Milliseconds(),
			result.Timestamp,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	store := NewStore(db)
	ctx := context.Background()

	if err := store.SaveResult(ctx, result); err != nil {
		t.Errorf("SaveResult with nil details failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetLatestResultPerCheck verifies retrieval of latest results
// [REQ:PERSIST-QUERY-001]
func TestStoreGetLatestResultPerCheck(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	now := time.Now()
	detailsJSON, _ := json.Marshal(map[string]interface{}{"key": "value"})

	rows := sqlmock.NewRows([]string{
		"check_id", "status", "message", "details", "duration_ms", "created_at",
	}).
		AddRow("infra-network", "ok", "Network OK", detailsJSON, 50, now).
		AddRow("infra-dns", "warning", "DNS slow", detailsJSON, 150, now)

	mock.ExpectQuery("SELECT DISTINCT ON").WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	results, err := store.GetLatestResultPerCheck(ctx)
	if err != nil {
		t.Errorf("GetLatestResultPerCheck failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Verify first result
	if results[0].CheckID != "infra-network" {
		t.Errorf("First result CheckID = %q, want infra-network", results[0].CheckID)
	}
	if results[0].Status != checks.StatusOK {
		t.Errorf("First result Status = %v, want OK", results[0].Status)
	}

	// Verify second result
	if results[1].CheckID != "infra-dns" {
		t.Errorf("Second result CheckID = %q, want infra-dns", results[1].CheckID)
	}
	if results[1].Status != checks.StatusWarning {
		t.Errorf("Second result Status = %v, want Warning", results[1].Status)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetLatestResultPerCheckEmpty verifies empty result handling
func TestStoreGetLatestResultPerCheckEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"check_id", "status", "message", "details", "duration_ms", "created_at",
	})
	// No rows added

	mock.ExpectQuery("SELECT DISTINCT ON").WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	results, err := store.GetLatestResultPerCheck(ctx)
	if err != nil {
		t.Errorf("GetLatestResultPerCheck failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for empty table, got %d", len(results))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetRecentResults verifies recent results retrieval
// [REQ:PERSIST-QUERY-002]
func TestStoreGetRecentResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	now := time.Now()
	detailsJSON, _ := json.Marshal(map[string]interface{}{})

	rows := sqlmock.NewRows([]string{
		"check_id", "status", "message", "details", "duration_ms", "created_at",
	}).
		AddRow("infra-network", "ok", "Network OK", detailsJSON, 50, now).
		AddRow("infra-network", "critical", "Network down", detailsJSON, 0, now.Add(-time.Hour))

	mock.ExpectQuery("SELECT check_id, status, message, details, duration_ms, created_at").
		WithArgs("infra-network", 10).
		WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	results, err := store.GetRecentResults(ctx, "infra-network", 10)
	if err != nil {
		t.Errorf("GetRecentResults failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreCleanupOldResults verifies old result cleanup
func TestStoreCleanupOldResults(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("DELETE FROM health_results").
		WithArgs(24).                               // 24 hours retention
		WillReturnResult(sqlmock.NewResult(0, 100)) // 100 rows deleted

	store := NewStore(db)
	ctx := context.Background()

	deleted, err := store.CleanupOldResults(ctx, 24)
	if err != nil {
		t.Errorf("CleanupOldResults failed: %v", err)
	}

	if deleted != 100 {
		t.Errorf("Expected 100 deleted rows, got %d", deleted)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreClose verifies connection closing
func TestStoreClose(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}

	mock.ExpectClose()

	store := NewStore(db)

	if err := store.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetTimelineEvents verifies timeline event retrieval
// [REQ:UI-EVENTS-001]
func TestStoreGetTimelineEvents(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	now := time.Now()
	detailsJSON, _ := json.Marshal(map[string]interface{}{"foo": "bar"})

	rows := sqlmock.NewRows([]string{
		"check_id", "status", "message", "details", "created_at",
	}).
		AddRow("infra-network", "ok", "Network OK", detailsJSON, now).
		AddRow("infra-dns", "critical", "DNS failed", detailsJSON, now.Add(-time.Minute))

	mock.ExpectQuery("SELECT check_id, status, message, details, created_at").
		WithArgs(50).
		WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	events, err := store.GetTimelineEvents(ctx, 50)
	if err != nil {
		t.Errorf("GetTimelineEvents failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	if events[0].CheckID != "infra-network" {
		t.Errorf("First event CheckID = %q, want infra-network", events[0].CheckID)
	}
	if events[0].Details["foo"] != "bar" {
		t.Error("Details not properly unmarshaled")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetUptimeStats verifies uptime statistics calculation
// [REQ:PERSIST-HISTORY-001]
func TestStoreGetUptimeStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"total", "ok_count", "warning_count", "critical_count",
	}).AddRow(100, 90, 7, 3)

	mock.ExpectQuery("SELECT").
		WithArgs(24). // 24 hour window
		WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	stats, err := store.GetUptimeStats(ctx, 24)
	if err != nil {
		t.Errorf("GetUptimeStats failed: %v", err)
	}

	if stats.TotalEvents != 100 {
		t.Errorf("TotalEvents = %d, want 100", stats.TotalEvents)
	}
	if stats.OkEvents != 90 {
		t.Errorf("OkEvents = %d, want 90", stats.OkEvents)
	}
	if stats.WarningEvents != 7 {
		t.Errorf("WarningEvents = %d, want 7", stats.WarningEvents)
	}
	if stats.CriticalEvents != 3 {
		t.Errorf("CriticalEvents = %d, want 3", stats.CriticalEvents)
	}
	if stats.WindowHours != 24 {
		t.Errorf("WindowHours = %d, want 24", stats.WindowHours)
	}
	// 90/100 = 90%
	if stats.UptimePercentage != 90.0 {
		t.Errorf("UptimePercentage = %f, want 90.0", stats.UptimePercentage)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetUptimeStatsEmpty verifies empty stats handling
func TestStoreGetUptimeStatsEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"total", "ok_count", "warning_count", "critical_count",
	}).AddRow(0, 0, 0, 0)

	mock.ExpectQuery("SELECT").
		WithArgs(24).
		WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	stats, err := store.GetUptimeStats(ctx, 24)
	if err != nil {
		t.Errorf("GetUptimeStats failed: %v", err)
	}

	// No data should default to 100% uptime
	if stats.UptimePercentage != 100.0 {
		t.Errorf("UptimePercentage = %f, want 100.0 for empty stats", stats.UptimePercentage)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreSaveActionLog verifies action log persistence
// [REQ:HEAL-ACTION-001]
func TestStoreSaveActionLog(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectExec("INSERT INTO action_logs").
		WithArgs(
			"infra-docker",                  // checkID
			"restart",                       // actionID
			true,                            // success
			"Docker restarted successfully", // message
			"Docker version 24.0.7",         // output
			"",                              // error
			int64(5000),                     // durationMs
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	store := NewStore(db)
	ctx := context.Background()

	err = store.SaveActionLog(ctx, "infra-docker", "restart", true, "Docker restarted successfully", "Docker version 24.0.7", "", 5000)
	if err != nil {
		t.Errorf("SaveActionLog failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetActionLogs verifies action log retrieval
// [REQ:HEAL-ACTION-001]
func TestStoreGetActionLogs(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "check_id", "action_id", "success", "message", "output", "error", "duration_ms", "created_at",
	}).
		AddRow(1, "infra-docker", "restart", true, "Docker restarted", "output", "", int64(5000), now).
		AddRow(2, "infra-dns", "flush-cache", true, "Cache flushed", "", "", int64(100), now.Add(-time.Hour))

	mock.ExpectQuery("SELECT id, check_id, action_id").
		WithArgs(50).
		WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	logs, err := store.GetActionLogs(ctx, 50)
	if err != nil {
		t.Errorf("GetActionLogs failed: %v", err)
	}

	if logs.Total != 2 {
		t.Errorf("Total = %d, want 2", logs.Total)
	}
	if len(logs.Logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs.Logs))
	}

	// Verify first log
	if logs.Logs[0].CheckID != "infra-docker" {
		t.Errorf("First log CheckID = %q, want infra-docker", logs.Logs[0].CheckID)
	}
	if logs.Logs[0].ActionID != "restart" {
		t.Errorf("First log ActionID = %q, want restart", logs.Logs[0].ActionID)
	}
	if !logs.Logs[0].Success {
		t.Error("First log should be successful")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetActionLogsForCheck verifies per-check action log retrieval
// [REQ:HEAL-ACTION-001]
func TestStoreGetActionLogsForCheck(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"id", "check_id", "action_id", "success", "message", "output", "error", "duration_ms", "created_at",
	}).
		AddRow(1, "infra-docker", "restart", true, "Restarted", "", "", int64(5000), now).
		AddRow(2, "infra-docker", "prune", true, "Pruned", "1GB freed", "", int64(3000), now.Add(-time.Minute))

	mock.ExpectQuery("SELECT id, check_id, action_id").
		WithArgs("infra-docker", 20).
		WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	logs, err := store.GetActionLogsForCheck(ctx, "infra-docker", 20)
	if err != nil {
		t.Errorf("GetActionLogsForCheck failed: %v", err)
	}

	if logs.Total != 2 {
		t.Errorf("Total = %d, want 2", logs.Total)
	}

	// Both should be for infra-docker
	for _, log := range logs.Logs {
		if log.CheckID != "infra-docker" {
			t.Errorf("Log CheckID = %q, want infra-docker", log.CheckID)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetCheckTrends verifies per-check trend data retrieval
// [REQ:PERSIST-HISTORY-001]
func TestStoreGetCheckTrends(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"check_id", "total", "ok_count", "warning_count", "critical_count",
		"last_checked", "current_status", "recent_statuses",
	}).
		AddRow("infra-network", 100, 98, 1, 1, now, sql.NullString{String: "ok", Valid: true}, pq.Array([]string{"ok", "ok", "warning"})).
		AddRow("infra-dns", 100, 90, 5, 5, now, sql.NullString{String: "ok", Valid: true}, pq.Array([]string{"ok", "critical", "ok"}))

	mock.ExpectQuery("WITH check_stats AS").
		WithArgs(24).
		WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	trends, err := store.GetCheckTrends(ctx, 24)
	if err != nil {
		t.Errorf("GetCheckTrends failed: %v", err)
	}

	if trends.TotalChecks != 2 {
		t.Errorf("TotalChecks = %d, want 2", trends.TotalChecks)
	}
	if trends.WindowHours != 24 {
		t.Errorf("WindowHours = %d, want 24", trends.WindowHours)
	}

	// Verify first trend
	if len(trends.Trends) < 1 {
		t.Fatal("Expected at least 1 trend")
	}
	trend := trends.Trends[0]
	if trend.CheckID != "infra-network" {
		t.Errorf("First trend CheckID = %q, want infra-network", trend.CheckID)
	}
	if trend.Total != 100 {
		t.Errorf("Total = %d, want 100", trend.Total)
	}
	if trend.UptimePercent != 98.0 {
		t.Errorf("UptimePercent = %f, want 98.0", trend.UptimePercent)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetIncidents verifies status transition detection
// [REQ:PERSIST-HISTORY-001]
func TestStoreGetIncidents(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	now := time.Now()

	rows := sqlmock.NewRows([]string{
		"check_id", "created_at", "prev_status", "status", "message",
	}).
		AddRow("infra-network", now, "ok", "critical", "Network down").
		AddRow("infra-network", now.Add(5*time.Minute), "critical", "ok", "Network restored")

	mock.ExpectQuery("WITH ordered_results AS").
		WithArgs(24, 50).
		WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	incidents, err := store.GetIncidents(ctx, 24, 50)
	if err != nil {
		t.Errorf("GetIncidents failed: %v", err)
	}

	if incidents.Total != 2 {
		t.Errorf("Total = %d, want 2", incidents.Total)
	}
	if incidents.WindowHours != 24 {
		t.Errorf("WindowHours = %d, want 24", incidents.WindowHours)
	}

	// Verify incidents
	if len(incidents.Incidents) < 2 {
		t.Fatal("Expected 2 incidents")
	}

	// First incident: ok -> critical
	if incidents.Incidents[0].FromStatus != "ok" {
		t.Errorf("First incident FromStatus = %q, want ok", incidents.Incidents[0].FromStatus)
	}
	if incidents.Incidents[0].ToStatus != "critical" {
		t.Errorf("First incident ToStatus = %q, want critical", incidents.Incidents[0].ToStatus)
	}

	// Second incident: critical -> ok
	if incidents.Incidents[1].FromStatus != "critical" {
		t.Errorf("Second incident FromStatus = %q, want critical", incidents.Incidents[1].FromStatus)
	}
	if incidents.Incidents[1].ToStatus != "ok" {
		t.Errorf("Second incident ToStatus = %q, want ok", incidents.Incidents[1].ToStatus)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreGetIncidentsDefaultLimits verifies default limit handling
func TestStoreGetIncidentsDefaultLimits(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	rows := sqlmock.NewRows([]string{
		"check_id", "created_at", "prev_status", "status", "message",
	})

	// Should use default 24 hours and 50 limit
	mock.ExpectQuery("WITH ordered_results AS").
		WithArgs(24, 50).
		WillReturnRows(rows)

	store := NewStore(db)
	ctx := context.Background()

	// Pass invalid values to trigger defaults
	_, err = store.GetIncidents(ctx, 0, 0)
	if err != nil {
		t.Errorf("GetIncidents failed: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestStoreQueryError verifies error handling
func TestStoreQueryError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT DISTINCT ON").
		WillReturnError(sql.ErrConnDone)

	store := NewStore(db)
	ctx := context.Background()

	_, err = store.GetLatestResultPerCheck(ctx)
	if err == nil {
		t.Error("Expected error when query fails")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled expectations: %v", err)
	}
}

// TestTimelineEventStruct verifies TimelineEvent JSON serialization
func TestTimelineEventStruct(t *testing.T) {
	event := TimelineEvent{
		CheckID:   "infra-network",
		Status:    "ok",
		Message:   "Network OK",
		Details:   map[string]interface{}{"key": "value"},
		Timestamp: "2024-01-01T12:00:00Z",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Errorf("Failed to marshal TimelineEvent: %v", err)
	}

	var decoded TimelineEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("Failed to unmarshal TimelineEvent: %v", err)
	}

	if decoded.CheckID != event.CheckID {
		t.Errorf("CheckID mismatch: %q vs %q", decoded.CheckID, event.CheckID)
	}
}

// TestActionLogStruct verifies ActionLog JSON serialization
func TestActionLogStruct(t *testing.T) {
	log := ActionLog{
		ID:         1,
		CheckID:    "infra-docker",
		ActionID:   "restart",
		Success:    true,
		Message:    "Docker restarted",
		Output:     "Docker version 24.0.7",
		Error:      "",
		DurationMs: 5000,
		Timestamp:  "2024-01-01T12:00:00Z",
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Errorf("Failed to marshal ActionLog: %v", err)
	}

	var decoded ActionLog
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Errorf("Failed to unmarshal ActionLog: %v", err)
	}

	if decoded.ActionID != log.ActionID {
		t.Errorf("ActionID mismatch: %q vs %q", decoded.ActionID, log.ActionID)
	}
}
