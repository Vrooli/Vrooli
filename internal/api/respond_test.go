package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	testprocess "github.com/vrooli/vrooli/packages/testkit-go/processfixture"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

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

func TestGetScenarioStatusNativeReturnsRealProcessDataAndStatusCode(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteScenarioService(t, root, "alpha", testscenario.ScenarioServiceManifest(
		"alpha",
		testscenario.WithDisplayName("alpha"),
		testscenario.WithPorts(map[string]scenario.Port{"api": {EnvVar: "API_PORT"}}),
	))
	testprocess.WriteScenarioProcessRecord(t, home, "alpha", "start-api", process.Record{
		PID:       os.Getpid(),
		PGID:      os.Getpid(),
		Scenario:  "alpha",
		Step:      "start-api",
		Port:      18080,
		StartedAt: time.Now().Add(-time.Minute).UTC(),
		Status:    "running",
	})

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
	if allocated := data["allocated_ports"].(map[string]any); int(allocated["API_PORT"].(float64)) != 18080 {
		t.Fatalf("allocated_ports = %#v", allocated)
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
