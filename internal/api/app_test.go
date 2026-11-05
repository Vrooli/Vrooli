//nolint:goconst // test data deliberately reuses stable API fixture values.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/maintenance"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=4 | LAST: 2026-04-13

func TestProjectAPIWriteTimeoutExceedsProgramRuntimeBridgeTimeout(t *testing.T) {
	if ProjectAPIWriteTimeout <= 90*time.Second {
		t.Fatalf("ProjectAPIWriteTimeout = %s, want greater than the 90s Program Runtime bridge timeout", ProjectAPIWriteTimeout)
	}
}

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
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDisplayName("alpha"),
		testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}),
	))
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
	logger, _ := logx.New(logx.Options{Component: "vrooli-api", Writer: &logs, Format: logx.FormatJSON})
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
	logger, _ := logx.New(logx.Options{Component: "vrooli-api", Writer: &logs, Format: logx.FormatJSON})
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
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDisplayName("alpha"),
		testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}),
	))
	logPath := filepath.Join(home, ".vrooli", "logs", "alpha.log")
	testkitgo.WriteFile(t, logPath, "first\nsecond\n")

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
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDisplayName("alpha"),
		testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}),
	))
	writeAPITestRegistryRuntime(t, home, "alpha", "api", "API_PORT", 18080)

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
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDisplayName("alpha"),
		testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}),
	))
	writeAPITestRegistryRuntime(t, home, "alpha", "api", "API_PORT", 18080)

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
	testscenario.WriteProjectResourceConfig(t, root, "redis", true)
	testresource.WriteExternalCLIResourceFixture(t, root, "redis", shelltest.BashShebang()+"exit 0\n")

	app := New(root, home)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/resources", nil)
	app.ListResources(rec, req)

	payload := decodeJSONMap(t, rec)
	if payload["success"] != true {
		t.Fatalf("success = %v", payload["success"])
	}
	data := payload["data"].(map[string]any)
	items := data["resources"].([]any)
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	resource := items[0].(map[string]any)
	if resource["health"] != "healthy" || resource["status_code"] != "ok" || resource["message"] != "available" {
		t.Fatalf("resource = %#v", resource)
	}
}

func TestHandleLifecycleReturnsProjectError(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectService(t, root, testscenario.ProjectServiceManifest())

	app := New(root, home)
	rec := httptest.NewRecorder()
	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/lifecycle/build", nil), map[string]string{"action": "build"})
	app.HandleLifecycle(rec, req)

	payload := decodeJSONMap(t, rec)
	if payload["success"] != false {
		t.Fatalf("success = %v", payload["success"])
	}
	if !strings.Contains(payload["error"].(string), "native-only") {
		t.Fatalf("error = %v", payload["error"])
	}
}

func TestCollectProcessHealthSnapshotUsesMaintenanceSnapshot(t *testing.T) {
	app := New(t.TempDir(), t.TempDir())
	app.ProcessSnapshotFn = func() (maintenance.ProcessSnapshot, error) {
		return maintenance.ProcessSnapshot{
			ZombieProcesses: 7,
			OrphanProcesses: 4,
		}, nil
	}

	health := app.collectProcessHealthSnapshot()
	if health.ZombieCount != 7 || health.OrphanCount != 4 {
		t.Fatalf("health = %#v", health)
	}
	if health.ZombieStatus != "warning" || health.OrphanStatus != "normal" || health.OverallStatus != "warning" {
		t.Fatalf("health status = %#v", health)
	}
}

func TestGetEnhancedProcessMetricsUsesMaintenanceSnapshot(t *testing.T) {
	app := New(t.TempDir(), t.TempDir())
	app.ProcessSnapshotFn = func() (maintenance.ProcessSnapshot, error) {
		return maintenance.ProcessSnapshot{
			TrackedProcesses: 3,
			RunningTracked:   2,
			ChildProcesses:   5,
			TotalProcesses:   9,
			ZombieProcesses:  1,
			OrphanProcesses:  2,
		}, nil
	}

	metrics := app.getEnhancedProcessMetrics()
	if metrics["tracked_processes"] != 3 || metrics["running_tracked"] != 2 {
		t.Fatalf("metrics = %#v", metrics)
	}
	if metrics["child_processes"] != 5 || metrics["total_processes"] != 9 {
		t.Fatalf("metrics = %#v", metrics)
	}
	if metrics["zombie_processes"] != 1 || metrics["orphan_processes"] != 2 {
		t.Fatalf("metrics = %#v", metrics)
	}
	portLookup, ok := metrics["port_lookup"].(map[string]int64)
	if !ok || portLookup["evaluations"] < 0 || portLookup["peer_hits"] < 0 || portLookup["registry_hits"] < 0 || portLookup["cli_hits"] < 0 {
		t.Fatalf("port lookup metrics = %#v", metrics["port_lookup"])
	}
}

// TestHealthCheckWarningStaysReachable pins the health-semantics contract: a
// warning-level condition (elevated zombie/orphan counts) is advisory, so the
// endpoint answers HTTP 200 with status:"degraded" in the body — never 503,
// which the shared CLI client reads as an unreachable control plane. Only a
// critical condition returns 503/"unhealthy".
func TestHealthCheckWarningStaysReachable(t *testing.T) {
	cases := []struct {
		name        string
		snapshot    maintenance.ProcessSnapshot
		wantHTTP    int
		wantStatus  string
		wantZombies int
	}{
		{
			name:        "warning zombies return 200 degraded",
			snapshot:    maintenance.ProcessSnapshot{ZombieProcesses: 18},
			wantHTTP:    http.StatusOK,
			wantStatus:  "degraded",
			wantZombies: 18,
		},
		{
			name:        "healthy returns 200 healthy",
			snapshot:    maintenance.ProcessSnapshot{},
			wantHTTP:    http.StatusOK,
			wantStatus:  "healthy",
			wantZombies: 0,
		},
		{
			name:        "critical zombies return 503 unhealthy",
			snapshot:    maintenance.ProcessSnapshot{ZombieProcesses: 50},
			wantHTTP:    http.StatusServiceUnavailable,
			wantStatus:  "unhealthy",
			wantZombies: 50,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app := New(t.TempDir(), t.TempDir())
			snap := tc.snapshot
			app.ProcessSnapshotFn = func() (maintenance.ProcessSnapshot, error) {
				return snap, nil
			}
			rec := httptest.NewRecorder()
			app.HealthCheck(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

			if rec.Code != tc.wantHTTP {
				t.Fatalf("HTTP status = %d, want %d (body=%s)", rec.Code, tc.wantHTTP, rec.Body.String())
			}
			payload := decodeJSONMap(t, rec)
			if payload["status"] != tc.wantStatus {
				t.Fatalf("status = %v, want %q", payload["status"], tc.wantStatus)
			}
			system, ok := payload["system"].(map[string]any)
			if !ok {
				t.Fatalf("system block missing or wrong shape: %v", payload["system"])
			}
			if got := int(system["zombie_processes"].(float64)); got != tc.wantZombies {
				t.Fatalf("zombie_processes = %d, want %d", got, tc.wantZombies)
			}
		})
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
