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
