package agentmanager

import (
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestDefaultProfileConfig(t *testing.T) {
	cfg := DefaultProfileConfig()

	if cfg.RunnerType != domainpb.RunnerType_RUNNER_TYPE_CODEX {
		t.Fatalf("expected default runner type CODEX, got %v", cfg.RunnerType)
	}
	if cfg.ModelPreset != domainpb.ModelPreset_MODEL_PRESET_SMART {
		t.Fatalf("expected default model preset SMART, got %v", cfg.ModelPreset)
	}
	if cfg.MaxTurns != 75 {
		t.Fatalf("expected default max turns 75, got %d", cfg.MaxTurns)
	}
	if cfg.TimeoutSeconds != 600 {
		t.Fatalf("expected default timeout 600, got %d", cfg.TimeoutSeconds)
	}
	if len(cfg.AllowedTools) == 0 {
		t.Fatalf("expected default allowed tools to be populated")
	}
	if !cfg.SkipPermissions || cfg.RequiresSandbox || cfg.RequiresApproval {
		t.Fatalf("expected default permissions to skip prompts without sandbox/approval")
	}
}

func TestBuildProfile(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "Scenario To Cloud",
		ProfileKey:  "scenario-to-cloud",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})

	cfg := &ProfileConfig{
		RunnerType:       domainpb.RunnerType_RUNNER_TYPE_CODEX,
		Model:            "model-x",
		ModelPreset:      domainpb.ModelPreset_MODEL_PRESET_SMART,
		MaxTurns:         10,
		TimeoutSeconds:   30,
		AllowedTools:     []string{"read_file"},
		SkipPermissions:  true,
		RequiresSandbox:  false,
		RequiresApproval: true,
	}

	profile := svc.buildProfile(cfg)
	if profile.Name != "Scenario To Cloud" {
		t.Fatalf("expected profile name to match, got %q", profile.Name)
	}
	if profile.ProfileKey != "scenario-to-cloud" {
		t.Fatalf("expected profile key to match, got %q", profile.ProfileKey)
	}
	if profile.Timeout.AsDuration() != 30*time.Second {
		t.Fatalf("expected timeout to be 30s, got %s", profile.Timeout.AsDuration())
	}
	if len(profile.AllowedTools) != 1 || profile.AllowedTools[0] != "read_file" {
		t.Fatalf("expected allowed tools to be preserved, got %+v", profile.AllowedTools)
	}
	if !profile.SkipPermissionPrompt || profile.RequiresSandbox || !profile.RequiresApproval {
		t.Fatalf("expected permission flags to be preserved")
	}
}
