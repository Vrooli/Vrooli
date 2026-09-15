package api

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
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

func TestRespondErrorSetsHTTPStatusAndCode(t *testing.T) {
	rec := httptest.NewRecorder()
	respondError(rec, newAPIError(http.StatusNotFound, "scenario_not_found", "scenario not found", errors.New("missing")))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("rec.Code = %d, want %d", rec.Code, http.StatusNotFound)
	}
	payload := decodeJSONMap(t, rec)
	if payload["success"] != false {
		t.Fatalf("success = %v", payload["success"])
	}
	if payload["error_code"] != "scenario_not_found" {
		t.Fatalf("error_code = %v", payload["error_code"])
	}
}

func TestRespondErrorAssignsFallbackCodeToUntypedError(t *testing.T) {
	rec := httptest.NewRecorder()
	respondError(rec, errors.New("plain failure"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("rec.Code = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	payload := decodeJSONMap(t, rec)
	if payload["error_code"] != "internal_error" || payload["error"] != "plain failure" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestGetScenarioStatusNativeReturnsRealProcessDataAndStatusCode(t *testing.T) {
	// Open a live listener so the registry's bound claim passes reconciliation
	// (which verifies listener evidence for bound claims).
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDisplayName("alpha"),
		testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}),
	))
	writeAPITestRegistryRuntime(t, home, "alpha", "api", "API_PORT", port)

	app := New(root, home)
	rec := httptest.NewRecorder()
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/scenarios/alpha/status", nil), map[string]string{"name": "alpha"})
	app.GetScenarioStatusNative(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("rec.Code = %d, want %d", rec.Code, http.StatusOK)
	}
	payload := decodeJSONMap(t, rec)
	data := payload["data"].(map[string]any)
	processes := data["processes"].([]any)
	if len(processes) != 1 {
		t.Fatalf("len(processes) = %d, want 1", len(processes))
	}
	processData := processes[0].(map[string]any)
	if int(processData["pid"].(float64)) != os.Getpid() {
		t.Fatalf("pid = %v, want %d", processData["pid"], os.Getpid())
	}
	if processData["step_name"] != "start-api" {
		t.Fatalf("step_name = %v", processData["step_name"])
	}
	allocated := data["allocated_ports"].(map[string]any)
	if int(allocated["API_PORT"].(float64)) != port {
		t.Fatalf("allocated_ports = %#v, want API_PORT %d", allocated, port)
	}
}

func TestGetScenarioStatusNativeReturnsNotFoundStatus(t *testing.T) {
	app := New(t.TempDir(), t.TempDir())
	rec := httptest.NewRecorder()
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/scenarios/missing/status", nil), map[string]string{"name": "missing"})
	app.GetScenarioStatusNative(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("rec.Code = %d, want %d", rec.Code, http.StatusNotFound)
	}
	payload := decodeJSONMap(t, rec)
	if payload["error_code"] != "scenario_not_found" {
		t.Fatalf("error_code = %v", payload["error_code"])
	}
}
