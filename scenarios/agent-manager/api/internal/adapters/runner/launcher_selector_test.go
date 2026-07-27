// Tests for launcherSelector.Pick: ensure the routing logic between host
// and sandbox launchers is exactly the contract documented on the
// selector. These tests pin the spec for execute/protected-sandbox-agent-
// launch and are shared by every runner that adopts the selector.

package runner_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	adapterrunner "agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
)

// recordingSink captures emitted log events so tests can assert on warnings.
type recordingSink struct {
	mu     sync.Mutex
	events []*domain.RunEvent
}

func (s *recordingSink) Emit(e *domain.RunEvent) error {
	s.mu.Lock()
	s.events = append(s.events, e)
	s.mu.Unlock()
	return nil
}
func (s *recordingSink) Close() error { return nil }
func (s *recordingSink) hasWarning(needle string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.events {
		if e == nil {
			continue
		}
		if msg, ok := e.Data.(*domain.LogEventData); ok && msg != nil {
			if msg.Level == "warn" && strings.Contains(msg.Message, needle) {
				return true
			}
		}
	}
	return false
}

func TestLauncherSelectorPick_TrackingWithFactoryAndIDPicksSandbox(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	sandbox := mocks.NewFakeLauncher("sandbox")
	selector := adapterrunner.NewLauncherSelector(host, mocks.NewFakeSandboxLauncherFactory(sandbox))

	sandboxID := uuid.New()
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeTracking},
	}
	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{ResolvedConfig: cfg, SandboxID: &sandboxID})
	if picked != sandbox {
		t.Errorf("tracking request picked %v; want sandbox launcher", picked)
	}
}

func TestLauncherSelectorPick_UnspecifiedModeUsesHost(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	selector := adapterrunner.NewLauncherSelector(host, mocks.NewFakeSandboxLauncherFactory(mocks.NewFakeLauncher("sandbox")))
	cfg := &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{}} // mode=""
	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{ResolvedConfig: cfg})
	if picked != host {
		t.Errorf("unspecified mode picked %v; want host launcher", picked)
	}
}

// TestLauncherSelectorPick_ProtectedWithFactoryAndIDPicksSandbox is the
// load-bearing case: protected mode + factory wired + SandboxID in request
// → SandboxLauncher.
func TestLauncherSelectorPick_ProtectedWithFactoryAndIDPicksSandbox(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	sandboxLauncher := mocks.NewFakeLauncher("sandbox")
	factory := mocks.NewFakeSandboxLauncherFactory(sandboxLauncher)
	selector := adapterrunner.NewLauncherSelector(host, factory)

	sandboxID := uuid.New()
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{
		ResolvedConfig: cfg,
		SandboxID:      &sandboxID,
	})
	if picked != sandboxLauncher {
		t.Errorf("protected request with factory picked %v; want sandbox launcher", picked)
	}
	calledIDs := factory.CalledIDs()
	if len(calledIDs) != 1 || calledIDs[0] != sandboxID {
		t.Errorf("factory called with sandboxID = %v; want %v", calledIDs, sandboxID)
	}
}

func TestLauncherSelectorPick_ProtectedNoFactoryFallsBackWithWarning(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	selector := adapterrunner.NewLauncherSelector(host, nil) // no factory
	sink := &recordingSink{}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	id := uuid.New()
	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		SandboxID:      &id,
		EventSink:      sink,
	})
	if picked != host {
		t.Errorf("no-factory fallback picked %v; want host launcher", picked)
	}
	if !sink.hasWarning("no SandboxLauncherFactory") {
		t.Errorf("no warning emitted; events=%v", sink.events)
	}
}

func TestLauncherSelectorPick_ProtectedNoSandboxIDFallsBackWithWarning(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	factory := mocks.NewFakeSandboxLauncherFactory(mocks.NewFakeLauncher("sandbox"))
	selector := adapterrunner.NewLauncherSelector(host, factory)
	sink := &recordingSink{}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		// SandboxID intentionally nil
		EventSink: sink,
	})
	if picked != host {
		t.Errorf("nil-SandboxID fallback picked %v; want host launcher", picked)
	}
	if !sink.hasWarning("SandboxID is nil") {
		t.Errorf("no warning emitted; events=%v", sink.events)
	}
	if len(factory.CalledIDs()) != 0 {
		t.Error("factory should not have been consulted when SandboxID is nil")
	}
}

func TestLauncherSelectorPick_FactoryReturnsNilFallsBackWithWarning(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	factory := mocks.NewFakeSandboxLauncherFactory(nil)
	selector := adapterrunner.NewLauncherSelector(host, factory)
	sink := &recordingSink{}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	id := uuid.New()
	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		SandboxID:      &id,
		EventSink:      sink,
	})
	if picked != host {
		t.Errorf("factory-nil fallback picked %v; want host launcher", picked)
	}
	if !sink.hasWarning("factory returned nil") {
		t.Errorf("no warning emitted; events=%v", sink.events)
	}
}

// TestLauncherSelectorPick_ProtectedContainmentGapWarns pins the capability
// honesty contract: when a protected run selects a sandbox whose reported
// containment is missing a protected-mode enforcement, the selector still
// returns the sandbox launcher (tracking value survives) but emits a warn
// run-event naming exactly the absent enforcements plus the effective
// containment.
func TestLauncherSelectorPick_ProtectedContainmentGapWarns(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	sandbox := mocks.NewFakeLauncher("sandbox")
	factory := mocks.NewFakeSandboxLauncherFactory(sandbox)
	// Uncontained sandbox: backend none, no enforcements.
	factory.Containment = &adapterrunner.Containment{
		Level: "none", Backend: "none", Enforcements: []string{},
	}
	selector := adapterrunner.NewLauncherSelector(host, factory)
	sink := &recordingSink{}
	cfg := &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected}}
	id := uuid.New()

	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{
		RunID: uuid.New(), ResolvedConfig: cfg, SandboxID: &id, EventSink: sink,
	})
	if picked != sandbox {
		t.Fatalf("containment gap must not change selection; picked %v, want sandbox", picked)
	}
	// Warn must name every missing protected enforcement and the backend.
	for _, needle := range []string{
		adapterrunner.EnforcementFilesystemWriteContainment,
		adapterrunner.EnforcementNetworkDeny,
		"backend=none",
	} {
		if !sink.hasWarning(needle) {
			t.Errorf("gap warn missing %q; events=%v", needle, sink.events)
		}
	}
}

// TestLauncherSelectorPick_ProtectedFullContainmentNoWarn pins that a fully
// enforced sandbox (all protected enforcements present, including
// loopback-only network) produces no gap warning.
func TestLauncherSelectorPick_ProtectedFullContainmentNoWarn(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	sandbox := mocks.NewFakeLauncher("sandbox")
	factory := mocks.NewFakeSandboxLauncherFactory(sandbox)
	factory.Containment = &adapterrunner.Containment{
		Level:   "required",
		Backend: "bwrap",
		Enforcements: []string{
			adapterrunner.EnforcementFilesystemWriteContainment,
			adapterrunner.EnforcementNetworkDeny,
			adapterrunner.EnforcementPIDNamespace,
			adapterrunner.EnforcementPathIllusion,
			adapterrunner.EnforcementNetworkLoopbackOnly,
		},
	}
	selector := adapterrunner.NewLauncherSelector(host, factory)
	sink := &recordingSink{}
	cfg := &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected}}
	id := uuid.New()

	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{
		RunID: uuid.New(), ResolvedConfig: cfg, SandboxID: &id, EventSink: sink,
	})
	if picked != sandbox {
		t.Fatalf("picked %v, want sandbox", picked)
	}
	if sink.hasWarning("does not enforce") || sink.hasWarning("unrestricted network access") {
		t.Errorf("fully contained sandbox must not warn; events=%v", sink.events)
	}
}

// TestLauncherSelectorPick_LoopbackGapWarnsEvenWithFullBaseline pins the
// honesty warn for the localhost==full-network gap
// (knw-1784006975589682125): a sandbox whose backend provides every
// baseline protected enforcement but not network-loopback-only must emit
// the network-posture warn — the agent launches under the vrooli-aware
// (localhost) profile yet gets unrestricted network — without emitting the
// baseline gap warn.
func TestLauncherSelectorPick_LoopbackGapWarnsEvenWithFullBaseline(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	sandbox := mocks.NewFakeLauncher("sandbox")
	factory := mocks.NewFakeSandboxLauncherFactory(sandbox)
	factory.Containment = &adapterrunner.Containment{
		Level:   "required",
		Backend: "bwrap",
		Enforcements: []string{
			adapterrunner.EnforcementFilesystemWriteContainment,
			adapterrunner.EnforcementNetworkDeny,
			adapterrunner.EnforcementPIDNamespace,
			adapterrunner.EnforcementPathIllusion,
		},
	}
	selector := adapterrunner.NewLauncherSelector(host, factory)
	sink := &recordingSink{}
	cfg := &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected}}
	id := uuid.New()

	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{
		RunID: uuid.New(), ResolvedConfig: cfg, SandboxID: &id, EventSink: sink,
	})
	if picked != sandbox {
		t.Fatalf("loopback gap must not change selection; picked %v, want sandbox", picked)
	}
	if !sink.hasWarning(adapterrunner.EnforcementNetworkLoopbackOnly) {
		t.Errorf("expected loopback-only gap warn; events=%v", sink.events)
	}
	if !sink.hasWarning("unrestricted network access") {
		t.Errorf("loopback gap warn must state the real network posture; events=%v", sink.events)
	}
	if sink.hasWarning("protected mode requested but the sandbox does not enforce") {
		t.Errorf("baseline gap warn must not fire when baseline enforcements are present; events=%v", sink.events)
	}
}

func TestLauncherSelectorPick_NoConfigUsesHost(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	selector := adapterrunner.NewLauncherSelector(host, mocks.NewFakeSandboxLauncherFactory(mocks.NewFakeLauncher("sandbox")))
	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{}) // no config
	if picked != host {
		t.Errorf("no-config request picked %v; want host launcher", picked)
	}
}

// TestLauncherSelectorPickFor_ContinueRequestProtectedRoutesToSandbox
// asserts that ContinueRequest with the same routing primitives as a
// protected ExecuteRequest reaches the sandbox launcher via PickFor —
// proving that the durable-transcript and Continue paths share the seam.
func TestLauncherSelectorPickFor_ContinueRequestProtectedRoutesToSandbox(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	sandbox := mocks.NewFakeLauncher("sandbox")
	selector := adapterrunner.NewLauncherSelector(host, mocks.NewFakeSandboxLauncherFactory(sandbox))

	id := uuid.New()
	cont := adapterrunner.ContinueRequest{
		RunID:     uuid.New(),
		SandboxID: &id,
		ResolvedConfig: &domain.RunConfig{
			SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
		},
	}
	picked := selector.PickFor(context.Background(), cont.RunID, cont.GetConfig(), cont.SandboxID, cont.EventSink)
	if picked != sandbox {
		t.Errorf("protected ContinueRequest picked %v; want sandbox launcher", picked)
	}
}

// TestLauncherSelectorPickFor_ContinueRequestNoSandboxIDFallsBack asserts
// that a Continue routed for protected mode without a SandboxID falls
// back to host with a warn event — same contract as the Execute path.
func TestLauncherSelectorPickFor_ContinueRequestNoSandboxIDFallsBack(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	factory := mocks.NewFakeSandboxLauncherFactory(mocks.NewFakeLauncher("sandbox"))
	selector := adapterrunner.NewLauncherSelector(host, factory)
	sink := &recordingSink{}

	cont := adapterrunner.ContinueRequest{
		RunID:     uuid.New(),
		EventSink: sink,
		ResolvedConfig: &domain.RunConfig{
			SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
		},
	}
	picked := selector.PickFor(context.Background(), cont.RunID, cont.GetConfig(), cont.SandboxID, cont.EventSink)
	if picked != host {
		t.Errorf("nil-SandboxID Continue picked %v; want host launcher", picked)
	}
	if !sink.hasWarning("SandboxID is nil") {
		t.Errorf("no warning emitted; events=%v", sink.events)
	}
	if len(factory.CalledIDs()) != 0 {
		t.Error("factory should not have been consulted when SandboxID is nil")
	}
}

// TestLauncherSelectorPickFor_NilContinueConfigUsesHost asserts that a
// ContinueRequest with no ResolvedConfig (the runtime default) routes to
// host. Mirrors the Execute path's "no SandboxConfig → host" rule.
func TestLauncherSelectorPickFor_NilContinueConfigUsesHost(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	selector := adapterrunner.NewLauncherSelector(host, mocks.NewFakeSandboxLauncherFactory(mocks.NewFakeLauncher("sandbox")))

	cont := adapterrunner.ContinueRequest{RunID: uuid.New()}
	picked := selector.PickFor(context.Background(), cont.RunID, cont.GetConfig(), cont.SandboxID, cont.EventSink)
	if picked != host {
		t.Errorf("default-config Continue picked %v; want host launcher", picked)
	}
}

// TestLauncherSelectorSetSandboxLauncherFactory_SwapsFactoryAtRuntime
// asserts that SetSandboxLauncherFactory replaces the factory used by
// subsequent Pick calls — main.go uses this when the sandbox provider is
// constructed after the runner registry.
func TestLauncherSelectorSetSandboxLauncherFactory_SwapsFactoryAtRuntime(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	selector := adapterrunner.NewLauncherSelector(host, nil)

	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
	}
	id := uuid.New()
	req := adapterrunner.ExecuteRequest{ResolvedConfig: cfg, SandboxID: &id, RunID: uuid.New()}

	// First call with no factory → host fallback.
	if got := selector.Pick(context.Background(), req); got != host {
		t.Fatalf("pre-set factory; want host, got %v", got)
	}

	// Wire a factory at runtime.
	sandboxLauncher := mocks.NewFakeLauncher("sandbox")
	selector.SetSandboxLauncherFactory(mocks.NewFakeSandboxLauncherFactory(sandboxLauncher))

	// Subsequent call uses the new factory.
	if got := selector.Pick(context.Background(), req); got != sandboxLauncher {
		t.Fatalf("post-set factory; want sandbox launcher, got %v", got)
	}
}
