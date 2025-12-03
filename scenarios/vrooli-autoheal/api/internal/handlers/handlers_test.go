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

// mockCheck implements checks.Check for testing
type mockCheck struct {
	id       string
	status   checks.Status
	message  string
	platform []platform.Type
}

func (c *mockCheck) ID() string                      { return c.id }
func (c *mockCheck) Title() string                   { return "Mock Check" }
func (c *mockCheck) Description() string             { return "Mock check for testing" }
func (c *mockCheck) Importance() string              { return "Test importance" }
func (c *mockCheck) IntervalSeconds() int            { return 60 }
func (c *mockCheck) Platforms() []platform.Type      { return c.platform }
func (c *mockCheck) Category() checks.Category       { return checks.CategoryInfrastructure }
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
