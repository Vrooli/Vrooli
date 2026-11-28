package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewHandler_ReturnsNonNil(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	h := NewHandler(db)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
	if h.healthFunc == nil {
		t.Fatal("expected healthFunc to be set")
	}
}

func TestNewHandler_NilDB(t *testing.T) {
	// NewHandler should not panic even with a nil DB.
	h := NewHandler(nil)
	if h == nil {
		t.Fatal("expected non-nil handler even with nil db")
	}
}

func TestHealth_Returns200_WhenDBIsHealthy(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	// The standardized health checker probes database liveness with PingContext.
	mock.ExpectPing()

	h := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

func TestHealth_ContentType_IsJSON(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	mock.ExpectPing()

	h := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

func TestHealth_ResponseBody_ContainsExpectedFields(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	mock.ExpectPing()

	h := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	// Check required top-level fields per the Vrooli health schema.
	requiredFields := []string{"status", "service", "timestamp", "readiness"}
	for _, field := range requiredFields {
		if _, ok := resp[field]; !ok {
			t.Errorf("expected field %q in response", field)
		}
	}

	// Status should be healthy when DB is up.
	if status, _ := resp["status"].(string); status != "healthy" {
		t.Errorf("expected status healthy, got %q", status)
	}

	// Readiness should be true.
	if readiness, _ := resp["readiness"].(bool); !readiness {
		t.Error("expected readiness true")
	}

	// Version should be 1.0.0 (set in NewHandler).
	if version, _ := resp["version"].(string); version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %q", version)
	}
}

func TestHealth_ResponseBody_ContainsDependencies(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	mock.ExpectPing()

	h := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	deps, ok := resp["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatal("expected dependencies map in response")
	}

	dbDep, ok := deps["database"].(map[string]interface{})
	if !ok {
		t.Fatal("expected database dependency in response")
	}

	if connected, _ := dbDep["connected"].(bool); !connected {
		t.Error("expected database connected to be true")
	}
}

func TestHealth_ResponseBody_ContainsMetrics(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	mock.ExpectPing()

	h := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	metrics, ok := resp["metrics"].(map[string]interface{})
	if !ok {
		t.Fatal("expected metrics map in response")
	}

	expectedMetrics := []string{"goroutines", "heap_mb", "uptime_seconds"}
	for _, m := range expectedMetrics {
		if _, ok := metrics[m]; !ok {
			t.Errorf("expected metric %q in response", m)
		}
	}
}

func TestHealth_Returns503_WhenDBPingFails(t *testing.T) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	// Simulate DB ping failure.
	mock.ExpectPing().WillReturnError(errTestDB)

	h := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	// DB check is Critical in NewHandler, so failure should yield 503.
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if status, _ := resp["status"].(string); status != "unhealthy" {
		t.Errorf("expected status unhealthy, got %q", status)
	}

	if readiness, _ := resp["readiness"].(bool); readiness {
		t.Error("expected readiness false when unhealthy")
	}

	// Verify the database dependency reports the error.
	deps, _ := resp["dependencies"].(map[string]interface{})
	dbDep, _ := deps["database"].(map[string]interface{})
	if connected, _ := dbDep["connected"].(bool); connected {
		t.Error("expected database connected to be false")
	}
	if dbDep["error"] == nil {
		t.Error("expected error in database dependency")
	}
}

func TestHealth_NilDB_Returns503(t *testing.T) {
	h := NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	h.Health(w, req)

	// nil db is passed as a Critical check, so it should be unhealthy.
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if status, _ := resp["status"].(string); status != "unhealthy" {
		t.Errorf("expected status unhealthy, got %q", status)
	}
}

// errTestDB is a sentinel error for testing DB failures.
type testDBError struct{}

func (e *testDBError) Error() string { return "database connection failed" }

var errTestDB = &testDBError{}
