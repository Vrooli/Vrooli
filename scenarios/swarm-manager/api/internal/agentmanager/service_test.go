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
	if !cfg.RequiresSandbox {
		t.Fatal("expected RequiresSandbox=true: swarm-manager agents run sandboxed so agent-manager's auto-approve-if-empty path handles read-only runs cleanly")
	}
	if !cfg.RequiresApproval {
		t.Fatal("expected RequiresApproval=true: sandboxed runs with diffs are human-reviewed; empty sandboxes auto-approve")
	}
}

func TestDefaultProfileRef_UpdateExistingTrue(t *testing.T) {
	svc := NewAgentService(AgentServiceConfig{
		ProfileName: "Swarm Manager",
		ProfileKey:  "swarm-manager",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})
	ref := svc.defaultProfileRef()
	if ref == nil {
		t.Fatal("defaultProfileRef returned nil for configured service")
	}
	if !ref.UpdateExisting {
		t.Fatal("expected UpdateExisting=true so code-declared defaults are authoritative over stale DB state")
	}
	if ref.Defaults == nil {
		t.Fatal("expected Defaults to be populated")
	}
	if !ref.Defaults.RequiresSandbox {
		t.Fatal("expected Defaults.RequiresSandbox=true to match DefaultProfileConfig")
	}
	if !ref.Defaults.RequiresApproval {
		t.Fatal("expected Defaults.RequiresApproval=true to match DefaultProfileConfig")
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
