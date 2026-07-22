package orchestrator

import (
	"context"
	"errors"
	"io"
	"testing"

	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/providerreadiness"
	"test-genie/internal/orchestrator/runnability"
	workspacepkg "test-genie/internal/orchestrator/workspace"
)

func providerDef(name phases.Name, provider string, policy phasepolicy.Policy) phases.Definition {
	return phases.Definition{
		Name:             name,
		Runner:           passingRunner,
		ProviderScenario: provider,
		Policy:           policy,
		Capabilities:     runnability.PhaseCapabilities{Phase: name.String()},
	}
}

func TestCheckProviderReadinessBlocksRequiredProviderAndKeepsActivePhases(t *testing.T) {
	required := phasepolicy.RequiredProviderPolicy()
	required.ProviderLifecycle = phasepolicy.ProviderLifecycleCheckOnly
	bestEffort := phasepolicy.BestEffortProviderPolicy()
	bestEffort.ProviderLifecycle = phasepolicy.ProviderLifecycleCheckOnly
	probes := 0
	o := &SuiteOrchestrator{
		readiness: &providerreadiness.Manager{
			Probe: func(_ context.Context, in providerreadiness.Input) (providerreadiness.ProbeResult, error) {
				probes++
				if in.Phase == "unit" {
					return providerreadiness.ProbeResult{}, errors.New("unit-health unreachable")
				}
				return providerreadiness.ProbeResult{Reachable: true, ContractValid: true, IdentityMatch: true}, nil
			},
			Lifecycle: noOpProviderLifecycle{},
		},
	}

	defs := []phases.Definition{
		providerDef(phases.Unit, "unit-health", required),
		providerDef(phases.Docs, "knowledge-observatory", bestEffort),
		staticDef(phases.Structure),
	}
	got := o.checkProviderReadiness(context.Background(), workspacepkg.Environment{ScenarioName: "demo", ScenarioDir: t.TempDir()}, defs, io.Discard, nil)

	if probes != 2 {
		t.Fatalf("probes = %d, want only provider-backed selected phases probed", probes)
	}
	if len(got.Active) != 2 {
		t.Fatalf("active = %d, want docs and structure active", len(got.Active))
	}
	if _, blocked := got.Blocked["unit"]; !blocked {
		t.Fatalf("unit should be blocked: %+v", got.Blocked)
	}
	if len(got.Stages) != 2 || got.Stages[0].Name != "provider_check" || got.Stages[0].Parent != "provider_readiness" || got.Stages[0].Subject != "unit-health" {
		t.Fatalf("readiness stages = %+v", got.Stages)
	}
	if got.Stages[0].Status != string(providerreadiness.OutcomeUnreachable) || got.Stages[0].DurationMilliseconds < 0 {
		t.Fatalf("required provider stage = %+v", got.Stages[0])
	}
}

func TestRunSelectedPhasesReturnsProviderReadinessResultInOrder(t *testing.T) {
	o := &SuiteOrchestrator{projectRoot: t.TempDir(), phaseTimeout: phases.DefaultTimeout}
	runLogDir := t.TempDir()
	defs := []phases.Definition{
		staticDef(phases.Structure),
		providerDef(phases.Unit, "unit-health", phasepolicy.RequiredProviderPolicy()),
	}
	blocked := map[string]providerreadiness.Outcome{
		"unit": {
			Phase:            "unit",
			ProviderScenario: "unit-health",
			Status:           providerreadiness.OutcomeUnreachable,
			Message:          "connection refused",
			Err:              errors.New("connection refused"),
		},
	}

	results, anyFailure := o.runSelectedPhasesWithEvents(context.Background(), workspacepkg.Environment{}, runnability.RunContext{}, runLogDir, defs, blocked, false, nil, nil)
	if !anyFailure {
		t.Fatal("required provider readiness failure should mark the run as failed")
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if results[0].Name != "structure" || results[0].Status != phaseStatusPassed {
		t.Fatalf("first result = %+v, want passing structure", results[0])
	}
	if results[1].Name != "unit" || results[1].Status != phaseStatusProviderUnavailable {
		t.Fatalf("second result = %+v, want provider_unavailable unit", results[1])
	}
	if results[1].Classification != phases.FailureClassMissingDependency {
		t.Fatalf("classification = %q, want missing dependency", results[1].Classification)
	}
}

type noOpProviderLifecycle struct{}

func (noOpProviderLifecycle) Start(context.Context, string, io.Writer) error {
	return nil
}

func (noOpProviderLifecycle) Restart(context.Context, string, io.Writer) error {
	return nil
}
