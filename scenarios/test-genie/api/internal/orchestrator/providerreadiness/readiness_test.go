package providerreadiness

import (
	"context"
	"errors"
	"io"
	"testing"

	"test-genie/internal/orchestrator/phasepolicy"
)

type fakeLifecycle struct {
	starts   int
	restarts int
	err      error
}

func (f *fakeLifecycle) Start(context.Context, string, io.Writer) error {
	f.starts++
	return f.err
}

func (f *fakeLifecycle) Restart(context.Context, string, io.Writer) error {
	f.restarts++
	return f.err
}

func TestCheckStartIfNeededDoesNotStartWhenReachable(t *testing.T) {
	lifecycle := &fakeLifecycle{}
	manager := &Manager{
		Lifecycle: lifecycle,
		Probe: func(context.Context, Input) (ProbeResult, error) {
			return ProbeResult{Reachable: true, ContractValid: true, IdentityMatch: true}, nil
		},
	}

	got := manager.Check(context.Background(), inputWithPolicy(phasepolicy.RequiredProviderPolicy()), io.Discard)
	if !got.Ready || got.Status != OutcomeReady {
		t.Fatalf("outcome = %+v, want ready", got)
	}
	if lifecycle.starts != 0 {
		t.Fatalf("starts = %d, want 0", lifecycle.starts)
	}
}

func TestCheckStartIfNeededStartsAfterUnreachableProbe(t *testing.T) {
	lifecycle := &fakeLifecycle{}
	probes := 0
	manager := &Manager{
		Lifecycle: lifecycle,
		Probe: func(context.Context, Input) (ProbeResult, error) {
			probes++
			if probes == 1 {
				return ProbeResult{}, errors.New("connection refused")
			}
			return ProbeResult{Reachable: true, ContractValid: true, IdentityMatch: true}, nil
		},
	}

	got := manager.Check(context.Background(), inputWithPolicy(phasepolicy.RequiredProviderPolicy()), io.Discard)
	if !got.Ready || got.Status != OutcomeStarted || !got.Started {
		t.Fatalf("outcome = %+v, want started ready", got)
	}
	if lifecycle.starts != 1 {
		t.Fatalf("starts = %d, want 1", lifecycle.starts)
	}
}

func TestCheckRestartBeforeProbeRestartsExactlyOnce(t *testing.T) {
	lifecycle := &fakeLifecycle{}
	policy := phasepolicy.RequiredProviderPolicy()
	policy.ProviderLifecycle = phasepolicy.ProviderLifecycleRestartBeforeProbe
	manager := &Manager{
		Lifecycle: lifecycle,
		Probe: func(context.Context, Input) (ProbeResult, error) {
			return ProbeResult{Reachable: true, ContractValid: true, IdentityMatch: true}, nil
		},
	}

	got := manager.Check(context.Background(), inputWithPolicy(policy), io.Discard)
	if !got.Ready || got.Status != OutcomeRestarted || !got.Restarted {
		t.Fatalf("outcome = %+v, want restarted ready", got)
	}
	if lifecycle.restarts != 1 {
		t.Fatalf("restarts = %d, want 1", lifecycle.restarts)
	}
}

func TestCheckRequiredProviderUnreachableBlocks(t *testing.T) {
	manager := &Manager{
		Probe: func(context.Context, Input) (ProbeResult, error) {
			return ProbeResult{}, errors.New("dial tcp failed")
		},
		Lifecycle: &fakeLifecycle{err: errors.New("start failed")},
	}

	got := manager.Check(context.Background(), inputWithPolicy(phasepolicy.RequiredProviderPolicy()), io.Discard)
	if got.Ready || got.Status != OutcomeUnreachable || got.SkipsWithoutFailure() {
		t.Fatalf("outcome = %+v, want blocking unreachable", got)
	}
}

func TestCheckBestEffortProviderUnreachableSkips(t *testing.T) {
	manager := &Manager{
		Probe: func(context.Context, Input) (ProbeResult, error) {
			return ProbeResult{}, errors.New("dial tcp failed")
		},
		Lifecycle: &fakeLifecycle{err: errors.New("start failed")},
	}

	got := manager.Check(context.Background(), inputWithPolicy(phasepolicy.BestEffortProviderPolicy()), io.Discard)
	if got.Ready || got.Status != OutcomeSkippedBestEffort || !got.SkipsWithoutFailure() {
		t.Fatalf("outcome = %+v, want best-effort skip", got)
	}
}

func TestCheckContractInvalidBlocksRequiredProvider(t *testing.T) {
	manager := &Manager{
		Probe: func(context.Context, Input) (ProbeResult, error) {
			return ProbeResult{Reachable: true, ContractValid: false, IdentityMatch: true, Message: "bad contract"}, nil
		},
	}

	got := manager.Check(context.Background(), inputWithPolicy(phasepolicy.RequiredProviderPolicy()), io.Discard)
	if got.Ready || got.Status != OutcomeContractInvalid {
		t.Fatalf("outcome = %+v, want contract invalid", got)
	}
}

func TestCheckNoProviderReadinessPolicySkipsProviderWork(t *testing.T) {
	called := false
	policy := phasepolicy.RequiredProviderPolicy()
	policy.ProviderReadiness = phasepolicy.ProviderReadinessNone
	policy.ProviderLifecycle = phasepolicy.ProviderLifecycleNone
	policy.Freshness = phasepolicy.FreshnessNone
	manager := &Manager{
		Probe: func(context.Context, Input) (ProbeResult, error) {
			called = true
			return ProbeResult{}, nil
		},
	}

	got := manager.Check(context.Background(), inputWithPolicy(policy), io.Discard)
	if !got.Ready || called {
		t.Fatalf("outcome = %+v called=%v, want ready with no probe", got, called)
	}
}

func inputWithPolicy(policy phasepolicy.Policy) Input {
	return Input{
		Phase:            "unit",
		ProviderScenario: "unit-health",
		TargetScenario:   "example",
		TargetPath:       "/tmp/example",
		Policy:           policy,
	}
}
