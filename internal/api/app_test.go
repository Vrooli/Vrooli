package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/process"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=2 | LAST: 2026-04-11

func TestStartAllScenariosEndpointReturnsTypedReport(t *testing.T) {
	app := New(t.TempDir(), t.TempDir())
	app.StartAllScenariosFn = func() (control.StartReport, error) {
		return control.StartReport{
			Started: []control.ResultItem{{Name: "alpha", Message: "Started successfully"}},
			Failed:  []control.ResultItem{{Name: "beta", Error: "boom"}},
			Message: "Started 1 scenarios, 1 failed",
		}, nil
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/scenarios/start-all", nil)
	app.StartAllScenariosEndpoint(rec, req)

	payload := decodeJSONMap(t, rec)
	if payload["success"] != true {
		t.Fatalf("success = %v", payload["success"])
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("data = %#v", payload["data"])
	}
	if data["message"] != "Started 1 scenarios, 1 failed" {
		t.Fatalf("data.message = %v", data["message"])
	}
	started, ok := data["started"].([]any)
	if !ok || len(started) != 1 {
		t.Fatalf("data.started = %#v", data["started"])
	}
}

func TestStopAllScenariosEndpointReturnsTypedReport(t *testing.T) {
	app := New(t.TempDir(), t.TempDir())
	app.StopAllScenariosFn = func() (control.StopReport, error) {
		return control.StopReport{
			Stopped: []control.ResultItem{{Name: "alpha", Message: "Stopped successfully"}},
			Failed:  []control.ResultItem{},
			Message: "Stopped 1 scenarios, 0 failed",
		}, nil
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/scenarios/stop-all", nil)
	app.StopAllScenariosEndpoint(rec, req)

	payload := decodeJSONMap(t, rec)
	if payload["success"] != true {
		t.Fatalf("success = %v", payload["success"])
	}
	data := payload["data"].(map[string]any)
	if data["message"] != "Stopped 1 scenarios, 0 failed" {
		t.Fatalf("data.message = %v", data["message"])
	}
}

func TestStopScenarioEndpointReturnsTypedMessage(t *testing.T) {
	root := t.TempDir()
	writeScenarioService(t, root, "alpha")
	app := New(root, t.TempDir())
	app.StopScenarioFn = func(name string) error { return nil }

	rec := httptest.NewRecorder()
	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/scenarios/alpha/stop", nil), map[string]string{"name": "alpha"})
	app.StopScenarioEndpoint(rec, req)

	payload := decodeJSONMap(t, rec)
	data := payload["data"].(map[string]any)
	if data["message"] != "Scenario alpha stopped successfully" {
		t.Fatalf("data.message = %v", data["message"])
	}
}

func TestStartAppLogsFailures(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	var logs bytes.Buffer
	logger, _ := logx.New(logx.Options{Component: "vrooli-api", Writer: &logs, JSON: true})
	originalDefault := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(originalDefault) })

	app := New(root, home, logger)
	rec := httptest.NewRecorder()
	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/apps/missing/start", nil), map[string]string{"name": "missing"})
	app.StartApp(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(logs.String(), `"msg":"Scenario start requested for missing scenario"`) {
		t.Fatalf("expected missing-scenario log, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), `"scenario":"missing"`) {
		t.Fatalf("expected structured scenario field, got %q", logs.String())
	}
}

func TestStartAllScenariosEndpointLogsFailures(t *testing.T) {
	var logs bytes.Buffer
	logger, _ := logx.New(logx.Options{Component: "vrooli-api", Writer: &logs, JSON: true})
	app := New(t.TempDir(), t.TempDir(), logger)
	app.StartAllScenariosFn = func() (control.StartReport, error) {
		return control.StartReport{}, errors.New("boom")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/scenarios/start-all", nil)
	app.StartAllScenariosEndpoint(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(logs.String(), `"msg":"Scenario start-all request failed"`) {
		t.Fatalf("expected start-all failure log, got %q", logs.String())
	}
	if !strings.Contains(logs.String(), `"operation":"start_all_scenarios"`) {
		t.Fatalf("expected structured operation field, got %q", logs.String())
	}
}

func TestGetDetailedAppStatusReturnsStoppedPayloadWhenScenarioMissing(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	app := New(root, home)

	rec := httptest.NewRecorder()
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/apps/missing/status", nil), map[string]string{"name": "missing"})
	app.GetDetailedAppStatus(rec, req)

	payload := decodeJSONMap(t, rec)
	data := payload["data"].(map[string]any)
	if data["status"] != "stopped" {
		t.Fatalf("status = %v", data["status"])
	}
	if data["runtime"] != "N/A" {
		t.Fatalf("runtime = %v", data["runtime"])
	}
}

func TestGetAppLogsReturnsTypedPayload(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioService(t, root, "alpha")
	logPath := filepath.Join(home, ".vrooli", "logs", "alpha.log")
	if err := osWriteFileAll(logPath, "first\nsecond\n"); err != nil {
		t.Fatalf("write log: %v", err)
	}

	app := New(root, home)
	rec := httptest.NewRecorder()
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/apps/alpha/logs?lines=1", nil), map[string]string{"name": "alpha"})
	app.GetAppLogs(rec, req)

	payload := decodeJSONMap(t, rec)
	data := payload["data"].(map[string]any)
	if data["scenario"] != "alpha" {
		t.Fatalf("scenario = %v", data["scenario"])
	}
	if !strings.Contains(data["logs"].(string), "second") {
		t.Fatalf("logs = %q", data["logs"])
	}
}

func TestListScenariosNativeReturnsScenarioData(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioService(t, root, "alpha")
	writeScenarioProcess(t, home, "alpha", 18080)

	app := New(root, home)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/scenarios", nil)
	app.ListScenariosNative(rec, req)

	payload := decodeJSONMap(t, rec)
	if payload["success"] != true {
		t.Fatalf("success = %v", payload["success"])
	}
	data := payload["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(data))
	}
	scenario := data[0].(map[string]any)
	if scenario["name"] != "alpha" || scenario["status"] != "running" {
		t.Fatalf("scenario = %#v", scenario)
	}
}

func TestGetScenarioStatusNativeReturnsDetailedScenarioData(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	writeScenarioService(t, root, "alpha")
	writeScenarioProcess(t, home, "alpha", 18080)

	app := New(root, home)
	rec := httptest.NewRecorder()
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/scenarios/alpha/status", nil), map[string]string{"name": "alpha"})
	app.GetScenarioStatusNative(rec, req)

	payload := decodeJSONMap(t, rec)
	if payload["success"] != true {
		t.Fatalf("success = %v", payload["success"])
	}
	data := payload["data"].(map[string]any)
	if data["name"] != "alpha" || data["status"] != "running" {
		t.Fatalf("data = %#v", data)
	}
}

func TestListResourcesReturnsTypedStatusPayload(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("PATH", "/usr/bin:/bin")
	writeResourceServiceConfig(t, root, "redis", true)
	writeResourceCLI(t, root, "redis", `{"installed":true,"running":true,"healthy":true,"message":"healthy"}`)

	app := New(root, home)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resources", nil)
	app.ListResources(rec, req)

	payload := decodeJSONMap(t, rec)
	if payload["success"] != true {
		t.Fatalf("success = %v", payload["success"])
	}
	data := payload["data"].([]any)
	if len(data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(data))
	}
	resource := data[0].(map[string]any)
	if resource["message"] != "healthy" {
		t.Fatalf("resource = %#v", resource)
	}
}

func TestHandleLifecycleReturnsProjectError(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	if err := osWriteFileAll(filepath.Join(root, ".vrooli", "service.json"), `{"service":{"name":"project-alpha"}}`); err != nil {
		t.Fatalf("write project service: %v", err)
	}

	app := New(root, home)
	rec := httptest.NewRecorder()
	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/lifecycle/build", nil), map[string]string{"action": "build"})
	app.HandleLifecycle(rec, req)

	payload := decodeJSONMap(t, rec)
	if payload["success"] != false {
		t.Fatalf("success = %v", payload["success"])
	}
	if !strings.Contains(payload["error"].(string), "not defined") {
		t.Fatalf("error = %v", payload["error"])
	}
}

func decodeJSONMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v\nbody=%s", err, rec.Body.String())
	}
	return payload
}

func writeScenarioService(t *testing.T, root, name string) {
	t.Helper()
	servicePath := filepath.Join(root, "scenarios", name, ".vrooli", "service.json")
	if err := osMkdirAll(filepath.Dir(servicePath)); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(servicePath), err)
	}
	data := `{"service":{"name":"` + name + `","displayName":"` + name + `"},"ports":{"api":{"env_var":"API_PORT"}}}`
	if err := osWriteFileAll(servicePath, data); err != nil {
		t.Fatalf("write service: %v", err)
	}
}

func writeScenarioProcess(t *testing.T, home, name string, port int) {
	t.Helper()
	record := process.Record{
		PID:       os.Getpid(),
		PGID:      os.Getpid(),
		Scenario:  name,
		Step:      "start-api",
		Port:      port,
		StartedAt: time.Now().Add(-time.Minute).UTC(),
		Status:    "running",
	}
	if err := process.WriteScenarioRecord(home, name, "start-api", record); err != nil {
		t.Fatalf("WriteScenarioRecord: %v", err)
	}
}

func writeResourceServiceConfig(t *testing.T, root, name string, enabled bool) {
	t.Helper()
	configPath := filepath.Join(root, ".vrooli", "service.json")
	payload := map[string]any{
		"dependencies": map[string]any{
			"resources": map[string]any{
				name: map[string]any{"enabled": enabled},
			},
		},
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := osWriteFileAll(configPath, string(data)+"\n"); err != nil {
		t.Fatalf("write resource config: %v", err)
	}
}

func writeResourceCLI(t *testing.T, root, name, statusJSON string) {
	t.Helper()
	manifestPath := filepath.Join(root, "resources", name, "resource.json")
	manifest := `{
  "name": "` + name + `",
  "display_name": "` + name + `",
  "description": "fixture resource",
  "template": "legacy-adapter",
  "driver": "legacy-adapter",
  "legacy_adapter": {
    "owner": "Matthew Halloran",
    "decision_deadline": "2026-05-31",
    "final_disposition": "migrate",
    "legacy_cli_path": "resources/` + name + `/cli.sh"
  },
  "portability_tier": "partial"
}`
	if err := osWriteFileAll(manifestPath, manifest+"\n"); err != nil {
		t.Fatalf("write resource manifest: %v", err)
	}
	scriptPath := filepath.Join(root, "resources", name, "cli.sh")
	script := "#!/usr/bin/env bash\nset -e\nif [[ \"$1\" == \"status\" ]]; then\n  printf '%s\\n' '" + statusJSON + "'\n  exit 0\nfi\nprintf '{\"message\":\"ok\"}\\n'\nexit 0\n"
	if err := osWriteFileAll(scriptPath, script); err != nil {
		t.Fatalf("write resource cli: %v", err)
	}
	if err := os.Chmod(scriptPath, 0o755); err != nil {
		t.Fatalf("chmod resource cli: %v", err)
	}
}

func osMkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func osWriteFileAll(path, data string) error {
	if err := ensureParentDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(data), 0o644)
}

func ensureParentDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}
