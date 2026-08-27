package api

import (
	"bytes"
	"context"
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

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/maintenance"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

// writeAPITestRegistryRuntime seeds an authoritative registry runtime for the
// given scenario with a single bound port claim. Used by the API tests that
// previously relied on legacy process records to make a scenario look running.
func writeAPITestRegistryRuntime(t *testing.T, home, scenarioName, portName, envVar string, port int) {
	t.Helper()
	ctx := context.Background()
	host, err := hostsession.DefaultProvider{}.Current(ctx, home)
	if err != nil {
		t.Fatalf("host session: %v", err)
	}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer store.Close()

	instance, err := store.CreateLease(ctx, scenarioruntime.Instance{
		InstanceID: "inst-" + scenarioName,
		Scenario:   scenarioName,
		Status:     scenarioruntime.StatusStarting,
		Phase:      "develop",
		OwnerKind:  scenarioruntime.OwnerKindLifecycle,
		StartedAt:  time.Now().Add(-time.Minute).UTC(),
		HostBootID: host.BootID,
	}, scenarioruntime.DefaultHeartbeatTTL)
	if err != nil {
		t.Fatalf("CreateLease: %v", err)
	}
	if _, err := store.UpdateInstanceStatus(ctx, instance.InstanceID, instance.Generation, scenarioruntime.StatusRunning, "develop"); err != nil {
		t.Fatalf("UpdateInstanceStatus: %v", err)
	}
	claim, err := store.AcquirePortClaim(ctx, scenarioruntime.PortClaim{
		ClaimID:    "claim-" + scenarioName + "-" + portName,
		InstanceID: instance.InstanceID,
		Scenario:   scenarioName,
		PortName:   portName,
		EnvVar:     envVar,
		Port:       port,
		BindHost:   "127.0.0.1",
		Status:     scenarioruntime.ClaimStatusReserved,
	})
	if err != nil {
		t.Fatalf("AcquirePortClaim: %v", err)
	}
	if _, err := store.BindPortClaim(ctx, claim.ClaimID); err != nil {
		t.Fatalf("BindPortClaim: %v", err)
	}
	pid := os.Getpid()
	if _, err := store.AddProcessRef(ctx, scenarioruntime.ProcessRef{
		RefID:      "proc-" + instance.InstanceID,
		InstanceID: instance.InstanceID,
		PID:        &pid,
		PGID:       &pid,
		Step:       "start-" + portName,
		Status:     "running",
		StartedAt:  time.Now().Add(-time.Minute).UTC(),
		HostBootID: host.BootID,
	}); err != nil {
		t.Fatalf("AddProcessRef: %v", err)
	}
}

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=4 | LAST: 2026-04-13

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
}

func decodeJSONMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v\nbody=%s", err, rec.Body.String())
	}
	return payload
}
