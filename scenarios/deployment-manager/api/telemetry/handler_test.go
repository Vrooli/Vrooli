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

func TestHandleUploadInvalidJSON(t *testing.T) {
	cleanupEnv := withTelemetryDir(t)
	defer cleanupEnv()

	h := NewHandler(func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/upload", strings.NewReader("not valid json{"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleUploadEmptyEvents(t *testing.T) {
	cleanupEnv := withTelemetryDir(t)
	defer cleanupEnv()

	h := NewHandler(func(msg string, fields map[string]interface{}) {})

	payload := UploadRequest{
		ScenarioName: "test-scenario",
		Events:       []map[string]interface{}{},
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/upload", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty events, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "no telemetry events") {
		t.Errorf("expected error about no events, got: %s", rec.Body.String())
	}
}

func TestHandleUploadInvalidJSONL(t *testing.T) {
	cleanupEnv := withTelemetryDir(t)
	defer cleanupEnv()

	h := NewHandler(func(msg string, fields map[string]interface{}) {})

	body := `{"event":"ready"}
not valid json line`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/upload?scenario=test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/jsonl")
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleListEmptyDir(t *testing.T) {
	cleanupEnv := withTelemetryDir(t)
	defer cleanupEnv()

	h := NewHandler(func(msg string, fields map[string]interface{}) {})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry", nil)
	rec := httptest.NewRecorder()
	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var summaries []Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &summaries); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(summaries) != 0 {
		t.Errorf("expected empty list, got %d summaries", len(summaries))
	}
}

func TestHandleUploadResponseFields(t *testing.T) {
	cleanupEnv := withTelemetryDir(t)
	defer cleanupEnv()

	h := NewHandler(func(msg string, fields map[string]interface{}) {})

	payload := UploadRequest{
		ScenarioName: "test-app",
		Events: []map[string]interface{}{
			{"event": "app_start"},
			{"event": "ready"},
		},
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/upload", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Upload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", resp["status"])
	}
	if resp["events_ingested"].(float64) != 2 {
		t.Errorf("expected events_ingested=2, got %v", resp["events_ingested"])
	}
	if resp["scenario"] != "test-app" {
		t.Errorf("expected scenario=test-app, got %v", resp["scenario"])
	}
	if resp["path"] == nil || resp["path"] == "" {
		t.Error("expected path in response")
	}
}

func TestIsFailureEvent(t *testing.T) {
	tests := []struct {
		event     string
		isFailure bool
	}{
		{EventDependencyUnreachable, true},
		{EventSwapMissingAsset, true},
		{EventAssetMissing, true},
		{EventMigrationFailed, true},
		{EventAPIUnreachable, true},
		{EventSecretsMissing, true},
		{EventHealthFailed, true},
		{EventAppStart, false},
		{EventReady, false},
		{EventShutdown, false},
		{EventRestart, false},
		{"custom_event", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			result := IsFailureEvent(tt.event)
			if result != tt.isFailure {
				t.Errorf("IsFailureEvent(%q) = %v, want %v", tt.event, result, tt.isFailure)
			}
		})
	}
}

func TestIsLifecycleEvent(t *testing.T) {
	tests := []struct {
		event       string
		isLifecycle bool
	}{
		{EventAppStart, true},
		{EventReady, true},
		{EventShutdown, true},
		{EventRestart, true},
		{EventDependencyUnreachable, false},
		{EventMigrationFailed, false},
		{"custom_event", false},
	}

	for _, tt := range tests {
		t.Run(tt.event, func(t *testing.T) {
			result := IsLifecycleEvent(tt.event)
			if result != tt.isLifecycle {
				t.Errorf("IsLifecycleEvent(%q) = %v, want %v", tt.event, result, tt.isLifecycle)
			}
		})
	}
}

func TestGetAllFailureEventTypes(t *testing.T) {
	types := GetAllFailureEventTypes()

	if len(types) == 0 {
		t.Fatal("expected failure event types")
	}

	// All returned types should be failure events
	for _, ev := range types {
		if !IsFailureEvent(ev) {
			t.Errorf("GetAllFailureEventTypes returned %q which is not a failure event", ev)
		}
	}

	// Known failure events should be present
	expected := []string{
		EventDependencyUnreachable,
		EventMigrationFailed,
		EventAssetMissing,
	}
	for _, exp := range expected {
		found := false
		for _, ev := range types {
			if ev == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected %q in failure types", exp)
		}
	}
}

func TestIsJSONLContentType(t *testing.T) {
	tests := []struct {
		contentType string
		isJSONL     bool
	}{
		{ContentTypeJSONL, true},
		{"application/x-jsonl", true},
		{"text/jsonl", true},
		{ContentTypeJSON, false},
		{"application/json", false},
		{"text/plain", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := IsJSONLContentType(tt.contentType)
			if result != tt.isJSONL {
				t.Errorf("IsJSONLContentType(%q) = %v, want %v", tt.contentType, result, tt.isJSONL)
			}
		})
	}
}

func TestParseJSONL(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantLen    int
		wantErr    bool
		firstEvent string
	}{
		{
			name:       "single line",
			input:      `{"event":"ready"}`,
			wantLen:    1,
			firstEvent: "ready",
		},
		{
			name:       "multiple lines",
			input:      `{"event":"start"}` + "\n" + `{"event":"ready"}`,
			wantLen:    2,
			firstEvent: "start",
		},
		{
			name:       "empty lines ignored",
			input:      `{"event":"start"}` + "\n\n" + `{"event":"ready"}` + "\n",
			wantLen:    2,
			firstEvent: "start",
		},
		{
			name:    "invalid JSON line",
			input:   `{"event":"ok"}` + "\n" + `not json`,
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events, err := ParseJSONL(strings.NewReader(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(events) != tt.wantLen {
				t.Errorf("got %d events, want %d", len(events), tt.wantLen)
			}

			if tt.firstEvent != "" && len(events) > 0 {
				if events[0]["event"] != tt.firstEvent {
					t.Errorf("first event = %v, want %s", events[0]["event"], tt.firstEvent)
				}
			}
		})
	}
}

func TestResolveUploadParams(t *testing.T) {
	tests := []struct {
		name         string
		queryParams  map[string]string
		reqBody      *UploadRequest
		wantScenario string
		wantMode     string
		wantSource   string
	}{
		{
			name:         "defaults when nothing provided",
			wantScenario: "unknown",
			wantMode:     "bundled",
			wantSource:   "upload",
		},
		{
			name:         "query params take precedence",
			queryParams:  map[string]string{"scenario": "query-app", "mode": "local", "source": "cli"},
			reqBody:      &UploadRequest{ScenarioName: "body-app", DeploymentMode: "cloud", Source: "ui"},
			wantScenario: "query-app",
			wantMode:     "local",
			wantSource:   "cli",
		},
		{
			name:         "request body fallback",
			reqBody:      &UploadRequest{ScenarioName: "body-app", DeploymentMode: "cloud", Source: "ui"},
			wantScenario: "body-app",
			wantMode:     "cloud",
			wantSource:   "ui",
		},
		{
			name:         "whitespace trimmed",
			queryParams:  map[string]string{"scenario": "  trimmed-app  "},
			wantScenario: "trimmed-app",
			wantMode:     "bundled",
			wantSource:   "upload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			q := req.URL.Query()
			for k, v := range tt.queryParams {
				q.Set(k, v)
			}
			req.URL.RawQuery = q.Encode()

			params := ResolveUploadParams(req.URL.Query(), tt.reqBody)

			if params.Scenario != tt.wantScenario {
				t.Errorf("Scenario = %q, want %q", params.Scenario, tt.wantScenario)
			}
			if params.Mode != tt.wantMode {
				t.Errorf("Mode = %q, want %q", params.Mode, tt.wantMode)
			}
			if params.Source != tt.wantSource {
				t.Errorf("Source = %q, want %q", params.Source, tt.wantSource)
			}
		})
	}
}

func TestEnrichEvents(t *testing.T) {
	params := UploadParams{Scenario: "test-app", Mode: "bundled", Source: "test"}
	timestamp := "2024-01-01T00:00:00Z"

	events := []map[string]interface{}{
		{"event": "start"},
		{"event": "ready", "timestamp": "existing-ts"},
	}

	EnrichEvents(events, params, timestamp)

	// First event should get defaults
	if events[0]["timestamp"] != timestamp {
		t.Errorf("expected timestamp to be set")
	}
	if events[0]["scenario_name"] != "test-app" {
		t.Errorf("expected scenario_name to be set")
	}
	if events[0]["deployment_mode"] != "bundled" {
		t.Errorf("expected deployment_mode to be set")
	}
	if events[0]["source"] != "test" {
		t.Errorf("expected source to be set")
	}

	// Second event should preserve existing timestamp
	if events[1]["timestamp"] != "existing-ts" {
		t.Errorf("existing timestamp should be preserved")
	}
}

func TestSummaryRecentEventsLimit(t *testing.T) {
	cleanupEnv := withTelemetryDir(t)
	defer cleanupEnv()

	h := NewHandler(func(msg string, fields map[string]interface{}) {})

	// Upload more than 5 events
	events := []map[string]interface{}{}
	for i := 0; i < 10; i++ {
		events = append(events, map[string]interface{}{
			"event": "test_event",
			"index": i,
		})
	}

	payload := UploadRequest{
		ScenarioName: "limit-test",
		Events:       events,
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/upload", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Upload(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("upload failed: %d", rec.Code)
	}

	// List and check recent events limit
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry", nil)
	listRec := httptest.NewRecorder()
	h.List(listRec, listReq)

	var summaries []Summary
	json.Unmarshal(listRec.Body.Bytes(), &summaries)

	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	summary := summaries[0]
	if summary.TotalEvents != 10 {
		t.Errorf("expected 10 total events, got %d", summary.TotalEvents)
	}
	// Recent events should be limited to 5
	if len(summary.RecentEvents) > 5 {
		t.Errorf("expected at most 5 recent events, got %d", len(summary.RecentEvents))
	}
}

func TestSummaryFailureTracking(t *testing.T) {
	cleanupEnv := withTelemetryDir(t)
	defer cleanupEnv()

	h := NewHandler(func(msg string, fields map[string]interface{}) {})

	events := []map[string]interface{}{
		{"event": EventAppStart},
		{"event": EventDependencyUnreachable, "service": "db"},
		{"event": EventDependencyUnreachable, "service": "cache"},
		{"event": EventMigrationFailed, "migration": "v2"},
		{"event": EventReady},
	}

	payload := UploadRequest{
		ScenarioName: "failure-test",
		Events:       events,
	}
	data, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/telemetry/upload", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Upload(rec, req)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/telemetry", nil)
	listRec := httptest.NewRecorder()
	h.List(listRec, listReq)

	var summaries []Summary
	json.Unmarshal(listRec.Body.Bytes(), &summaries)

	summary := summaries[0]
	if summary.FailureCounts[EventDependencyUnreachable] != 2 {
		t.Errorf("expected 2 dependency_unreachable failures, got %d", summary.FailureCounts[EventDependencyUnreachable])
	}
	if summary.FailureCounts[EventMigrationFailed] != 1 {
		t.Errorf("expected 1 migration_failed failure, got %d", summary.FailureCounts[EventMigrationFailed])
	}
	// RecentFailures should contain failure events
	if len(summary.RecentFailures) < 3 {
		t.Errorf("expected at least 3 recent failures, got %d", len(summary.RecentFailures))
	}
}
