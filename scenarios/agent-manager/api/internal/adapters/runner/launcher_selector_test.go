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

func TestLauncherSelectorPick_ExplicitOffModeUsesHost(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	selector := adapterrunner.NewLauncherSelector(host, mocks.NewFakeSandboxLauncherFactory(mocks.NewFakeLauncher("sandbox")))
	cfg := &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeOff}}
	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{ResolvedConfig: cfg})
	if picked != host {
		t.Errorf("explicit off mode picked %v; want host launcher", picked)
	}
}

func TestLauncherSelector_IncompletePolicyCannotSelectHost(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  *domain.RunConfig
	}{
		{"missing-run-config", nil},
		{"missing-sandbox-config", &domain.RunConfig{}},
		{"unknown-mode", &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{Mode: "unknown"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := mocks.NewFakeLauncher("host")
			selector := adapterrunner.NewLauncherSelector(host, nil)
			picked := selector.PickFor(context.Background(), uuid.New(), tc.cfg, nil, nil)
			if _, err := picked.Launch(context.Background(), adapterrunner.LaunchRequest{Command: "agent"}); err == nil {
				t.Fatal("an incomplete or unknown policy must refuse launch, not imply permission for host execution")
			}
			if len(host.LaunchCalls()) != 0 {
				t.Fatal("an incomplete policy launched a host process")
			}
		})
	}
}

func TestLauncherSelector_WritePolicyRequiresAdvertisedEnforcement(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	sandbox := mocks.NewFakeLauncher("sandbox")
	factory := mocks.NewFakeSandboxLauncherFactory(sandbox)
	factory.Containment = protectedContainment()
	selector := adapterrunner.NewLauncherSelector(host, factory)
	cfg := domain.DefaultRunConfig()
	cfg.SandboxConfig.WritePolicy = &domain.WorkspaceWritePolicy{}
	id := uuid.New()
	request := adapterrunner.ExecuteRequest{ResolvedConfig: cfg, SandboxID: &id}
	if _, err := selector.Pick(context.Background(), request).Launch(context.Background(), adapterrunner.LaunchRequest{Command: "agent"}); err == nil {
		t.Fatal("an old provider may discard the policy; refuse it before launch")
	}
	factory.Containment.Enforcements = append(factory.Containment.Enforcements, adapterrunner.EnforcementWorkspaceWritePolicy)
	if _, err := selector.Pick(context.Background(), request).Launch(context.Background(), adapterrunner.LaunchRequest{Command: "agent"}); err == nil {
		t.Fatal("backend capability is not proof the sandbox persisted this run's policy")
	}
	factory.Containment.WritePolicy = &domain.WorkspaceWritePolicy{Paths: []string{"src"}}
	if _, err := selector.Pick(context.Background(), request).Launch(context.Background(), adapterrunner.LaunchRequest{Command: "agent"}); err == nil {
		t.Fatal("a wider retained sandbox policy must not satisfy an empty read-only grant")
	}
	factory.Containment.WritePolicy = &domain.WorkspaceWritePolicy{}
	if got := selector.Pick(context.Background(), request); got != sandbox {
		t.Fatalf("capable protected provider refused: %T", got)
	}
	for _, mode := range []domain.SandboxMode{domain.SandboxModeTracking, domain.SandboxModeOff} {
		cfg.SandboxConfig.Mode = mode
		if _, err := selector.Pick(context.Background(), request).Launch(context.Background(), adapterrunner.LaunchRequest{Command: "agent"}); err == nil {
			t.Fatal("write policy allowed an unprotected mode")
		}
	}
	if len(host.LaunchCalls())+len(sandbox.LaunchCalls()) != 0 {
		t.Fatal("refusal started a process")
	}
}

// TestLauncherSelectorPick_ProtectedWithFactoryAndIDPicksSandbox is the
// load-bearing case: protected mode + factory wired + SandboxID in request
// → SandboxLauncher.
func TestLauncherSelectorPick_ProtectedWithFactoryAndIDPicksSandbox(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	sandboxLauncher := mocks.NewFakeLauncher("sandbox")
	factory := mocks.NewFakeSandboxLauncherFactory(sandboxLauncher)
	factory.Containment = protectedContainment()
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

func TestLauncherSelectorPick_EffectGrantRefusesUnverifiableContainment(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	selector := adapterrunner.NewLauncherSelector(host, mocks.NewFakeSandboxLauncherFactory(mocks.NewFakeLauncher("sandbox")))
	id := uuid.New()
	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{
		RunID: uuid.New(), SandboxID: &id,
		ResolvedConfig: &domain.RunConfig{
			RequireEffectContainment: true,
			SandboxConfig:            &domain.SandboxConfig{Mode: domain.SandboxModeProtected},
		},
	})
	if picked == host {
		t.Fatal("effect-bearing run downgraded to host launcher")
	}
	if _, err := picked.Launch(context.Background(), adapterrunner.LaunchRequest{Command: "agent"}); err == nil || !strings.Contains(err.Error(), "containment") {
		t.Fatalf("launch error=%v, want containment refusal", err)
	}
}

func TestLauncherSelector_ProtectedAndEffectRunsNeverDowngrade(t *testing.T) {
	for _, mode := range []domain.SandboxMode{domain.SandboxModeProtected, domain.SandboxModeOff} {
		for _, defect := range []string{"missing-config", "missing-factory", "missing-id", "zero-id", "missing-launcher", "missing-report", "missing-enforcement", "uncontained-report"} {
			t.Run(string(mode)+"/"+defect, func(t *testing.T) {
				host := mocks.NewFakeLauncher("host")
				sandbox := mocks.NewFakeLauncher("sandbox")
				factory := mocks.NewFakeSandboxLauncherFactory(sandbox)
				factory.Containment = &adapterrunner.Containment{Level: "required", Backend: "bwrap", Enforcements: []string{
					adapterrunner.EnforcementFilesystemWriteContainment, adapterrunner.EnforcementNetworkDeny,
				}}
				cfg := &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{Mode: mode}, RequireEffectContainment: mode == domain.SandboxModeOff}
				id := uuid.New()
				sandboxID := &id
				switch defect {
				case "missing-config":
					cfg.SandboxConfig = nil
					cfg.RequireEffectContainment = true
				case "missing-factory":
					factory = nil
				case "missing-id":
					sandboxID = nil
				case "zero-id":
					id = uuid.Nil
				case "missing-launcher":
					factory = mocks.NewFakeSandboxLauncherFactory(nil)
				case "missing-report":
					factory.Containment = nil
				case "missing-enforcement":
					factory.Containment.Enforcements = []string{adapterrunner.EnforcementNetworkDeny}
				case "uncontained-report":
					factory.Containment.Backend = "none"
					factory.Containment.Level = "none"
				}
				var selectedFactory adapterrunner.SandboxLauncherFactory
				if factory != nil {
					selectedFactory = factory
				}
				selector := adapterrunner.NewLauncherSelector(host, selectedFactory)
				picked := selector.PickFor(context.Background(), uuid.New(), cfg, sandboxID, nil)
				if _, err := picked.Launch(context.Background(), adapterrunner.LaunchRequest{Command: "agent"}); err == nil {
					t.Fatal("required containment must refuse launch")
				}
				if len(host.LaunchCalls()) != 0 || len(sandbox.LaunchCalls()) != 0 {
					t.Fatal("refusal must not launch on host or unqualified sandbox")
				}
			})
		}
	}
}

func TestLauncherSelectorPick_TrackingNoFactoryFallsBackWithWarning(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	selector := adapterrunner.NewLauncherSelector(host, nil) // no factory
	sink := &recordingSink{}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeTracking},
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

func TestLauncherSelectorPick_TrackingNoSandboxIDFallsBackWithWarning(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	factory := mocks.NewFakeSandboxLauncherFactory(mocks.NewFakeLauncher("sandbox"))
	selector := adapterrunner.NewLauncherSelector(host, factory)
	sink := &recordingSink{}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeTracking},
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

func TestLauncherSelectorPick_TrackingFactoryReturnsNilFallsBackWithWarning(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	factory := mocks.NewFakeSandboxLauncherFactory(nil)
	selector := adapterrunner.NewLauncherSelector(host, factory)
	sink := &recordingSink{}
	cfg := &domain.RunConfig{
		SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeTracking},
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

// Tracking explicitly permits reduced containment, with a visible warning.
func TestLauncherSelectorPick_TrackingContainmentGapWarns(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	sandbox := mocks.NewFakeLauncher("sandbox")
	factory := mocks.NewFakeSandboxLauncherFactory(sandbox)
	// Uncontained sandbox: backend none, no enforcements.
	factory.Containment = &adapterrunner.Containment{
		Level: "none", Backend: "none", Enforcements: []string{},
	}
	selector := adapterrunner.NewLauncherSelector(host, factory)
	sink := &recordingSink{}
	cfg := &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeTracking}}
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

func TestLauncherSelectorPick_ExplicitNetworkDenyDoesNotClaimLoopbackGap(t *testing.T) {
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
	cfg := &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{
		Mode:        domain.SandboxModeProtected,
		NetworkMode: domain.NetworkAccessNone,
	}}
	id := uuid.New()

	if picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{
		RunID: uuid.New(), ResolvedConfig: cfg, SandboxID: &id, EventSink: sink,
	}); picked != sandbox {
		t.Fatalf("picked %v, want sandbox", picked)
	}
	if sink.hasWarning("unrestricted network access") || sink.hasWarning(adapterrunner.EnforcementNetworkLoopbackOnly) {
		t.Errorf("network-denied run emitted localhost warning: events=%v", sink.events)
	}
}

func TestLauncherSelectorPick_DefaultConfigRequiresSandbox(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	selector := adapterrunner.NewLauncherSelector(host, mocks.NewFakeSandboxLauncherFactory(mocks.NewFakeLauncher("sandbox")))
	picked := selector.Pick(context.Background(), adapterrunner.ExecuteRequest{}) // no config
	if _, err := picked.Launch(context.Background(), adapterrunner.LaunchRequest{}); err == nil {
		t.Fatal("default protected config must refuse without a sandbox")
	}
}

// TestLauncherSelectorPickFor_ContinueRequestProtectedRoutesToSandbox
// asserts that ContinueRequest with the same routing primitives as a
// protected ExecuteRequest reaches the sandbox launcher via PickFor —
// proving that the durable-transcript and Continue paths share the seam.
func TestLauncherSelectorPickFor_ContinueRequestProtectedRoutesToSandbox(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	sandbox := mocks.NewFakeLauncher("sandbox")
	factory := mocks.NewFakeSandboxLauncherFactory(sandbox)
	factory.Containment = protectedContainment()
	selector := adapterrunner.NewLauncherSelector(host, factory)

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

// Continuation must refuse missing sandbox identity just like execution.
func TestLauncherSelectorPickFor_ContinueRequestNoSandboxIDRefuses(t *testing.T) {
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
	if _, err := picked.Launch(context.Background(), adapterrunner.LaunchRequest{}); err == nil || !strings.Contains(err.Error(), "SandboxID") {
		t.Fatalf("nil-SandboxID Continue launch error = %v; want refusal", err)
	}
	if len(factory.CalledIDs()) != 0 {
		t.Error("factory should not have been consulted when SandboxID is nil")
	}
}

func TestLauncherSelectorPickFor_DefaultContinueConfigRequiresSandbox(t *testing.T) {
	host := mocks.NewFakeLauncher("host")
	selector := adapterrunner.NewLauncherSelector(host, mocks.NewFakeSandboxLauncherFactory(mocks.NewFakeLauncher("sandbox")))

	cont := adapterrunner.ContinueRequest{RunID: uuid.New()}
	picked := selector.PickFor(context.Background(), cont.RunID, cont.GetConfig(), cont.SandboxID, cont.EventSink)
	if _, err := picked.Launch(context.Background(), adapterrunner.LaunchRequest{}); err == nil {
		t.Fatal("default protected continuation must refuse without a sandbox")
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

	// Missing wiring must refuse without caching the failed selection.
	if _, err := selector.Pick(context.Background(), req).Launch(context.Background(), adapterrunner.LaunchRequest{}); err == nil {
		t.Fatal("missing factory must refuse launch")
	}

	// Wire a factory at runtime.
	sandboxLauncher := mocks.NewFakeLauncher("sandbox")
	factory := mocks.NewFakeSandboxLauncherFactory(sandboxLauncher)
	factory.Containment = protectedContainment()
	selector.SetSandboxLauncherFactory(factory)

	// Subsequent call uses the new factory.
	if got := selector.Pick(context.Background(), req); got != sandboxLauncher {
		t.Fatalf("post-set factory; want sandbox launcher, got %v", got)
	}
}

func protectedContainment() *adapterrunner.Containment {
	return &adapterrunner.Containment{Level: "required", Backend: "bwrap", Enforcements: []string{
		adapterrunner.EnforcementFilesystemWriteContainment, adapterrunner.EnforcementNetworkDeny,
	}}
}
