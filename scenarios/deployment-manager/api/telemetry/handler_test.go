package telemetry

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"deployment-manager/shared"
)

func withTelemetryDir(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("DEPLOYMENT_MANAGER_TELEMETRY_DIR", dir)
	return func() {
		os.Unsetenv("DEPLOYMENT_MANAGER_TELEMETRY_DIR")
	}
}

func TestHandleUploadTelemetryJSON(t *testing.T) {
	cleanupEnv := withTelemetryDir(t)
	defer cleanupEnv()

	h := NewHandler(func(msg string, fields map[string]interface{}) {})

	payload := UploadRequest{
		ScenarioName:   "picker-wheel",
		DeploymentMode: "bundled",
		Source:         "ui-test",
		Events: []map[string]interface{}{
			{"event": "ready"},
			{"event": "dependency_unreachable", "details": map[string]interface{}{"service": "api"}},
		},
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/upload", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Ensure file was written
	out := shared.GetConfigResolver().ResolveTelemetryDir()
	contents, err := os.ReadFile(filepath.Join(out, "picker-wheel.jsonl"))
	if err != nil {
		t.Fatalf("expected telemetry file: %v", err)
	}
	if !strings.Contains(string(contents), "ready") {
		t.Fatalf("expected telemetry content to include event")
	}
}

func TestHandleUploadTelemetryJSONL(t *testing.T) {
	cleanupEnv := withTelemetryDir(t)
	defer cleanupEnv()

	h := NewHandler(func(msg string, fields map[string]interface{}) {})

	body := `{"event":"ready","timestamp":"2024-01-01T00:00:00Z"}
{"event":"dependency_unreachable","details":{"service":"db"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/upload?scenario=demo-app&mode=bundled", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/jsonl")
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// List summaries
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry", nil)
	listRec := httptest.NewRecorder()
	h.List(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected 200 from list, got %d: %s", listRec.Code, listRec.Body.String())
	}

	var summaries []Summary
	if err := json.Unmarshal(listRec.Body.Bytes(), &summaries); err != nil {
		t.Fatalf("decode summaries: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}
	if summaries[0].Scenario != "demo-app" {
		t.Fatalf("unexpected scenario: %s", summaries[0].Scenario)
	}
	if summaries[0].TotalEvents != 2 {
		t.Fatalf("expected 2 events, got %d", summaries[0].TotalEvents)
	}
	if summaries[0].FailureCounts["dependency_unreachable"] != 1 {
		t.Fatalf("expected failure count recorded")
	}
}
