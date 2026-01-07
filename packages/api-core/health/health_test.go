package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/scenario"
)

func TestHandler_Minimal(t *testing.T) {
	// Create handler with no checks
	b := New("test-service")
	b.startTime = time.Date(2024, 1, 15, 10, 29, 50, 0, time.UTC)
	b.nowFunc = func() time.Time { return time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC) }

	handler := b.Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Status != StatusHealthy {
		t.Errorf("expected status healthy, got %s", resp.Status)
	}
	if resp.Service != "test-service" {
		t.Errorf("expected service test-service, got %s", resp.Service)
	}
	if !resp.Readiness {
		t.Error("expected readiness true")
	}
	if resp.Timestamp != "2024-01-15T10:30:00Z" {
		t.Errorf("expected timestamp 2024-01-15T10:30:00Z, got %s", resp.Timestamp)
	}
	if resp.UptimeSeconds != 10 {
		t.Errorf("expected uptime_seconds 10, got %v", resp.UptimeSeconds)
	}
	if resp.Metrics == nil {
		t.Error("expected metrics to be present")
	}
	if _, ok := resp.Metrics["goroutines"]; !ok {
		t.Error("expected goroutines metric")
	}
}

func TestHandler_ServiceFromEnv(t *testing.T) {
	// Use scenario package test hooks
	cleanup := scenario.SetTestHooks(
		func() (string, error) { return "/some/path", nil },
		func(key string) string {
			if key == "SCENARIO_NAME" {
				return "my-scenario"
			}
			return ""
		},
	)
	defer cleanup()

	handler := New().Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Service != "my-scenario-api" {
		t.Errorf("expected service my-scenario-api, got %s", resp.Service)
	}
}

func TestHandler_WithVersion(t *testing.T) {
	handler := New("test").Version("1.2.3").Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Version != "1.2.3" {
		t.Errorf("expected version 1.2.3, got %s", resp.Version)
	}
}

// mockChecker is a test checker that returns configurable results.
type mockChecker struct {
	name      string
	connected bool
	latency   time.Duration
	err       error
}

func (c *mockChecker) Check(ctx context.Context) CheckResult {
	return CheckResult{
		Name:      c.name,
		Connected: c.connected,
		Latency:   c.latency,
		Error:     c.err,
	}
}

func TestHandler_WithPassingCheck(t *testing.T) {
	handler := New("test").
		Check(&mockChecker{name: "database", connected: true, latency: 5 * time.Millisecond}, Optional).
		Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != StatusHealthy {
		t.Errorf("expected healthy, got %s", resp.Status)
	}

	db, ok := resp.Dependencies["database"]
	if !ok {
		t.Fatal("expected database dependency")
	}
	if !db.Connected {
		t.Error("expected database connected")
	}
	if db.Latency == nil {
		t.Error("expected latency to be set")
	}
}

func TestHandler_NonCriticalFailure_Degraded(t *testing.T) {
	handler := New("test").
		Check(&mockChecker{name: "cache", connected: false, err: errTest}, Optional).
		Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Non-critical failures still return 200
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != StatusDegraded {
		t.Errorf("expected degraded, got %s", resp.Status)
	}
	if !resp.Readiness {
		t.Error("expected readiness true for degraded")
	}

	cache := resp.Dependencies["cache"]
	if cache.Connected {
		t.Error("expected cache not connected")
	}
	if cache.Error != "test error" {
		t.Errorf("expected error message, got %s", cache.Error)
	}
}

func TestHandler_CriticalFailure_Unhealthy(t *testing.T) {
	handler := New("test").
		Check(&mockChecker{name: "database", connected: false, err: errTest}, Critical).
		Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Critical failures return 503
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != StatusUnhealthy {
		t.Errorf("expected unhealthy, got %s", resp.Status)
	}
	if resp.Readiness {
		t.Error("expected readiness false for unhealthy")
	}
}

func TestHandler_MixedChecks(t *testing.T) {
	handler := New("test").
		Check(&mockChecker{name: "database", connected: true}, Critical).
		Check(&mockChecker{name: "cache", connected: false, err: errTest}, Optional).
		Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Database passes (critical), cache fails (non-critical) = degraded
	if resp.Status != StatusDegraded {
		t.Errorf("expected degraded, got %s", resp.Status)
	}
}

func TestHandlerFunc_Shorthand(t *testing.T) {
	// Test the Handler() shorthand function
	handler := Handler(
		&mockChecker{name: "db", connected: true},
	)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != StatusHealthy {
		t.Errorf("expected healthy, got %s", resp.Status)
	}
	if _, ok := resp.Dependencies["db"]; !ok {
		t.Error("expected db dependency")
	}
}

func TestHTTPChecker(t *testing.T) {
	// Create a test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	checker := HTTP("external-api", ts.URL)
	result := checker.Check(context.Background())

	if !result.Connected {
		t.Error("expected connected")
	}
	if result.Name != "external-api" {
		t.Errorf("expected name external-api, got %s", result.Name)
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %v", result.Error)
	}
}

func TestHTTPChecker_Failure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	checker := HTTP("external-api", ts.URL)
	result := checker.Check(context.Background())

	if result.Connected {
		t.Error("expected not connected")
	}
	if result.Error == nil {
		t.Error("expected error")
	}
}

func TestHTTPChecker_Unreachable(t *testing.T) {
	checker := HTTP("unreachable", "http://localhost:99999")
	result := checker.Check(context.Background())

	if result.Connected {
		t.Error("expected not connected")
	}
	if result.Error == nil {
		t.Error("expected error")
	}
}

func TestHandler_WithStructuredError(t *testing.T) {
	handler := New("test").
		Check(&mockChecker{
			name:      "storage",
			connected: false,
			err: &ErrorDetail{
				Code:      "STORAGE_NOT_INITIALIZED",
				Message:   "storage client not initialized",
				Category:  "internal",
				Retryable: false,
			},
		}, Optional).
		Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	deps, ok := payload["dependencies"].(map[string]any)
	if !ok {
		t.Fatal("expected dependencies in response")
	}
	storage, ok := deps["storage"].(map[string]any)
	if !ok {
		t.Fatal("expected storage dependency in response")
	}
	errObj, ok := storage["error"].(map[string]any)
	if !ok {
		t.Fatal("expected structured error object in response")
	}
	if errObj["code"] != "STORAGE_NOT_INITIALIZED" {
		t.Errorf("expected error code STORAGE_NOT_INITIALIZED, got %v", errObj["code"])
	}
}

var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func TestDB_NilDatabase(t *testing.T) {
	// When db is nil, health.DB should report as not connected
	handler := New("test").
		Check(DB(nil), Critical).
		Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	// Critical check with nil db should result in unhealthy
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != StatusUnhealthy {
		t.Errorf("expected unhealthy, got %s", resp.Status)
	}

	db := resp.Dependencies["database"]
	if db.Connected {
		t.Error("expected database not connected")
	}
	if db.Error != "not configured" {
		t.Errorf("expected 'not configured' error, got %s", db.Error)
	}
}

func TestCheck_NilChecker(t *testing.T) {
	// Passing nil to Check should be safely ignored
	handler := New("test").
		Check(nil, Critical).
		Check(&mockChecker{name: "valid", connected: true}, Optional).
		Handler()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	var resp Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Status != StatusHealthy {
		t.Errorf("expected healthy, got %s", resp.Status)
	}

	// Only the valid check should be present
	if len(resp.Dependencies) != 1 {
		t.Errorf("expected 1 dependency, got %d", len(resp.Dependencies))
	}
}
