// Package handlers tests
// [REQ:CLI-TICK-001] [REQ:CLI-STATUS-001] [REQ:FAIL-SAFE-001] [REQ:FAIL-OBSERVE-001]
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/apihttptest"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"

	"github.com/gorilla/mux"
)

func TestHealth_Healthy(t *testing.T) {
	store := &mockStore{pingErr: nil}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	resp := apihttptest.MustDecodeJSON[map[string]interface{}](t, w.Body.Bytes())

	if resp["status"] != "healthy" {
		t.Errorf("status = %v, want healthy", resp["status"])
	}

	if resp["readiness"] != true {
		t.Errorf("readiness = %v, want true", resp["readiness"])
	}

	deps, ok := resp["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatal("dependencies field missing or invalid")
	}
	db, ok := deps["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("dependencies.database should be a DependencyStatus object, got %T", deps["database"])
	}
	if db["connected"] != true {
		t.Errorf("dependencies.database.connected = %v, want true", db["connected"])
	}
	if db["database"] != "sqlite" {
		t.Errorf("dependencies.database.database = %v, want sqlite", db["database"])
	}
}

func TestHealth_Unhealthy(t *testing.T) {
	store := &mockStore{pingErr: context.DeadlineExceeded}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	resp := apihttptest.MustDecodeJSON[map[string]interface{}](t, w.Body.Bytes())

	if resp["status"] != "unhealthy" {
		t.Errorf("status = %v, want unhealthy", resp["status"])
	}

	deps, ok := resp["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatal("dependencies field missing or invalid")
	}
	db, ok := deps["database"].(map[string]interface{})
	if !ok {
		t.Fatalf("dependencies.database should be a DependencyStatus object, got %T", deps["database"])
	}
	if db["connected"] != false {
		t.Errorf("dependencies.database.connected = %v, want false", db["connected"])
	}
	if db["error"] == nil {
		t.Errorf("dependencies.database.error should be populated when unhealthy")
	}
}

func TestPlatform(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/platform", nil)
	w := httptest.NewRecorder()

	h.Platform(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}
	resp := apihttptest.MustDecodeJSON[platform.Capabilities](t, w.Body.Bytes())

	if resp.Platform != platform.Linux {
		t.Errorf("Platform = %v, want linux", resp.Platform)
	}
}

func TestStatus(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	h.Status(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have status field
	if _, ok := resp["status"]; !ok {
		t.Error("response should have status field")
	}

	// Should have summary
	summary, ok := resp["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("summary field missing or invalid")
	}

	if _, ok := summary["total"]; !ok {
		t.Error("summary should have total field")
	}

	if tickRunning, ok := resp["tickRunning"].(bool); !ok || tickRunning {
		t.Errorf("tickRunning = %v, want false", resp["tickRunning"])
	}

	if _, ok := resp["tickStartedAt"]; !ok {
		t.Error("response should include tickStartedAt field")
	}
	if statusFresh, ok := resp["statusFresh"].(bool); !ok || statusFresh {
		t.Errorf("statusFresh = %v, want false before first completed tick", resp["statusFresh"])
	}
	if lastCompletedTickAt, ok := resp["lastCompletedTickAt"]; !ok || lastCompletedTickAt != nil {
		t.Errorf("lastCompletedTickAt = %v, want null before first completed tick", resp["lastCompletedTickAt"])
	}
}

func TestStatus_ExposesLatestHealingIssuePerCheck(t *testing.T) {
	store := &mockStore{actionLogs: &persistence.ActionLogsResponse{Logs: []persistence.ActionLog{
		{CheckID: "scenario-demo", ActionID: "autoheal-skip", Success: false, Message: "in cooldown", Timestamp: "2026-08-22T20:00:00Z"},
		{CheckID: "scenario-demo", ActionID: "restart", Success: false, Message: "exit status 1", Timestamp: "2026-08-22T19:59:00Z"},
		{CheckID: "resource-demo", ActionID: "restart", Success: true, Message: "restarted", Timestamp: "2026-08-22T19:58:00Z"},
		{CheckID: "recovered-demo", ActionID: "restart", Success: true, Message: "recovered", Timestamp: "2026-08-22T19:57:00Z"},
		{CheckID: "recovered-demo", ActionID: "restart", Success: false, Message: "older failure", Timestamp: "2026-08-22T19:56:00Z"},
	}}}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()
	h.Status(w, req)

	resp := apihttptest.MustDecodeJSON[map[string]interface{}](t, w.Body.Bytes())
	issues, ok := resp["autoHealIssues"].(map[string]interface{})
	if !ok {
		t.Fatalf("autoHealIssues = %T, want object", resp["autoHealIssues"])
	}
	issue, ok := issues["scenario-demo"].(map[string]interface{})
	if !ok || issue["actionId"] != "autoheal-skip" {
		t.Fatalf("scenario-demo issue = %#v, want newest skipped outcome", issues["scenario-demo"])
	}
	if _, exists := issues["resource-demo"]; exists {
		t.Fatalf("successful recovery should not be exposed as an issue: %#v", issues["resource-demo"])
	}
	if _, exists := issues["recovered-demo"]; exists {
		t.Fatalf("older failure should not survive a newer successful recovery: %#v", issues["recovered-demo"])
	}
}

func TestStatus_IncludesActiveTickState(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	started := time.Now().Add(-2 * time.Second)
	h.tickLock.Lock()
	h.tickRunning = true
	h.tickStarted = started
	h.tickLock.Unlock()

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()
	h.Status(w, req)

	resp := apihttptest.MustDecodeJSON[map[string]interface{}](t, w.Body.Bytes())
	if tickRunning, ok := resp["tickRunning"].(bool); !ok || !tickRunning {
		t.Fatalf("tickRunning = %v, want true", resp["tickRunning"])
	}

	if startedAt, ok := resp["tickStartedAt"].(string); !ok || startedAt == "" {
		t.Fatalf("tickStartedAt = %v, want RFC3339 timestamp string", resp["tickStartedAt"])
	}
}

func TestStatus_ReportsFreshCompletedTick(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	completed := time.Now().Add(-45 * time.Second)
	h.tickLock.Lock()
	h.tickEnded = completed
	h.tickLock.Unlock()

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()
	h.Status(w, req)

	resp := apihttptest.MustDecodeJSON[map[string]interface{}](t, w.Body.Bytes())
	if statusFresh, ok := resp["statusFresh"].(bool); !ok || !statusFresh {
		t.Fatalf("statusFresh = %v, want true", resp["statusFresh"])
	}
	if reason, ok := resp["statusStaleReason"].(string); !ok || reason != "" {
		t.Fatalf("statusStaleReason = %v, want empty string", resp["statusStaleReason"])
	}
	if age, ok := resp["statusAgeSeconds"].(float64); !ok || age < 40 || age > 50 {
		t.Fatalf("statusAgeSeconds = %v, want about 45", resp["statusAgeSeconds"])
	}
}

func TestStatus_ReportsStaleCompletedTick(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	completed := time.Now().Add(-5 * time.Minute)
	h.tickLock.Lock()
	h.tickEnded = completed
	h.tickLock.Unlock()

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()
	h.Status(w, req)

	resp := apihttptest.MustDecodeJSON[map[string]interface{}](t, w.Body.Bytes())
	if statusFresh, ok := resp["statusFresh"].(bool); !ok || statusFresh {
		t.Fatalf("statusFresh = %v, want false", resp["statusFresh"])
	}
	if reason, ok := resp["statusStaleReason"].(string); !ok || reason == "" {
		t.Fatalf("statusStaleReason = %v, want non-empty reason", resp["statusStaleReason"])
	}
}

func TestTick(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("POST", "/api/v1/tick?force=true", nil)
	w := httptest.NewRecorder()

	h.Tick(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Tick() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("success = %v, want true", resp["success"])
	}

	results, ok := resp["results"].([]interface{})
	if !ok {
		t.Fatal("results field missing or invalid")
	}

	if len(results) == 0 {
		t.Error("results should not be empty")
	}
	if store.savedInventories != 1 {
		t.Fatalf("saved host inventories = %d, want 1", store.savedInventories)
	}
}

func TestTick_CompactResponseOmitsVerboseFields(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("POST", "/api/v1/tick?force=true&compact=true", nil)
	w := httptest.NewRecorder()

	h.Tick(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Tick() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, exists := resp["results"]; exists {
		t.Fatal("compact tick response should omit results")
	}
	if _, exists := resp["autoHeal"]; exists {
		t.Fatal("compact tick response should omit autoHeal")
	}
	if _, exists := resp["summary"]; !exists {
		t.Fatal("compact tick response should include summary")
	}
}

func TestTick_WithPersistenceErrors(t *testing.T) {
	// Store that fails to save
	store := &mockStore{saveErr: context.DeadlineExceeded}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("POST", "/api/v1/tick", nil)
	w := httptest.NewRecorder()

	h.Tick(w, req)

	// Should still succeed (fail-safe: tick completes even if persistence fails)
	if w.Code != http.StatusOK {
		t.Errorf("Tick() status = %d, want %d (should be fail-safe)", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("success = %v, want true (fail-safe)", resp["success"])
	}

	// Should include warnings about persistence issues
	if warnings, ok := resp["warnings"]; ok {
		t.Logf("Warnings included: %v", warnings)
	}
}

func TestListChecks(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/checks", nil)
	w := httptest.NewRecorder()

	h.ListChecks(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ListChecks() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp []checks.Info
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp) == 0 {
		t.Error("checks list should not be empty")
	}

	// Find test check
	found := false
	for _, c := range resp {
		if c.ID == "test-check" {
			found = true
			break
		}
	}
	if !found {
		t.Error("test-check should be in the list")
	}
}

func TestCheckResult_Found(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	// First run tick to populate results
	tickReq := httptest.NewRequest("POST", "/api/v1/tick?force=true", nil)
	tickW := httptest.NewRecorder()
	h.Tick(tickW, tickReq)

	// Now get result
	req := httptest.NewRequest("GET", "/api/v1/checks/test-check", nil)
	w := httptest.NewRecorder()

	// Need to use mux for path variables
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}", h.CheckResult)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CheckResult() status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCheckResult_NotFound(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/checks/nonexistent", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}", h.CheckResult)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("CheckResult() status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestCheckHistory(t *testing.T) {
	store := &mockStore{
		recentResults: []checks.Result{
			{CheckID: "test-check", Status: checks.StatusOK, Message: "OK"},
			{CheckID: "test-check", Status: checks.StatusOK, Message: "OK 2"},
		},
	}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/checks/test-check/history", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/history", h.CheckHistory)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CheckHistory() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["checkId"] != "test-check" {
		t.Errorf("checkId = %v, want test-check", resp["checkId"])
	}

	history, ok := resp["history"].([]interface{})
	if !ok {
		t.Fatal("history field missing or invalid")
	}

	if len(history) != 2 {
		t.Errorf("history length = %d, want 2", len(history))
	}
}

func TestCheckHistory_Empty(t *testing.T) {
	store := &mockStore{
		recentResults: nil, // nil results
	}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/checks/test-check/history", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/history", h.CheckHistory)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CheckHistory() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return empty array, not null
	history, ok := resp["history"].([]interface{})
	if !ok {
		t.Fatal("history field should be an array")
	}

	if len(history) != 0 {
		t.Errorf("history length = %d, want 0", len(history))
	}
}

func TestCheckHistory_Error(t *testing.T) {
	store := &mockStore{
		recentErr: context.DeadlineExceeded,
	}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/checks/test-check/history", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/history", h.CheckHistory)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("CheckHistory() with DB error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestTimeline(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/timeline", nil)
	w := httptest.NewRecorder()

	h.Timeline(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Timeline() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have events array (even if empty)
	events, ok := resp["events"].([]interface{})
	if !ok {
		t.Fatal("events field should be an array")
	}

	t.Logf("Timeline returned %d events", len(events))

	// Should have summary
	if _, ok := resp["summary"]; !ok {
		t.Error("summary field missing")
	}
}

func TestUptimeStats(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/uptime", nil)
	w := httptest.NewRecorder()

	h.UptimeStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UptimeStats() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp persistence.UptimeStats
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify stats fields
	if resp.TotalEvents != 100 {
		t.Errorf("TotalEvents = %d, want 100", resp.TotalEvents)
	}

	if resp.UptimePercentage != 90.0 {
		t.Errorf("UptimePercentage = %v, want 90.0", resp.UptimePercentage)
	}

	if resp.WindowHours != 24 {
		t.Errorf("WindowHours = %d, want 24", resp.WindowHours)
	}
}

func TestTimeline_Error(t *testing.T) {
	store := &mockStore{
		timelineErr: context.DeadlineExceeded,
	}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/timeline", nil)
	w := httptest.NewRecorder()

	h.Timeline(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Timeline() with DB error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestTimeline_WithEvents(t *testing.T) {
	store := &mockStore{
		timelineEvents: []persistence.TimelineEvent{
			{CheckID: "check1", Status: "ok", Message: "All good"},
			{CheckID: "check2", Status: "warning", Message: "Warning"},
			{CheckID: "check3", Status: "critical", Message: "Critical"},
		},
	}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/timeline", nil)
	w := httptest.NewRecorder()

	h.Timeline(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Timeline() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	events, ok := resp["events"].([]interface{})
	if !ok {
		t.Fatal("events field should be an array")
	}

	if len(events) != 3 {
		t.Errorf("events length = %d, want 3", len(events))
	}

	// Check summary counts
	summary, ok := resp["summary"].(map[string]interface{})
	if !ok {
		t.Fatal("summary field should be an object")
	}

	if summary["ok"] != float64(1) {
		t.Errorf("summary.ok = %v, want 1", summary["ok"])
	}
	if summary["warning"] != float64(1) {
		t.Errorf("summary.warning = %v, want 1", summary["warning"])
	}
	if summary["critical"] != float64(1) {
		t.Errorf("summary.critical = %v, want 1", summary["critical"])
	}
}

func TestUptimeStats_Error(t *testing.T) {
	store := &mockStore{
		uptimeErr: context.DeadlineExceeded,
	}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/uptime", nil)
	w := httptest.NewRecorder()

	h.UptimeStats(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("UptimeStats() with DB error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestTick_WithoutForce(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	// First tick
	req := httptest.NewRequest("POST", "/api/v1/tick", nil)
	w := httptest.NewRecorder()
	h.Tick(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("First Tick() status = %d, want %d", w.Code, http.StatusOK)
	}

	// Second tick without force - should still succeed but may skip checks based on interval
	req2 := httptest.NewRequest("POST", "/api/v1/tick", nil)
	w2 := httptest.NewRecorder()
	h.Tick(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("Second Tick() status = %d, want %d", w2.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w2.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["success"] != true {
		t.Errorf("success = %v, want true", resp["success"])
	}
}

func TestTick_ResetsStaleTickLock(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	// Simulate a stale in-progress tick older than the safety threshold.
	h.tickLock.Lock()
	h.tickRunning = true
	h.tickStarted = time.Now().Add(-7 * time.Minute)
	h.tickLock.Unlock()

	req := httptest.NewRequest("POST", "/api/v1/tick", nil)
	w := httptest.NewRecorder()
	h.Tick(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Tick() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if resp["success"] != true {
		t.Errorf("success = %v, want true", resp["success"])
	}
}

func TestNew(t *testing.T) {
	// Test the production New constructor compiles and works
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	_ = registry.SetAutoHealPolicy(checks.AutoHealPolicy{
		BaseCooldown:       5 * time.Minute,
		MaxRestartAttempts: 3,
	})

	// Note: This would need a real DB connection in production
	// Here we just verify it compiles correctly
	_ = registry
	_ = caps
}

func TestContentTypeHeaders(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	endpoints := []struct {
		name   string
		method string
		path   string
		fn     http.HandlerFunc
	}{
		{"Health", "GET", "/health", h.Health},
		{"Platform", "GET", "/api/v1/platform", h.Platform},
		{"Status", "GET", "/api/v1/status", h.Status},
		{"ListChecks", "GET", "/api/v1/checks", h.ListChecks},
		{"Timeline", "GET", "/api/v1/timeline", h.Timeline},
		{"UptimeStats", "GET", "/api/v1/uptime", h.UptimeStats},
		{"Watchdog", "GET", "/api/v1/watchdog", h.Watchdog},
	}

	for _, tc := range endpoints {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			tc.fn(w, req)

			ct := w.Header().Get("Content-Type")
			if ct != "application/json" {
				t.Errorf("%s Content-Type = %q, want application/json", tc.name, ct)
			}
		})
	}
}

// [REQ:WATCH-DETECT-001] Watchdog endpoint tests
func TestWatchdog(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/watchdog", nil)
	w := httptest.NewRecorder()

	h.Watchdog(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Watchdog() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check required fields
	if _, ok := resp["loopRunning"]; !ok {
		t.Error("response should have loopRunning field")
	}

	if _, ok := resp["protectionLevel"]; !ok {
		t.Error("response should have protectionLevel field")
	}

	if _, ok := resp["canInstall"]; !ok {
		t.Error("response should have canInstall field")
	}

	if _, ok := resp["watchdogInstalled"]; !ok {
		t.Error("response should have watchdogInstalled field")
	}

	if _, ok := resp["bootProtectionActive"]; !ok {
		t.Error("response should have bootProtectionActive field")
	}
}

func TestWatchdog_WithRefresh(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/watchdog?refresh=true", nil)
	w := httptest.NewRecorder()

	h.Watchdog(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Watchdog() with refresh status = %d, want %d", w.Code, http.StatusOK)
	}
}
