package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestTelemetrySummary_Missing(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	req := httptest.NewRequest("GET", "/api/v1/deployment/telemetry/test-scenario/summary", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_name": "test-scenario"})
	w := httptest.NewRecorder()

	env.Server.telemetrySummaryHandler(w, req)
	resp := assertJSONResponse(t, w, http.StatusOK)

	if exists, ok := resp["exists"].(bool); !ok || exists {
		t.Fatalf("expected exists=false, got %#v", resp["exists"])
	}
}

func TestTelemetrySummary_WithData(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	scenario := "test-scenario"
	telemetryDir := filepath.Join(env.TempDir, ".vrooli", "deployment", "telemetry")
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		t.Fatalf("failed to create telemetry dir: %v", err)
	}
	filePath := filepath.Join(telemetryDir, "test-scenario.jsonl")
	content := strings.Join([]string{
		`{"event":"app_start","ingested_at":"2026-01-01T00:00:00Z"}`,
		`{"event":"app_ready","ingested_at":"2026-01-02T00:00:00Z"}`,
		"",
	}, "\n")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write telemetry file: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/deployment/telemetry/test-scenario/summary", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_name": scenario})
	w := httptest.NewRecorder()

	env.Server.telemetrySummaryHandler(w, req)
	resp := assertJSONResponse(t, w, http.StatusOK)

	assertFieldValue(t, resp, "exists", true)
	assertFieldValue(t, resp, "event_count", float64(2))
	assertFieldValue(t, resp, "last_ingested_at", "2026-01-02T00:00:00Z")
}

func TestTelemetryTail_WithLimit(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	telemetryDir := filepath.Join(env.TempDir, ".vrooli", "deployment", "telemetry")
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		t.Fatalf("failed to create telemetry dir: %v", err)
	}
	filePath := filepath.Join(telemetryDir, "tail-scenario.jsonl")
	content := strings.Join([]string{
		`{"event":"first","ingested_at":"2026-01-01T00:00:00Z"}`,
		`{"event":"second","ingested_at":"2026-01-02T00:00:00Z"}`,
		`{"event":"third","ingested_at":"2026-01-03T00:00:00Z"}`,
		"",
	}, "\n")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write telemetry file: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/deployment/telemetry/tail-scenario/tail?limit=2", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_name": "tail-scenario"})
	w := httptest.NewRecorder()

	env.Server.telemetryTailHandler(w, req)
	resp := assertJSONResponse(t, w, http.StatusOK)

	assertFieldValue(t, resp, "exists", true)
	assertFieldValue(t, resp, "total_lines", float64(3))
	entries, ok := resp["entries"].([]interface{})
	if !ok || len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %#v", resp["entries"])
	}
	lastEntry, ok := entries[1].(map[string]interface{})
	if !ok {
		t.Fatalf("expected entry map, got %#v", entries[1])
	}
	raw, _ := lastEntry["raw"].(string)
	if !strings.Contains(raw, `"event":"third"`) {
		t.Fatalf("expected tail to include last event, got %q", raw)
	}
}

func TestTelemetryInsights_WithData(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	scenario := "insights-scenario"
	telemetryDir := filepath.Join(env.TempDir, ".vrooli", "deployment", "telemetry")
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		t.Fatalf("failed to create telemetry dir: %v", err)
	}
	filePath := filepath.Join(telemetryDir, scenario+".jsonl")
	content := strings.Join([]string{
		`{"event":"app_session_failed","timestamp":"2026-01-01T00:00:00Z","session_id":"sess-1","details":{"reason":"boom","started_at":"2026-01-01T00:00:00Z"}}`,
		`{"event":"smoke_test_passed","timestamp":"2026-01-02T00:00:00Z","session_id":"smoke-1"}`,
		`{"event":"startup_error","timestamp":"2026-01-03T00:00:00Z","level":"error","details":{"message":"bad"}}`,
		"",
	}, "\n")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write telemetry file: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/deployment/telemetry/insights-scenario/insights", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_name": scenario})
	w := httptest.NewRecorder()

	env.Server.telemetryInsightsHandler(w, req)
	resp := assertJSONResponse(t, w, http.StatusOK)

	assertFieldValue(t, resp, "exists", true)
	lastSession, ok := resp["last_session"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected last_session, got %#v", resp["last_session"])
	}
	assertFieldValue(t, lastSession, "status", "failed")
	assertFieldValue(t, lastSession, "session_id", "sess-1")

	lastSmoke, ok := resp["last_smoke_test"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected last_smoke_test, got %#v", resp["last_smoke_test"])
	}
	assertFieldValue(t, lastSmoke, "status", "passed")

	lastError, ok := resp["last_error"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected last_error, got %#v", resp["last_error"])
	}
	assertFieldValue(t, lastError, "event", "startup_error")
}

func TestTelemetryDownload(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	telemetryDir := filepath.Join(env.TempDir, ".vrooli", "deployment", "telemetry")
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		t.Fatalf("failed to create telemetry dir: %v", err)
	}
	filePath := filepath.Join(telemetryDir, "download-scenario.jsonl")
	content := `{"event":"first","ingested_at":"2026-01-01T00:00:00Z"}`
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write telemetry file: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/deployment/telemetry/download-scenario/download", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_name": "download-scenario"})
	w := httptest.NewRecorder()

	env.Server.telemetryDownloadHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, `"event":"first"`) {
		t.Fatalf("expected response to contain telemetry, got %q", body)
	}
}

func TestTelemetryDelete(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()
	t.Setenv("VROOLI_ROOT", env.TempDir)

	scenario := "delete-scenario"
	telemetryDir := filepath.Join(env.TempDir, ".vrooli", "deployment", "telemetry")
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		t.Fatalf("failed to create telemetry dir: %v", err)
	}
	filePath := filepath.Join(telemetryDir, scenario+".jsonl")
	if err := os.WriteFile(filePath, []byte(`{"event":"first"}`), 0o644); err != nil {
		t.Fatalf("failed to write telemetry file: %v", err)
	}

	req := httptest.NewRequest("DELETE", "/api/v1/deployment/telemetry/delete-scenario", nil)
	req = mux.SetURLVars(req, map[string]string{"scenario_name": scenario})
	w := httptest.NewRecorder()

	env.Server.telemetryDeleteHandler(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected telemetry file to be deleted, got %v", err)
	}
}
