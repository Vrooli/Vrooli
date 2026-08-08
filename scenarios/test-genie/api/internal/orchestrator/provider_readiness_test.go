package orchestrator

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"
	"time"

	"test-genie/internal/captureprofile"
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

func TestExecutionConfigurationFingerprintIgnoresCaptureDepth(t *testing.T) {
	defaultRequest := SuiteExecutionRequest{Preset: "comprehensive", CaptureProfile: captureprofile.NameDefault}
	baselineRequest := defaultRequest
	baselineRequest.CaptureProfile = captureprofile.NameBaseline

	if got, want := ExecutionConfigurationFingerprint(defaultRequest, "descriptor:digest"), ExecutionConfigurationFingerprint(baselineRequest, "descriptor:digest"); got != want {
		t.Fatalf("capture depth changed validation configuration fingerprint: default=%q baseline=%q", got, want)
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
		providerDef(phases.Name("unit"), "unit-health", required),
		providerDef(phases.Name("docs"), "knowledge-observatory", bestEffort),
		staticDef(phases.Name("structure")),
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
		staticDef(phases.Name("structure")),
		providerDef(phases.Name("unit"), "unit-health", phasepolicy.RequiredProviderPolicy()),
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

func TestCheckProviderReadinessConcurrentPreservesOutcomeOrderAndProviderExclusion(t *testing.T) {
	t.Setenv("TEST_GENIE_PROVIDER_READINESS_CONCURRENCY", "4")
	var mu sync.Mutex
	inFlight := map[string]int{}
	maxInFlight := map[string]int{}
	o := &SuiteOrchestrator{readiness: &providerreadiness.Manager{
		Probe: func(_ context.Context, in providerreadiness.Input) (providerreadiness.ProbeResult, error) {
			mu.Lock()
			inFlight[in.ProviderScenario]++
			if inFlight[in.ProviderScenario] > maxInFlight[in.ProviderScenario] {
				maxInFlight[in.ProviderScenario] = inFlight[in.ProviderScenario]
			}
			mu.Unlock()
			time.Sleep(20 * time.Millisecond)
			mu.Lock()
			inFlight[in.ProviderScenario]--
			mu.Unlock()
			return providerreadiness.ProbeResult{Reachable: true, ContractValid: true, IdentityMatch: true}, nil
		}, Lifecycle: noOpProviderLifecycle{},
	}}
	required := phasepolicy.RequiredProviderPolicy()
	defs := []phases.Definition{
		providerDef(phases.Name("unit"), "shared-health", required),
		providerDef(phases.Name("docs"), "docs-health", required),
		providerDef(phases.Name("security"), "shared-health", required),
	}
	got := o.checkProviderReadiness(context.Background(), workspacepkg.Environment{ScenarioName: "demo", ScenarioDir: t.TempDir()}, defs, io.Discard, nil)
	if len(got.Active) != len(defs) || len(got.Stages) != len(defs) {
		t.Fatalf("readiness result = %+v, want all phases active and staged", got)
	}
	if got.Stages[0].Subject != "shared-health" || got.Stages[1].Subject != "docs-health" || got.Stages[2].Subject != "shared-health" {
		t.Fatalf("stages lost definition order: %+v", got.Stages)
	}
	mu.Lock()
	defer mu.Unlock()
	if maxInFlight["shared-health"] > 1 {
		t.Fatalf("same provider was probed concurrently: %v", maxInFlight)
	}
}

func TestCheckProviderReadinessSerialAndConcurrentPreserveOutcomes(t *testing.T) {
	required := phasepolicy.RequiredProviderPolicy()
	required.ProviderLifecycle = phasepolicy.ProviderLifecycleCheckOnly
	defs := []phases.Definition{
		providerDef(phases.Name("unit"), "ready-health", required),
		providerDef(phases.Name("docs"), "contract-health", required),
		providerDef(phases.Name("security"), "ready-health", required),
		providerDef(phases.Name("quality"), "identity-health", required),
	}
	probe := func(_ context.Context, in providerreadiness.Input) (providerreadiness.ProbeResult, error) {
		switch in.ProviderScenario {
		case "contract-health":
			return providerreadiness.ProbeResult{Reachable: true, ContractValid: false, IdentityMatch: true, Message: "contract mismatch"}, nil
		case "identity-health":
			return providerreadiness.ProbeResult{Reachable: true, ContractValid: true, IdentityMatch: false, Message: "identity mismatch"}, nil
		default:
			return providerreadiness.ProbeResult{Reachable: true, ContractValid: true, IdentityMatch: true}, nil
		}
	}
	check := func(t *testing.T, concurrency string) []providerreadiness.Outcome {
		t.Helper()
		t.Setenv("TEST_GENIE_PROVIDER_READINESS_CONCURRENCY", concurrency)
		o := &SuiteOrchestrator{readiness: &providerreadiness.Manager{Probe: probe, Lifecycle: noOpProviderLifecycle{}}}
		return o.checkProviderReadiness(context.Background(), workspacepkg.Environment{ScenarioName: "demo", ScenarioDir: t.TempDir()}, defs, io.Discard, nil).Outcomes
	}

	serial := check(t, "serial")
	concurrent := check(t, "4")
	if len(serial) != len(concurrent) {
		t.Fatalf("serial outcomes = %d, concurrent outcomes = %d", len(serial), len(concurrent))
	}
	for i := range serial {
		got := []providerreadiness.Outcome{serial[i], concurrent[i]}
		for j := range got {
			got[j].Err = nil
		}
		if !reflect.DeepEqual(got[0], got[1]) {
			t.Fatalf("outcome %d changed between serial and concurrent readiness: serial=%+v concurrent=%+v", i, got[0], got[1])
		}
	}
}

func TestCheckProviderReadinessSerialLever(t *testing.T) {
	t.Setenv("TEST_GENIE_PROVIDER_READINESS_CONCURRENCY", "serial")
	var mu sync.Mutex
	inFlight := 0
	maxInFlight := 0
	o := &SuiteOrchestrator{readiness: &providerreadiness.Manager{
		Probe: func(context.Context, providerreadiness.Input) (providerreadiness.ProbeResult, error) {
			mu.Lock()
			inFlight++
			if inFlight > maxInFlight {
				maxInFlight = inFlight
			}
			mu.Unlock()
			time.Sleep(5 * time.Millisecond)
			mu.Lock()
			inFlight--
			mu.Unlock()
			return providerreadiness.ProbeResult{Reachable: true, ContractValid: true, IdentityMatch: true}, nil
		}, Lifecycle: noOpProviderLifecycle{},
	}}
	defs := []phases.Definition{providerDef(phases.Name("unit"), "one", phasepolicy.RequiredProviderPolicy()), providerDef(phases.Name("docs"), "two", phasepolicy.RequiredProviderPolicy())}
	o.checkProviderReadiness(context.Background(), workspacepkg.Environment{ScenarioName: "demo", ScenarioDir: t.TempDir()}, defs, io.Discard, nil)
	if maxInFlight != 1 {
		t.Fatalf("serial readiness max in-flight = %d, want 1", maxInFlight)
	}
}

func TestSynchronizedWriterAcceptsAbsentRunLog(t *testing.T) {
	w := &synchronizedWriter{}
	payload := []byte("readiness output")
	if n, err := w.Write(payload); err != nil || n != len(payload) {
		t.Fatalf("nil writer write = (%d, %v), want discarded successful write", n, err)
	}
}

type noOpProviderLifecycle struct{}

func (noOpProviderLifecycle) Start(context.Context, string, io.Writer) error {
	return nil
}

func (noOpProviderLifecycle) Restart(context.Context, string, io.Writer) error {
	return nil
}
