package orchestration

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestNormalizeSandboxConfig_AutoApproveDefaultsDeleteOn(t *testing.T) {
	cfg := &domain.SandboxConfig{
		Acceptance: domain.SandboxAcceptanceConfig{
			AutoApprove: true,
		},
	}

	result := normalizeSandboxConfig(cfg)

	if len(result.Lifecycle.DeleteOn) != 1 {
		t.Fatalf("expected 1 deleteOn event, got %d", len(result.Lifecycle.DeleteOn))
	}
	if result.Lifecycle.DeleteOn[0] != domain.SandboxLifecycleTerminal {
		t.Errorf("expected deleteOn[0]=%q, got %q", domain.SandboxLifecycleTerminal, result.Lifecycle.DeleteOn[0])
	}
}

func TestNormalizeSandboxConfig_AutoApproveRespectsExplicitDeleteOn(t *testing.T) {
	cfg := &domain.SandboxConfig{
		Acceptance: domain.SandboxAcceptanceConfig{
			AutoApprove: true,
		},
		Lifecycle: domain.SandboxLifecycleConfig{
			DeleteOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleApproved},
		},
	}

	result := normalizeSandboxConfig(cfg)

	if len(result.Lifecycle.DeleteOn) != 1 {
		t.Fatalf("expected 1 deleteOn event, got %d", len(result.Lifecycle.DeleteOn))
	}
	if result.Lifecycle.DeleteOn[0] != domain.SandboxLifecycleApproved {
		t.Errorf("expected explicit deleteOn preserved, got %q", result.Lifecycle.DeleteOn[0])
	}
}

func TestNormalizeSandboxConfig_AutoApproveRespectsExplicitStopOn(t *testing.T) {
	cfg := &domain.SandboxConfig{
		Acceptance: domain.SandboxAcceptanceConfig{
			AutoApprove: true,
		},
		Lifecycle: domain.SandboxLifecycleConfig{
			StopOn: []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal},
		},
	}

	result := normalizeSandboxConfig(cfg)

	// Should NOT add deleteOn because the caller explicitly configured stopOn
	if len(result.Lifecycle.DeleteOn) != 0 {
		t.Errorf("expected no deleteOn when stopOn is configured, got %v", result.Lifecycle.DeleteOn)
	}
	if len(result.Lifecycle.StopOn) != 1 {
		t.Fatalf("expected 1 stopOn event, got %d", len(result.Lifecycle.StopOn))
	}
}

func TestNormalizeSandboxConfig_NoAutoApproveNoDefaultLifecycle(t *testing.T) {
	cfg := &domain.SandboxConfig{
		Acceptance: domain.SandboxAcceptanceConfig{
			AutoApprove: false,
		},
	}

	result := normalizeSandboxConfig(cfg)

	if len(result.Lifecycle.DeleteOn) != 0 {
		t.Errorf("expected no deleteOn for manual approval, got %v", result.Lifecycle.DeleteOn)
	}
	if len(result.Lifecycle.StopOn) != 0 {
		t.Errorf("expected no stopOn for manual approval, got %v", result.Lifecycle.StopOn)
	}
}

func TestNormalizeSandboxConfig_Nil(t *testing.T) {
	result := normalizeSandboxConfig(nil)
	if result != nil {
		t.Errorf("expected nil result for nil input, got %v", result)
	}
}

// TestResolveSandboxConfig_AllInputsNil verifies the contract that
// resolveSandboxConfig never returns nil. Before 2026-04-24 it would cascade
// through a chain of nil inputs (default -> profile -> request) and return nil,
// which caused tryAutoApproval to silently short-circuit and leave runs
// stuck in NEEDS_REVIEW. A non-nil normalized zero-value is the fix.
func TestResolveSandboxConfig_AllInputsNil(t *testing.T) {
	o := &Orchestrator{}
	cfg, err := o.resolveSandboxConfig(CreateRunRequest{}, nil)
	if err != nil {
		t.Fatalf("resolveSandboxConfig returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("resolveSandboxConfig must never return nil — contract for auto-approval logic")
	}
	if cfg.Acceptance.Mode != "allowlist" {
		t.Errorf("expected normalized Acceptance.Mode=allowlist, got %q", cfg.Acceptance.Mode)
	}
	if cfg.Acceptance.DisableAutoApproveIfEmpty {
		t.Error("expected DisableAutoApproveIfEmpty=false by default so empty sandboxes auto-approve")
	}
}

// TestResolveSandboxConfig_ProfileConfigUsed verifies that a profile-provided
// SandboxConfig is honored (precedence over the zero-value default).
func TestResolveSandboxConfig_ProfileConfigUsed(t *testing.T) {
	o := &Orchestrator{}
	profile := &domain.AgentProfile{
		SandboxConfig: &domain.SandboxConfig{
			Acceptance: domain.SandboxAcceptanceConfig{
				AutoApprove: true,
			},
		},
	}
	cfg, err := o.resolveSandboxConfig(CreateRunRequest{}, profile)
	if err != nil {
		t.Fatalf("resolveSandboxConfig returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("cfg must not be nil")
	}
	if !cfg.Acceptance.AutoApprove {
		t.Error("expected AutoApprove=true from profile")
	}
}

// TestResolveSandboxConfig_RequestOverridesProfile verifies inline request
// config wins over profile config (documented precedence).
func TestResolveSandboxConfig_RequestOverridesProfile(t *testing.T) {
	o := &Orchestrator{}
	profile := &domain.AgentProfile{
		SandboxConfig: &domain.SandboxConfig{
			Acceptance: domain.SandboxAcceptanceConfig{AutoApprove: true},
		},
	}
	req := CreateRunRequest{
		SandboxConfig: &domain.SandboxConfig{
			Acceptance: domain.SandboxAcceptanceConfig{AutoReject: true},
		},
	}
	cfg, err := o.resolveSandboxConfig(req, profile)
	if err != nil {
		t.Fatalf("resolveSandboxConfig returned error: %v", err)
	}
	if cfg.Acceptance.AutoApprove {
		t.Error("request config should have overridden profile's AutoApprove")
	}
	if !cfg.Acceptance.AutoReject {
		t.Error("expected AutoReject=true from request")
	}
}
