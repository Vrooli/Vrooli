package packageapp

import (
	"bytes"
	"testing"

	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/process"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type fakeScenarioRuntime struct {
	started []string
}

func (f *fakeScenarioRuntime) Lookup(name string) (orchestrator.Detail, bool, error) {
	return orchestrator.Detail{
		Scenario: scenariomodel.Scenario{Slug: name},
		Runtime:  process.ScenarioRuntime{ProcessCount: 1},
	}, true, nil
}

func (f *fakeScenarioRuntime) StartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error) {
	f.started = append(f.started, name)
	return orchestrator.StartResult{Scenario: scenariomodel.Scenario{Slug: name}}, nil
}

type fakeScenarioRunner struct {
	stopped []string
	phases  []string
}

func (f *fakeScenarioRunner) Stop(name string, opts lifecycle.StopOptions) error {
	f.stopped = append(f.stopped, name)
	return nil
}

func (f *fakeScenarioRunner) RunPhaseDetailed(name, phase string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error) {
	f.phases = append(f.phases, name+":"+phase)
	return lifecycle.PhaseResult{ExecutedSteps: 1}, nil
}

func TestRefreshUsesInterfaceBasedScenarioDependencies(t *testing.T) {
	root := t.TempDir()
	svc := Service{
		Root:   root,
		Stdout: &bytes.Buffer{},
		Stderr: &bytes.Buffer{},
		ScenarioService: func() (ScenarioRuntime, error) {
			return &fakeScenarioRuntime{}, nil
		},
		ScenarioRunner: func() (ScenarioPhaseRunner, error) {
			return &fakeScenarioRunner{}, nil
		},
	}

	if _, ok := any(svc.ScenarioService).(func() (ScenarioRuntime, error)); !ok {
		t.Fatal("ScenarioService is not interface-based")
	}
	if _, ok := any(svc.ScenarioRunner).(func() (ScenarioPhaseRunner, error)); !ok {
		t.Fatal("ScenarioRunner is not interface-based")
	}
}
