// Tests for resolveSandboxConfig and normalizeSandboxConfig under the
// auditability contract. Contract levers: SandboxConfig.ManualReview /
// AutoApply / ApplyOnFailure.

package orchestration

import (
	"testing"

	"agent-manager/internal/domain"
)

// TestNormalizeSandboxConfig_AutoApplyDefaultsCheckpointOn pins the
// resumable-turn behavior: AutoApply=true (the contract default) without an
// explicit lifecycle config checkpoints each turn and reserves deletion for
// explicit run finalization.
func TestNormalizeSandboxConfig_AutoApplyDefaultsCheckpointOn(t *testing.T) {
	cfg := domain.DefaultSandboxConfig() // AutoApply=true, ManualReview=false

	result := normalizeSandboxConfig(cfg)

	if len(result.Lifecycle.CheckpointOn) != 3 {
		t.Fatalf("expected 3 checkpointOn events, got %d", len(result.Lifecycle.CheckpointOn))
	}
	if result.Lifecycle.CheckpointOn[0] != domain.SandboxLifecycleTurnCompleted {
		t.Errorf("expected checkpointOn[0]=%q, got %q", domain.SandboxLifecycleTurnCompleted, result.Lifecycle.CheckpointOn[0])
	}
	if len(result.Lifecycle.DeleteOn) != 1 {
		t.Fatalf("expected 1 deleteOn event, got %d", len(result.Lifecycle.DeleteOn))
	}
	if result.Lifecycle.DeleteOn[0] != domain.SandboxLifecycleRunFinalized {
		t.Errorf("expected deleteOn[0]=%q, got %q", domain.SandboxLifecycleRunFinalized, result.Lifecycle.DeleteOn[0])
	}
}

// TestNormalizeSandboxConfig_AutoApplyRespectsExplicitDeleteOn verifies
// caller-provided DeleteOn is preserved.
func TestNormalizeSandboxConfig_AutoApplyRespectsExplicitDeleteOn(t *testing.T) {
	cfg := domain.DefaultSandboxConfig()
	cfg.Lifecycle.DeleteOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleApproved}

	result := normalizeSandboxConfig(cfg)

	if len(result.Lifecycle.DeleteOn) != 1 {
		t.Fatalf("expected 1 deleteOn event, got %d", len(result.Lifecycle.DeleteOn))
	}
	if result.Lifecycle.DeleteOn[0] != domain.SandboxLifecycleApproved {
		t.Errorf("expected explicit deleteOn preserved, got %q", result.Lifecycle.DeleteOn[0])
	}
}

// TestNormalizeSandboxConfig_AutoApplyRespectsExplicitStopOn ensures we do
// not stomp on caller-provided StopOn either.
func TestNormalizeSandboxConfig_AutoApplyRespectsExplicitStopOn(t *testing.T) {
	cfg := domain.DefaultSandboxConfig()
	cfg.Lifecycle.StopOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal}

	result := normalizeSandboxConfig(cfg)

	if len(result.Lifecycle.DeleteOn) != 0 {
		t.Errorf("expected no deleteOn when stopOn is configured, got %v", result.Lifecycle.DeleteOn)
	}
	if len(result.Lifecycle.StopOn) != 1 {
		t.Fatalf("expected 1 stopOn event, got %d", len(result.Lifecycle.StopOn))
	}
}

// TestNormalizeSandboxConfig_ManualReviewSkipsDefaultLifecycle verifies the
// manual-review-deferral case: when ManualReview=true the sandbox must
// persist past run end, so we MUST NOT inject a default DeleteOn=[terminal].
// The TTL GC in workspace-sandbox LifecycleReconciler (Phase 4) is the
// authoritative cleanup path for these sandboxes.
func TestNormalizeSandboxConfig_ManualReviewSkipsDefaultLifecycle(t *testing.T) {
	cfg := domain.DefaultSandboxConfig()
	cfg.ManualReview = true

	result := normalizeSandboxConfig(cfg)

	if len(result.Lifecycle.DeleteOn) != 0 {
		t.Errorf("expected no DeleteOn for manualReview=true (sandbox must persist), got %v", result.Lifecycle.DeleteOn)
	}
}

// TestNormalizeSandboxConfig_AutoApplyOffSkipsDefaultLifecycle covers the
// operator opt-out: AutoApply=false means no run-end apply, so the default
// post-terminal cleanup also doesn't apply.
func TestNormalizeSandboxConfig_AutoApplyOffSkipsDefaultLifecycle(t *testing.T) {
	cfg := domain.DefaultSandboxConfig()
	off := false
	cfg.AutoApply = &off

	result := normalizeSandboxConfig(cfg)

	if len(result.Lifecycle.DeleteOn) != 0 {
		t.Errorf("expected no DeleteOn when AutoApply=false, got %v", result.Lifecycle.DeleteOn)
	}
}

func TestNormalizeSandboxConfig_Nil(t *testing.T) {
	if result := normalizeSandboxConfig(nil); result != nil {
		t.Errorf("expected nil result for nil input, got %v", result)
	}
}

// TestResolveSandboxConfig_AllInputsNil pins the never-nil contract:
// resolveSandboxConfig must return a non-nil config so applyAtRunEnd has
// something to consult. Before 2026-04-24 a chain of nil inputs cascaded
// to a nil return, which silently short-circuited apply behavior.
func TestResolveSandboxConfig_AllInputsNil(t *testing.T) {
	o := &Orchestrator{}
	cfg, err := o.resolveSandboxConfig(CreateRunRequest{}, nil)
	if err != nil {
		t.Fatalf("resolveSandboxConfig returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("resolveSandboxConfig must never return nil — contract for applyAtRunEnd")
	}
	if cfg.Acceptance.Mode != "allowlist" {
		t.Errorf("expected normalized Acceptance.Mode=allowlist, got %q", cfg.Acceptance.Mode)
	}
}

// TestResolveSandboxConfig_ProfileConfigUsed pins precedence: profile
// config wins over the zero-value default.
func TestResolveSandboxConfig_ProfileConfigUsed(t *testing.T) {
	o := &Orchestrator{}
	profileCfg := domain.DefaultSandboxConfig()
	profileCfg.ManualReview = true
	profile := &domain.AgentProfile{SandboxConfig: profileCfg}

	cfg, err := o.resolveSandboxConfig(CreateRunRequest{}, profile)
	if err != nil {
		t.Fatalf("resolveSandboxConfig returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg must not be nil")
	}
	if !cfg.ManualReview {
		t.Error("expected ManualReview=true from profile")
	}
}

// TestResolveSandboxConfig_RequestOverridesProfile pins the inline-wins
// precedence rule.
func TestResolveSandboxConfig_RequestOverridesProfile(t *testing.T) {
	o := &Orchestrator{}
	profileCfg := domain.DefaultSandboxConfig()
	profileCfg.ManualReview = true
	profile := &domain.AgentProfile{SandboxConfig: profileCfg}

	reqCfg := domain.DefaultSandboxConfig()
	reqCfg.ManualReview = false
	req := CreateRunRequest{SandboxConfig: reqCfg}

	cfg, err := o.resolveSandboxConfig(req, profile)
	if err != nil {
		t.Fatalf("resolveSandboxConfig returned error: %v", err)
	}
	if cfg.ManualReview {
		t.Error("request config should have overridden profile's ManualReview=true")
	}
}
