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
	if cfg.MaxTurns != 60 {
		t.Fatalf("expected default max turns 60, got %d", cfg.MaxTurns)
	}
	if cfg.TimeoutSeconds != 900 {
		t.Fatalf("expected default timeout 900, got %d", cfg.TimeoutSeconds)
	}
	if len(cfg.AllowedTools) == 0 {
		t.Fatalf("expected default allowed tools to be populated")
	}
	if cfg.SkipPermissions || cfg.RequiresSandbox || !cfg.RequiresApproval {
		t.Fatalf("expected default permissions to require approval without sandbox or skip prompts")
	}
}

func TestBuildProfile(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "Swarm Manager",
		ProfileKey:  "swarm-manager",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})

	cfg := &ProfileConfig{
		RunnerType:       domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		Model:            "model-x",
		ModelPreset:      domainpb.ModelPreset_MODEL_PRESET_SMART,
		MaxTurns:         10,
		TimeoutSeconds:   30,
		AllowedTools:     []string{"Read"},
		SkipPermissions:  false,
		RequiresSandbox:  false,
		RequiresApproval: true,
	}

	profile := svc.buildProfile(cfg)
	if profile.Name != "Swarm Manager" {
		t.Fatalf("expected profile name to match, got %q", profile.Name)
	}
	if profile.ProfileKey != "swarm-manager" {
		t.Fatalf("expected profile key to match, got %q", profile.ProfileKey)
	}
	if profile.Timeout.AsDuration() != 30*time.Second {
		t.Fatalf("expected timeout to be 30s, got %s", profile.Timeout.AsDuration())
	}
	if len(profile.AllowedTools) != 1 || profile.AllowedTools[0] != "Read" {
		t.Fatalf("expected allowed tools to be preserved, got %+v", profile.AllowedTools)
	}
	if profile.SkipPermissionPrompt || profile.RequiresSandbox || !profile.RequiresApproval {
		t.Fatalf("expected permission flags to be preserved")
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
