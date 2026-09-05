package providerreadiness

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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

func TestCheckSerializesSameProviderAcrossConcurrentRuns(t *testing.T) {
	manager := &Manager{}
	firstStarted := make(chan struct{})
	secondEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	var active int32
	var once sync.Once
	manager.Probe = func(context.Context, Input) (ProbeResult, error) {
		current := atomic.AddInt32(&active, 1)
		if current == 1 {
			once.Do(func() { close(firstStarted) })
			<-releaseFirst
		} else {
			select {
			case <-secondEntered:
			default:
				close(secondEntered)
			}
		}
		atomic.AddInt32(&active, -1)
		return ProbeResult{Reachable: true, ContractValid: true, IdentityMatch: true}, nil
	}
	policy := phasepolicy.RequiredProviderPolicy()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.Check(context.Background(), inputWithPolicy(policy), io.Discard)
	}()
	select {
	case <-firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first provider check did not start")
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.Check(context.Background(), inputWithPolicy(policy), io.Discard)
	}()
	select {
	case <-secondEntered:
		t.Fatal("same-provider checks overlapped across runs")
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseFirst)
	wg.Wait()
}

func TestDefaultProbeBoundsDiscovery(t *testing.T) {
	original := resolveScenarioURL
	originalTimeout := defaultProbeTimeout
	resolveScenarioURL = func(ctx context.Context, _ string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	}
	defaultProbeTimeout = 10 * time.Millisecond
	t.Cleanup(func() {
		resolveScenarioURL = original
		defaultProbeTimeout = originalTimeout
	})

	started := time.Now()
	_, err := DefaultProbe(context.Background(), Input{
		ProviderScenario: "provider",
		TargetScenario:   "target",
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > defaultProbeTimeout+time.Second {
		t.Fatalf("probe took %s, exceeded timeout %s", elapsed, defaultProbeTimeout)
	}
}

func TestRunBoundsLifecycleCommand(t *testing.T) {
	original := commandContext
	originalTimeout := defaultLifecycleTimeout
	commandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, os.Args[0], "-test.run=TestProviderReadinessCommandHelper", "--")
	}
	defaultLifecycleTimeout = 10 * time.Millisecond
	t.Setenv("GO_WANT_PROVIDER_READINESS_HELPER", "1")
	t.Cleanup(func() {
		commandContext = original
		defaultLifecycleTimeout = originalTimeout
	})

	started := time.Now()
	err := run(context.Background(), io.Discard, "scenario", "start", "provider")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
	if elapsed := time.Since(started); elapsed > defaultLifecycleTimeout+time.Second {
		t.Fatalf("lifecycle command took %s, exceeded timeout %s", elapsed, defaultLifecycleTimeout)
	}
}

func TestProviderReadinessCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_PROVIDER_READINESS_HELPER") != "1" {
		return
	}
	time.Sleep(time.Hour)
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
