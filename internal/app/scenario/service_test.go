package scenarioapp

import (
	"testing"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/process"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type fakeScenarioOps struct {
	started []string
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
	}, nil
}

func (f *fakeScenarioOps) RestartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error) {
	return orchestrator.StartResult{}, nil
}
func (f *fakeScenarioOps) Inventory() ([]orchestrator.Detail, error) { return nil, nil }
func (f *fakeScenarioOps) Detail(name string) (orchestrator.Detail, error) {
	return orchestrator.Detail{Runtime: process.ScenarioRuntime{}}, nil
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
	if len(ops.started) != 1 || ops.started[0] != "demo" {
		t.Fatalf("started = %#v", ops.started)
	}
}
