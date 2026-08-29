package scenarioapp

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/process"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type fakeScenarioOps struct {
	started      []string
	detail       orchestrator.Detail
	detailAtPath string
}

func (f *fakeScenarioOps) StartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error) {
	f.started = append(f.started, name)
	return orchestrator.StartResult{
		Scenario: scenariomodel.Scenario{Slug: name, Manifest: scenariomodel.ServiceManifest{
			Ports: map[string]scenariomodel.Port{
				"api": {EnvVar: "API_PORT", Description: "Backend"},
			},
		}},
		AllocatedPorts:     map[string]int{"api": 8080},
		Details:            scenariomodel.RuntimeDetails{Ports: map[string]int{"API_PORT": 8080}},
		FailedDependencies: nil,
		FailedResources:    []string{"qdrant"},
	}, nil
}

func (f *fakeScenarioOps) RestartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error) {
	return orchestrator.StartResult{}, nil
}
func (f *fakeScenarioOps) Inventory() ([]orchestrator.Detail, error) { return nil, nil }
func (f *fakeScenarioOps) InventoryReport() (orchestrator.InventoryReport, error) {
	return orchestrator.InventoryReport{}, nil
}

func (f *fakeScenarioOps) Detail(name string) (orchestrator.Detail, error) {
	if f.detail.Scenario.Slug != "" || f.detail.Details.Status != "" {
		return f.detail, nil
	}
	return orchestrator.Detail{Runtime: process.ScenarioRuntime{}}, nil
}

func (f *fakeScenarioOps) DetailAtPath(_ string, path string) (orchestrator.Detail, error) {
	f.detailAtPath = path
	return f.Detail("")
}

func (f *fakeScenarioOps) StartAll() (control.StartReport, error) {
	return control.StartReport{}, nil
}

func (f *fakeScenarioOps) StopAll() (control.StopReport, error) {
	return control.StopReport{}, nil
}

func (f *fakeScenarioOps) ResolvePort(name, portName string) (orchestrator.ResolvedPort, error) {
	return orchestrator.ResolvedPort{Name: portName, Port: 8080, URL: "http://localhost:8080"}, nil
}

type fakeRunner struct{}

func (fakeRunner) Stop(name string, opts lifecycle.StopOptions) error { return nil }
func (fakeRunner) RunPhaseDetailed(name, phase string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error) {
	return lifecycle.PhaseResult{}, nil
}
func (fakeRunner) RunPhase(name, phase string, opts lifecycle.PhaseOptions) error { return nil }
func (fakeRunner) FreshnessReportByName(name, customPath string) (lifecycle.FreshnessReport, error) {
	return lifecycle.FreshnessReport{}, nil
}

func (fakeRunner) WaitScenario(name string, opts lifecycle.WaitOptions) (lifecycle.WaitOutcome, error) {
	return lifecycle.WaitOutcome{Scenario: name, Verdict: lifecycle.WaitVerdictHealthy}, nil
}

func TestStartUsesScenarioOperationsInterface(t *testing.T) {
	ops := &fakeScenarioOps{}
	svc := Service{Scenarios: ops, Runner: fakeRunner{}}

	items, err := svc.Start(StartRequest{Names: []string{"demo"}})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if len(items) != 1 || items[0].Name != "demo" {
		t.Fatalf("items = %#v", items)
	}
	if len(items[0].Endpoints) != 1 || items[0].Endpoints[0].URL != "http://localhost:8080" {
		t.Fatalf("items[0].Endpoints = %#v", items[0].Endpoints)
	}
	if len(items[0].FailedResources) != 1 || items[0].FailedResources[0] != "qdrant" {
		t.Fatalf("items[0].FailedResources = %#v", items[0].FailedResources)
	}
	if len(ops.started) != 1 || ops.started[0] != "demo" {
		t.Fatalf("started = %#v", ops.started)
	}
}

func TestStatusBuildersPreserveHealthError(t *testing.T) {
	detail := orchestrator.Detail{
		Scenario: scenariomodel.Scenario{Slug: "demo"},
		Details: scenariomodel.RuntimeDetails{
			Status:      "running",
			HealthError: "api_endpoint: invalid health response schema",
		},
	}
	status := BuildStatusDetail(detail)
	if status.HealthError != detail.Details.HealthError {
		t.Fatalf("status HealthError = %q", status.HealthError)
	}
	runtime := BuildRuntimeDataFromDetail(detail)
	if runtime.HealthError != detail.Details.HealthError {
		t.Fatalf("runtime HealthError = %q", runtime.HealthError)
	}
}

func TestPortRejectsSpecificPortWhenRuntimeIsNotRunning(t *testing.T) {
	ops := &fakeScenarioOps{detail: portDetail("starting", 18080)}
	svc := Service{Scenarios: ops, Runner: fakeRunner{}}

	_, err := svc.Port(PortRequest{ScenarioName: "demo", PortName: "API_PORT"})
	if err == nil {
		t.Fatal("Port(API_PORT) error = nil, want non-running runtime error")
	}
	if !strings.Contains(err.Error(), "registry") || !strings.Contains(err.Error(), "starting") {
		t.Fatalf("Port(API_PORT) error = %v, want stale registry context", err)
	}

	resp, err := svc.Port(PortRequest{ScenarioName: "demo", PortName: "API_PORT", JSON: true})
	if err != nil {
		t.Fatalf("Port(API_PORT,json) error = %v", err)
	}
	if resp.Single == nil || resp.Single.Success {
		t.Fatalf("response = %#v, want unsuccessful single-port response", resp)
	}
	if !strings.Contains(resp.Single.Error, "registry") || !strings.Contains(resp.Single.Error, "starting") {
		t.Fatalf("response error = %q, want stale registry context", resp.Single.Error)
	}
}

func TestPortResolvesSpecificPortFromRuntimeDetails(t *testing.T) {
	ops := &fakeScenarioOps{detail: portDetail("running", 18080)}
	svc := Service{Scenarios: ops, Runner: fakeRunner{}}

	resp, err := svc.Port(PortRequest{ScenarioName: "demo", PortName: "api"})
	if err != nil {
		t.Fatalf("Port(api) error = %v", err)
	}
	if resp.Single == nil || !resp.Single.Success {
		t.Fatalf("response = %#v, want successful single-port response", resp)
	}
	if resp.Single.PortName != "API_PORT" || resp.Single.Step != "api" || resp.Single.Port != 18080 {
		t.Fatalf("single = %#v, want API_PORT/api/18080", resp.Single)
	}
}

func TestPortUsesExplicitScenarioPath(t *testing.T) {
	ops := &fakeScenarioOps{detail: portDetail("running", 18080)}
	svc := Service{Scenarios: ops, Runner: fakeRunner{}}

	resp, err := svc.Port(PortRequest{ScenarioName: "demo", PortName: "api", Path: "/tmp/generated/scenarios/demo"})
	if err != nil {
		t.Fatalf("Port(api,path) error = %v", err)
	}
	if resp.Single == nil || !resp.Single.Success {
		t.Fatalf("response = %#v, want successful single-port response", resp)
	}
	if got, want := ops.detailAtPath, "/tmp/generated/scenarios/demo"; got != want {
		t.Fatalf("DetailAtPath path = %q, want %q", got, want)
	}
}

func TestParsePort(t *testing.T) {
	port, err := parsePort("API_PORT=18060\n")
	if err != nil {
		t.Fatalf("parsePort: %v", err)
	}
	if port != 18060 {
		t.Fatalf("port = %d, want 18060", port)
	}
	if _, err := parsePort("not running"); err == nil {
		t.Fatal("expected parse error")
	}
}

func portDetail(status string, port int) orchestrator.Detail {
	manifest := scenariomodel.ServiceManifest{
		Ports: map[string]scenariomodel.Port{
			"api": {EnvVar: "API_PORT", Description: "Backend"},
		},
	}
	return orchestrator.Detail{
		Scenario: scenariomodel.Scenario{Slug: "demo", Manifest: manifest},
		Details: scenariomodel.RuntimeDetails{
			Status: status,
			Ports:  map[string]int{"API_PORT": port},
			PortBindings: []scenariomodel.RuntimePortBinding{
				{Key: "API_PORT", Step: "api", Port: port},
			},
		},
	}
}
