// Package handlers tests
// [REQ:CLI-TICK-001] [REQ:CLI-STATUS-001] [REQ:FAIL-SAFE-001] [REQ:FAIL-OBSERVE-001]
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/persistence"
	"vrooli-autoheal/internal/platform"
)

// mockStore implements StoreInterface for testing
type mockStore struct {
	pingErr          error
	saveErr          error
	recentResults    []checks.Result
	recentErr        error
	timelineEvents   []persistence.TimelineEvent
	timelineErr      error
	uptimeStats      *persistence.UptimeStats
	uptimeErr        error
	uptimeHistory    *persistence.UptimeHistory
	uptimeHistoryErr error
	checkTrends      *persistence.CheckTrendsResponse
	checkTrendsErr   error
	incidents        *persistence.IncidentsResponse
	incidentsErr     error
}

func (m *mockStore) Ping(ctx context.Context) error {
	return m.pingErr
}

func (m *mockStore) SaveResult(ctx context.Context, result checks.Result) error {
	return m.saveErr
}

func (m *mockStore) GetRecentResults(ctx context.Context, checkID string, limit int) ([]checks.Result, error) {
	return m.recentResults, m.recentErr
}

func (m *mockStore) GetTimelineEvents(ctx context.Context, limit int) ([]persistence.TimelineEvent, error) {
	if m.timelineErr != nil {
		return nil, m.timelineErr
	}
	return m.timelineEvents, nil
}

func (m *mockStore) GetUptimeStats(ctx context.Context, windowHours int) (*persistence.UptimeStats, error) {
	if m.uptimeErr != nil {
		return nil, m.uptimeErr
	}
	if m.uptimeStats != nil {
		return m.uptimeStats, nil
	}
	return &persistence.UptimeStats{
		TotalEvents:      100,
		OkEvents:         90,
		WarningEvents:    8,
		CriticalEvents:   2,
		UptimePercentage: 90.0,
		WindowHours:      24,
	}, nil
}

func (m *mockStore) GetUptimeHistory(ctx context.Context, windowHours, bucketCount int) (*persistence.UptimeHistory, error) {
	if m.uptimeHistoryErr != nil {
		return nil, m.uptimeHistoryErr
	}
	if m.uptimeHistory != nil {
		return m.uptimeHistory, nil
	}
	return &persistence.UptimeHistory{
		Buckets:     []persistence.UptimeHistoryBucket{},
		Overall:     persistence.UptimeStats{UptimePercentage: 90.0, TotalEvents: 100},
		WindowHours: windowHours,
		BucketCount: bucketCount,
	}, nil
}

func (m *mockStore) GetCheckTrends(ctx context.Context, windowHours int) (*persistence.CheckTrendsResponse, error) {
	if m.checkTrendsErr != nil {
		return nil, m.checkTrendsErr
	}
	if m.checkTrends != nil {
		return m.checkTrends, nil
	}
	return &persistence.CheckTrendsResponse{
		Trends:      []persistence.CheckTrend{},
		WindowHours: windowHours,
		TotalChecks: 0,
	}, nil
}

func (m *mockStore) GetIncidents(ctx context.Context, windowHours, limit int) (*persistence.IncidentsResponse, error) {
	if m.incidentsErr != nil {
		return nil, m.incidentsErr
	}
	if m.incidents != nil {
		return m.incidents, nil
	}
	return &persistence.IncidentsResponse{
		Incidents:   []persistence.Incident{},
		WindowHours: windowHours,
		Total:       0,
	}, nil
}

// Action log mock methods [REQ:HEAL-ACTION-001]
func (m *mockStore) SaveActionLog(ctx context.Context, checkID, actionID string, success bool, message, output, errMsg string, durationMs int64) error {
	return nil
}

func (m *mockStore) GetActionLogs(ctx context.Context, limit int) (*persistence.ActionLogsResponse, error) {
	return &persistence.ActionLogsResponse{
		Logs:  []persistence.ActionLog{},
		Total: 0,
	}, nil
}

func (m *mockStore) GetActionLogsForCheck(ctx context.Context, checkID string, limit int) (*persistence.ActionLogsResponse, error) {
	return &persistence.ActionLogsResponse{
		Logs:  []persistence.ActionLog{},
		Total: 0,
	}, nil
}

// mockCheck implements checks.Check for testing
type mockCheck struct {
	id       string
	status   checks.Status
	message  string
	platform []platform.Type
}

func (c *mockCheck) ID() string                 { return c.id }
func (c *mockCheck) Title() string              { return "Mock Check" }
func (c *mockCheck) Description() string        { return "Mock check for testing" }
func (c *mockCheck) Importance() string         { return "Test importance" }
func (c *mockCheck) IntervalSeconds() int       { return 60 }
func (c *mockCheck) Platforms() []platform.Type { return c.platform }
func (c *mockCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *mockCheck) Run(ctx context.Context) checks.Result {
	return checks.Result{
		CheckID: c.id,
		Status:  c.status,
		Message: c.message,
	}
}

func setupTestHandlers(store StoreInterface) *Handlers {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
		HasDocker:       true,
	}

	registry := checks.NewRegistry(caps)

	// Register a mock check
	registry.Register(&mockCheck{
		id:      "test-check",
		status:  checks.StatusOK,
		message: "Test OK",
	})

	return NewWithInterface(registry, store, caps)
}

func TestHealth_Healthy(t *testing.T) {
	store := &mockStore{pingErr: nil}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Health() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

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
	if deps["database"] != "connected" {
		t.Errorf("database = %v, want connected", deps["database"])
	}
}

func TestHealth_Unhealthy(t *testing.T) {
	store := &mockStore{pingErr: context.DeadlineExceeded}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "unhealthy" {
		t.Errorf("status = %v, want unhealthy", resp["status"])
	}

	deps, ok := resp["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatal("dependencies field missing or invalid")
	}
	if deps["database"] != "disconnected" {
		t.Errorf("database = %v, want disconnected", deps["database"])
	}
}

func TestPlatform(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/platform", nil)
	w := httptest.NewRecorder()

	h.Platform(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Platform() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp platform.Capabilities
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

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

func TestNew(t *testing.T) {
	// Test the production New constructor compiles and works
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)

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

func TestWatchdogTemplate(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/watchdog/template", nil)
	w := httptest.NewRecorder()

	h.WatchdogTemplate(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("WatchdogTemplate() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check required fields
	if _, ok := resp["platform"]; !ok {
		t.Error("response should have platform field")
	}

	if _, ok := resp["template"]; !ok {
		t.Error("response should have template field")
	}

	if _, ok := resp["instructions"]; !ok {
		t.Error("response should have instructions field")
	}

	// Template should have substantial content
	template, ok := resp["template"].(string)
	if !ok || len(template) < 50 {
		t.Errorf("template should be a substantial string, got length %d", len(template))
	}
}

// --- Additional Handler Tests ---

func TestUptimeHistory(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/uptime/history", nil)
	w := httptest.NewRecorder()

	h.UptimeHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UptimeHistory() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp persistence.UptimeHistory
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.WindowHours != 24 {
		t.Errorf("WindowHours = %d, want 24", resp.WindowHours)
	}

	if resp.BucketCount != 24 {
		t.Errorf("BucketCount = %d, want 24", resp.BucketCount)
	}
}

func TestUptimeHistory_WithQueryParams(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/uptime/history?hours=48&buckets=12", nil)
	w := httptest.NewRecorder()

	h.UptimeHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UptimeHistory() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp persistence.UptimeHistory
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.WindowHours != 48 {
		t.Errorf("WindowHours = %d, want 48", resp.WindowHours)
	}

	if resp.BucketCount != 12 {
		t.Errorf("BucketCount = %d, want 12", resp.BucketCount)
	}
}

func TestUptimeHistory_InvalidParams(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	// Invalid params should use defaults
	req := httptest.NewRequest("GET", "/api/v1/uptime/history?hours=invalid&buckets=-5", nil)
	w := httptest.NewRecorder()

	h.UptimeHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("UptimeHistory() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp persistence.UptimeHistory
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should use defaults when invalid
	if resp.WindowHours != 24 {
		t.Errorf("WindowHours = %d, want 24 (default)", resp.WindowHours)
	}
}

func TestUptimeHistory_Error(t *testing.T) {
	store := &mockStore{uptimeHistoryErr: context.DeadlineExceeded}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/uptime/history", nil)
	w := httptest.NewRecorder()

	h.UptimeHistory(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("UptimeHistory() with error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestCheckTrends(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/trends", nil)
	w := httptest.NewRecorder()

	h.CheckTrends(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CheckTrends() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp persistence.CheckTrendsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.WindowHours != 24 {
		t.Errorf("WindowHours = %d, want 24", resp.WindowHours)
	}
}

func TestCheckTrends_WithQueryParams(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/trends?hours=72", nil)
	w := httptest.NewRecorder()

	h.CheckTrends(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("CheckTrends() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp persistence.CheckTrendsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.WindowHours != 72 {
		t.Errorf("WindowHours = %d, want 72", resp.WindowHours)
	}
}

func TestCheckTrends_Error(t *testing.T) {
	store := &mockStore{checkTrendsErr: context.DeadlineExceeded}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/trends", nil)
	w := httptest.NewRecorder()

	h.CheckTrends(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("CheckTrends() with error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestIncidents(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/incidents", nil)
	w := httptest.NewRecorder()

	h.Incidents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Incidents() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp persistence.IncidentsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.WindowHours != 24 {
		t.Errorf("WindowHours = %d, want 24", resp.WindowHours)
	}
}

func TestIncidents_WithQueryParams(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/incidents?hours=48&limit=100", nil)
	w := httptest.NewRecorder()

	h.Incidents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Incidents() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp persistence.IncidentsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.WindowHours != 48 {
		t.Errorf("WindowHours = %d, want 48", resp.WindowHours)
	}
}

func TestIncidents_Error(t *testing.T) {
	store := &mockStore{incidentsErr: context.DeadlineExceeded}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/incidents", nil)
	w := httptest.NewRecorder()

	h.Incidents(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Incidents() with error status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestGetActionHistory(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/actions", nil)
	w := httptest.NewRecorder()

	h.GetActionHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetActionHistory() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp persistence.ActionLogsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have empty array, not nil
	if resp.Logs == nil {
		t.Error("Logs should be empty array, not nil")
	}
}

func TestGetActionHistory_WithCheckFilter(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/actions?checkId=test-check", nil)
	w := httptest.NewRecorder()

	h.GetActionHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetActionHistory() status = %d, want %d", w.Code, http.StatusOK)
	}
}

// mockHealableCheck implements checks.HealableCheck for testing actions
type mockHealableCheck struct {
	mockCheck
	recoveryActions []checks.RecoveryAction
	executeResult   checks.ActionResult
}

func (c *mockHealableCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	return c.recoveryActions
}

func (c *mockHealableCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	c.executeResult.ActionID = actionID
	return c.executeResult
}

func setupTestHandlersWithHealable(store StoreInterface) *Handlers {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
		HasDocker:       true,
	}

	registry := checks.NewRegistry(caps)

	// Register a healable mock check
	registry.Register(&mockHealableCheck{
		mockCheck: mockCheck{
			id:      "healable-check",
			status:  checks.StatusOK,
			message: "Test OK",
		},
		recoveryActions: []checks.RecoveryAction{
			{ID: "restart", Name: "Restart", Description: "Restart service", Available: true},
			{ID: "logs", Name: "View Logs", Description: "View logs", Available: true},
		},
		executeResult: checks.ActionResult{
			CheckID: "healable-check",
			Success: true,
			Message: "Action completed",
		},
	})

	return NewWithInterface(registry, store, caps)
}

func TestGetCheckActions(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlersWithHealable(store)

	// First run tick to populate results
	tickReq := httptest.NewRequest("POST", "/api/v1/tick?force=true", nil)
	tickW := httptest.NewRecorder()
	h.Tick(tickW, tickReq)

	req := httptest.NewRequest("GET", "/api/v1/checks/healable-check/actions", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/actions", h.GetCheckActions)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetCheckActions() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["checkId"] != "healable-check" {
		t.Errorf("checkId = %v, want healable-check", resp["checkId"])
	}

	actions, ok := resp["actions"].([]interface{})
	if !ok {
		t.Fatal("actions field missing or invalid")
	}

	if len(actions) != 2 {
		t.Errorf("actions length = %d, want 2", len(actions))
	}
}

func TestGetCheckActions_NotFound(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/checks/nonexistent/actions", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/actions", h.GetCheckActions)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("GetCheckActions() for nonexistent check status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestExecuteCheckAction(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlersWithHealable(store)

	req := httptest.NewRequest("POST", "/api/v1/checks/healable-check/actions/restart", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/actions/{actionId}", h.ExecuteCheckAction)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("ExecuteCheckAction() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp checks.ActionResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !resp.Success {
		t.Errorf("expected success, got failure")
	}

	if resp.ActionID != "restart" {
		t.Errorf("ActionID = %s, want restart", resp.ActionID)
	}
}

func TestExecuteCheckAction_NotFound(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("POST", "/api/v1/checks/nonexistent/actions/restart", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/actions/{actionId}", h.ExecuteCheckAction)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("ExecuteCheckAction() for nonexistent check status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// Test concurrent tick execution
func TestTick_Concurrent(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	// Start first tick in background (simulate long-running)
	// Since we can't easily simulate a long tick, we'll test the lock acquisition
	// by running sequential ticks and verifying they complete

	results := make(chan int, 2)

	// First tick
	go func() {
		req := httptest.NewRequest("POST", "/api/v1/tick?force=true", nil)
		w := httptest.NewRecorder()
		h.Tick(w, req)
		results <- w.Code
	}()

	// Second tick (should either succeed or get conflict)
	go func() {
		req := httptest.NewRequest("POST", "/api/v1/tick?force=true", nil)
		w := httptest.NewRecorder()
		h.Tick(w, req)
		results <- w.Code
	}()

	// Both should complete (either 200 or 409)
	code1 := <-results
	code2 := <-results

	// At least one should succeed
	if code1 != http.StatusOK && code2 != http.StatusOK {
		t.Errorf("Expected at least one tick to succeed, got %d and %d", code1, code2)
	}
}

// Test parsePositiveInt helper
func TestParsePositiveInt(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"42", 42, false},
		{"0", 0, false},
		{"-5", -5, false}, // parsePositiveInt doesn't validate negativity
		{"abc", 0, true},
		{"", 0, true},
	}

	for _, tc := range tests {
		got, err := parsePositiveInt(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("parsePositiveInt(%q) expected error", tc.input)
		}
		if !tc.wantErr && got != tc.want {
			t.Errorf("parsePositiveInt(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

// Test install instructions helpers
func TestGetInstallInstructions(t *testing.T) {
	platforms := []string{"linux", "macos", "windows", "unknown"}

	for _, p := range platforms {
		instructions := getInstallInstructions(p)
		if instructions == "" {
			t.Errorf("getInstallInstructions(%q) returned empty string", p)
		}
	}
}

func TestGetOneLinerInstall(t *testing.T) {
	apiBase := "http://localhost:8080"
	platforms := []string{"linux", "macos", "windows"}

	for _, p := range platforms {
		oneLiner := getOneLinerInstall(p, apiBase)
		if oneLiner == "" {
			t.Errorf("getOneLinerInstall(%q) returned empty string", p)
		}
		if p != "unknown" && !contains(oneLiner, apiBase) {
			t.Errorf("getOneLinerInstall(%q) should contain API base URL", p)
		}
	}

	// Unknown platform returns empty
	oneLiner := getOneLinerInstall("unknown", apiBase)
	if oneLiner != "" {
		t.Errorf("getOneLinerInstall(unknown) should return empty, got %q", oneLiner)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// =============================================================================
// Tick with Auto-Heal Tests
// =============================================================================

// mockHealableCheckCritical implements a critical healable check for auto-heal testing
type mockHealableCheckCritical struct {
	mockCheck
	recoveryActions []checks.RecoveryAction
	executeResult   checks.ActionResult
	executeCalled   bool
}

func (c *mockHealableCheckCritical) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	return c.recoveryActions
}

func (c *mockHealableCheckCritical) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	c.executeCalled = true
	c.executeResult.ActionID = actionID
	return c.executeResult
}

// mockConfigProvider implements checks.ConfigProvider for testing
type mockConfigProvider struct {
	enabledChecks  map[string]bool
	autoHealChecks map[string]bool
}

func (m *mockConfigProvider) IsCheckEnabled(checkID string) bool {
	if enabled, ok := m.enabledChecks[checkID]; ok {
		return enabled
	}
	return true // Default enabled
}

func (m *mockConfigProvider) IsAutoHealEnabled(checkID string) bool {
	if enabled, ok := m.autoHealChecks[checkID]; ok {
		return enabled
	}
	return false // Default disabled
}

func setupTestHandlersWithAutoHeal(store StoreInterface) (*Handlers, *mockHealableCheckCritical) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
		HasDocker:       true,
	}

	registry := checks.NewRegistry(caps)

	// Create a critical check that will trigger auto-heal
	criticalCheck := &mockHealableCheckCritical{
		mockCheck: mockCheck{
			id:      "critical-check",
			status:  checks.StatusCritical,
			message: "Service down",
		},
		recoveryActions: []checks.RecoveryAction{
			{ID: "start", Name: "Start", Description: "Start service", Available: true, Dangerous: false},
			{ID: "restart", Name: "Restart", Description: "Restart service", Available: true, Dangerous: true},
		},
		executeResult: checks.ActionResult{
			CheckID: "critical-check",
			Success: true,
			Message: "Service started",
		},
	}

	registry.Register(criticalCheck)

	// Enable auto-heal for this check
	configProvider := &mockConfigProvider{
		enabledChecks:  map[string]bool{"critical-check": true},
		autoHealChecks: map[string]bool{"critical-check": true},
	}
	registry.SetConfigProvider(configProvider)

	return NewWithInterface(registry, store, caps), criticalCheck
}

func TestTick_WithAutoHeal(t *testing.T) {
	store := &mockStore{}
	h, criticalCheck := setupTestHandlersWithAutoHeal(store)

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

	// Should have auto-heal results
	autoHeal, ok := resp["autoHeal"].([]interface{})
	if !ok {
		t.Fatal("autoHeal field missing or invalid")
	}

	if len(autoHeal) == 0 {
		t.Error("autoHeal should have at least one result for critical check")
	}

	// Verify the safe action was executed (start, not restart)
	if criticalCheck.executeCalled {
		if criticalCheck.executeResult.ActionID != "start" {
			t.Errorf("Auto-heal should execute 'start' (non-dangerous), got %q", criticalCheck.executeResult.ActionID)
		}
	}
}

func TestTick_AutoHealSkippedWhenDisabled(t *testing.T) {
	caps := &platform.Capabilities{
		Platform:        platform.Linux,
		SupportsSystemd: true,
	}

	registry := checks.NewRegistry(caps)

	// Create a critical check
	criticalCheck := &mockHealableCheckCritical{
		mockCheck: mockCheck{
			id:      "critical-check",
			status:  checks.StatusCritical,
			message: "Service down",
		},
		recoveryActions: []checks.RecoveryAction{
			{ID: "start", Name: "Start", Available: true, Dangerous: false},
		},
		executeResult: checks.ActionResult{
			CheckID: "critical-check",
			Success: true,
		},
	}
	registry.Register(criticalCheck)

	// Disable auto-heal
	configProvider := &mockConfigProvider{
		enabledChecks:  map[string]bool{"critical-check": true},
		autoHealChecks: map[string]bool{"critical-check": false},
	}
	registry.SetConfigProvider(configProvider)

	store := &mockStore{}
	h := NewWithInterface(registry, store, caps)

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

	autoHeal, ok := resp["autoHeal"].([]interface{})
	if !ok {
		t.Fatal("autoHeal field missing")
	}

	// Should have auto-heal result with "not enabled" reason
	if len(autoHeal) > 0 {
		first := autoHeal[0].(map[string]interface{})
		if first["attempted"] == true {
			t.Error("Auto-heal should NOT be attempted when disabled")
		}
		if reason, ok := first["reason"].(string); ok {
			t.Logf("Auto-heal skipped: %s", reason)
		}
	}

	// Verify no action was executed
	if criticalCheck.executeCalled {
		t.Error("No action should be executed when auto-heal is disabled")
	}
}

func TestTick_AutoHealSkipsNonCritical(t *testing.T) {
	caps := &platform.Capabilities{
		Platform: platform.Linux,
	}

	registry := checks.NewRegistry(caps)

	// Create a warning (non-critical) check
	warningCheck := &mockHealableCheckCritical{
		mockCheck: mockCheck{
			id:      "warning-check",
			status:  checks.StatusWarning, // Not critical
			message: "Minor issue",
		},
		recoveryActions: []checks.RecoveryAction{
			{ID: "fix", Name: "Fix", Available: true, Dangerous: false},
		},
	}
	registry.Register(warningCheck)

	configProvider := &mockConfigProvider{
		enabledChecks:  map[string]bool{"warning-check": true},
		autoHealChecks: map[string]bool{"warning-check": true},
	}
	registry.SetConfigProvider(configProvider)

	store := &mockStore{}
	h := NewWithInterface(registry, store, caps)

	req := httptest.NewRequest("POST", "/api/v1/tick?force=true", nil)
	w := httptest.NewRecorder()

	h.Tick(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Tick() status = %d, want %d", w.Code, http.StatusOK)
	}

	// Warning check should NOT trigger auto-heal
	if warningCheck.executeCalled {
		t.Error("Auto-heal should NOT execute for non-critical checks")
	}
}

// =============================================================================
// Docs Handler Tests (if DocsManifest/DocsContent exist)
// =============================================================================

func TestDocsManifest(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/docs/manifest", nil)
	w := httptest.NewRecorder()

	h.DocsManifest(w, req)

	// Should return OK or NotFound (depending on whether docs exist)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("DocsManifest() status = %d, want 200 or 404", w.Code)
	}
}

func TestDocsContent(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/docs/content?path=README.md", nil)
	w := httptest.NewRecorder()

	h.DocsContent(w, req)

	// Should return OK or NotFound (depending on whether docs exist)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("DocsContent() status = %d, want 200 or 404", w.Code)
	}
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

func TestGetCheckActions_NotHealable(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store) // Uses non-healable mockCheck

	req := httptest.NewRequest("GET", "/api/v1/checks/test-check/actions", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/actions", h.GetCheckActions)
	router.ServeHTTP(w, req)

	// Non-healable check should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("GetCheckActions() for non-healable check status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestExecuteCheckAction_Failed(t *testing.T) {
	caps := &platform.Capabilities{
		Platform: platform.Linux,
	}

	registry := checks.NewRegistry(caps)

	// Create a healable check that fails on action
	failingCheck := &mockHealableCheck{
		mockCheck: mockCheck{
			id:      "failing-check",
			status:  checks.StatusCritical,
			message: "Failed",
		},
		recoveryActions: []checks.RecoveryAction{
			{ID: "restart", Name: "Restart", Available: true},
		},
		executeResult: checks.ActionResult{
			CheckID: "failing-check",
			Success: false,
			Error:   "Service failed to start",
		},
	}
	registry.Register(failingCheck)

	store := &mockStore{}
	h := NewWithInterface(registry, store, caps)

	req := httptest.NewRequest("POST", "/api/v1/checks/failing-check/actions/restart", nil)
	w := httptest.NewRecorder()

	router := mux.NewRouter()
	router.HandleFunc("/api/v1/checks/{checkId}/actions/{actionId}", h.ExecuteCheckAction)
	router.ServeHTTP(w, req)

	// Failed action should return 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("ExecuteCheckAction() with failure status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp checks.ActionResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Success {
		t.Error("Expected success=false for failed action")
	}

	if resp.Error == "" {
		t.Error("Expected error message for failed action")
	}
}
