package agentmanager

import (
	"strings"
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestDefaultProfileConfig(t *testing.T) {
	cfg := DefaultProfileConfig()

	if cfg.RunnerType != domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE {
		t.Fatalf("expected default runner type CLAUDE_CODE, got %v", cfg.RunnerType)
	}
	if cfg.ModelPreset != domainpb.ModelPreset_MODEL_PRESET_SMART {
		t.Fatalf("expected default model preset SMART, got %v", cfg.ModelPreset)
	}
	if cfg.MaxTurns != DefaultAgentMaxTurns {
		t.Fatalf("expected default max turns %d, got %d", DefaultAgentMaxTurns, cfg.MaxTurns)
	}
	if cfg.TimeoutSeconds != 3600 {
		t.Fatalf("expected default timeout 3600, got %d", cfg.TimeoutSeconds)
	}
	if len(cfg.AllowedTools) == 0 {
		t.Fatalf("expected default allowed tools to be populated")
	}
	if cfg.SkipPermissions {
		t.Fatal("expected SkipPermissions=false by default")
	}
	if cfg.SandboxMode != domainpb.SandboxMode_SANDBOX_MODE_PROTECTED {
		t.Fatalf("expected SandboxMode=PROTECTED so swarm-manager runs always carry the auditability sandbox; got %v", cfg.SandboxMode)
	}
	// swarm-manager auto-accepts; the ManualReview field was removed
	// from ProfileConfig 2026-04-28. There is no per-skill override
	// path — sandbox is the auditability layer, not a review queue.
}

func TestDefaultProfileRef_UsesManifestProfileOnly(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "Swarm Manager",
		ProfileKey:  "swarm-manager/default",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})
	ref := svc.defaultProfileRef()
	if ref == nil {
		t.Fatal("defaultProfileRef returned nil for configured service")
	}
	if ref.ProfileKey != "swarm-manager/default" {
		t.Fatalf("expected manifest profile key, got %q", ref.ProfileKey)
	}
	if ref.UpdateExisting || ref.Defaults != nil {
		t.Fatalf("expected run creation to reference the reconciled manifest profile without inline defaults")
	}
}

func TestProfileRefFor_UsesExplicitProfileKey(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "Swarm Manager",
		ProfileKey:  "swarm-manager/default",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})
	ref, err := svc.profileRefFor("swarm-manager/deep-work")
	if err != nil {
		t.Fatalf("profileRefFor returned error: %v", err)
	}
	if ref == nil || ref.ProfileKey != "swarm-manager/deep-work" {
		t.Fatalf("profileRefFor explicit key = %+v", ref)
	}
}

func TestProfileRefFor_FailsWhenReconciledProfilesMissingExplicitKey(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "Swarm Manager",
		ProfileKey:  "swarm-manager/default",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})
	svc.profileIDs = map[string]string{"swarm-manager/default": "profile-default"}

	_, err := svc.profileRefFor("swarm-manager/deep-work")
	if err == nil {
		t.Fatal("expected missing reconciled profile error")
	}
}

func TestValidateRequiredProfilesAcceptsAllRequiredProfiles(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName:  "Swarm Manager",
		ProfileKey:   "swarm-manager/default",
		RequiredKeys: []string{"swarm-manager/deep-work", "swarm-manager/analysis"},
		Timeout:      5 * time.Second,
		Enabled:      true,
	})

	err := svc.validateRequiredProfiles(map[string]string{
		"swarm-manager/default":   "profile-default",
		"swarm-manager/deep-work": "profile-deep-work",
		"swarm-manager/analysis":  "profile-analysis",
	})
	if err != nil {
		t.Fatalf("validateRequiredProfiles returned error: %v", err)
	}
}

func TestValidateRequiredProfilesRejectsMissingDeepWork(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName:  "Swarm Manager",
		ProfileKey:   "swarm-manager/default",
		RequiredKeys: []string{"swarm-manager/deep-work", "swarm-manager/analysis"},
		Timeout:      5 * time.Second,
		Enabled:      true,
	})

	err := svc.validateRequiredProfiles(map[string]string{
		"swarm-manager/default":  "profile-default",
		"swarm-manager/analysis": "profile-analysis",
	})
	if err == nil || !strings.Contains(err.Error(), "swarm-manager/deep-work") {
		t.Fatalf("expected missing deep-work profile error, got %v", err)
	}
}

func TestValidateRequiredProfilesRejectsMissingAnalysis(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName:  "Swarm Manager",
		ProfileKey:   "swarm-manager/default",
		RequiredKeys: []string{"swarm-manager/deep-work", "swarm-manager/analysis"},
		Timeout:      5 * time.Second,
		Enabled:      true,
	})

	err := svc.validateRequiredProfiles(map[string]string{
		"swarm-manager/default":   "profile-default",
		"swarm-manager/deep-work": "profile-deep-work",
	})
	if err == nil || !strings.Contains(err.Error(), "swarm-manager/analysis") {
		t.Fatalf("expected missing analysis profile error, got %v", err)
	}
}

func TestValidateRequiredProfilesRejectsNonOwnedProfileKey(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName:  "Swarm Manager",
		ProfileKey:   "swarm-manager/default",
		RequiredKeys: []string{"other-scenario/analysis"},
		Timeout:      5 * time.Second,
		Enabled:      true,
	})

	err := svc.validateRequiredProfiles(map[string]string{
		"swarm-manager/default":   "profile-default",
		"other-scenario/analysis": "profile-analysis",
	})
	if err == nil || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("expected non-owned profile error, got %v", err)
	}
}

func TestBuildProfile(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "Swarm Manager",
		ProfileKey:  "swarm-manager/default",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})

	cfg := &ProfileConfig{
		RunnerType:      domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		Model:           "model-x",
		ModelPreset:     domainpb.ModelPreset_MODEL_PRESET_SMART,
		MaxTurns:        10,
		TimeoutSeconds:  30,
		AllowedTools:    []string{"Read"},
		SkipPermissions: false,
		SandboxMode:     domainpb.SandboxMode_SANDBOX_MODE_TRACKING,
	}

	profile := svc.buildProfile(cfg)
	if profile.Name != "Swarm Manager" {
		t.Fatalf("expected profile name to match, got %q", profile.Name)
	}
	if profile.ProfileKey != "swarm-manager/default" {
		t.Fatalf("expected profile key to match, got %q", profile.ProfileKey)
	}
	if profile.Timeout.AsDuration() != 30*time.Second {
		t.Fatalf("expected timeout to be 30s, got %s", profile.Timeout.AsDuration())
	}
	if len(profile.AllowedTools) != 1 || profile.AllowedTools[0] != "Read" {
		t.Fatalf("expected allowed tools to be preserved, got %+v", profile.AllowedTools)
	}
	if profile.SkipPermissionPrompt {
		t.Fatalf("expected SkipPermissionPrompt to be preserved as false")
	}
	// Phase-1 contract: buildProfile carries the explicit Mode through
	// agent-manager fills in the remaining defaults at
	// resolveSandboxConfig (AutoApply, ApplyOnFailure, etc).
	if profile.SandboxConfig == nil {
		t.Fatal("expected SandboxConfig to carry the explicit Mode")
	}
	if profile.SandboxConfig.Mode != domainpb.SandboxMode_SANDBOX_MODE_TRACKING {
		t.Fatalf("expected SandboxConfig.Mode=TRACKING, got %v", profile.SandboxConfig.Mode)
	}
}

func TestTruncateDescription_Short(t *testing.T) {
	desc := "short description"
	result := truncateDescription(desc)
	if result != desc {
		t.Fatalf("expected unchanged description, got %q", result)
	}
}

func TestTruncateDescription_ExactLimit(t *testing.T) {
	desc := strings.Repeat("a", maxTaskDescriptionLen)
	result := truncateDescription(desc)
	if result != desc {
		t.Fatalf("expected unchanged description at exact limit, got len=%d", len(result))
	}
}

func TestTruncateDescription_OverLimit(t *testing.T) {
	desc := strings.Repeat("x", maxTaskDescriptionLen+1000)
	result := truncateDescription(desc)
	if len(result) > maxTaskDescriptionLen {
		t.Fatalf("truncated description exceeds limit: len=%d, max=%d", len(result), maxTaskDescriptionLen)
	}
	if !strings.HasSuffix(result, "[truncated — full prompt provided via run request]") {
		t.Fatal("expected truncation suffix")
	}
}

func TestTruncateDescription_LargePrompt(t *testing.T) {
	// A 20KB prompt fits within the 64KB limit.
	desc := strings.Repeat("y", 20195)
	result := truncateDescription(desc)
	if result != desc {
		t.Fatalf("20KB prompt should pass through unchanged, got len=%d", len(result))
	}
}

func TestTruncateDescription_ExceedsNewLimit(t *testing.T) {
	// Verify truncation still works for prompts exceeding the 64KB limit.
	desc := strings.Repeat("z", maxTaskDescriptionLen+500)
	result := truncateDescription(desc)
	if len(result) > maxTaskDescriptionLen {
		t.Fatalf("prompt exceeding 64KB not truncated: len=%d", len(result))
	}
	if !strings.HasSuffix(result, "[truncated — full prompt provided via run request]") {
		t.Fatal("expected truncation suffix")
	}
}
