package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"

	"github.com/gorilla/mux"
)

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

	var resp incidents.ListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Filters.Limit != 50 {
		t.Errorf("Limit = %d, want 50", resp.Filters.Limit)
	}
}

func TestIncidents_WithQueryParams(t *testing.T) {
	store := &mockStore{}
	h := setupTestHandlers(store)

	req := httptest.NewRequest("GET", "/api/v1/incidents?status=open&severity=critical&limit=100", nil)
	w := httptest.NewRecorder()

	h.Incidents(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Incidents() status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp incidents.ListResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Filters.Status != incidents.StatusOpen {
		t.Errorf("Status = %q, want open", resp.Filters.Status)
	}
	if resp.Filters.Severity != incidents.SeverityCritical {
		t.Errorf("Severity = %q, want critical", resp.Filters.Severity)
	}
	if resp.Filters.Limit != 100 {
		t.Errorf("Limit = %d, want 100", resp.Filters.Limit)
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
