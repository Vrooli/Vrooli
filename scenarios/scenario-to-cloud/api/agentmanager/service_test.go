package agentmanager

import (
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestDefaultProfileConfig(t *testing.T) {
	cfg := DefaultProfileConfig()

	if cfg.RoleRef != "code.smart" {
		t.Fatalf("expected default portable role code.smart, got %q", cfg.RoleRef)
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
	if !cfg.SkipPermissions {
		t.Fatalf("expected default permissions to skip prompts")
	}
	if cfg.SandboxMode != domainpb.SandboxMode_SANDBOX_MODE_OFF {
		t.Fatalf("expected SandboxMode=OFF for SSH-driven cloud investigations; got %v", cfg.SandboxMode)
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
		RoleRef:         "code.smart",
		MaxTurns:        10,
		TimeoutSeconds:  30,
		AllowedTools:    []string{"read_file"},
		SkipPermissions: true,
		SandboxMode:     domainpb.SandboxMode_SANDBOX_MODE_OFF,
	}

	profile := svc.buildProfile(cfg)
	if profile.Name != "Scenario To Cloud" {
		t.Fatalf("expected profile name to match, got %q", profile.Name)
	}
	if profile.ProfileKey != "scenario-to-cloud" {
		t.Fatalf("expected profile key to match, got %q", profile.ProfileKey)
	}
	if profile.RoleRef != "code.smart" {
		t.Fatalf("expected portable role to be preserved, got %q", profile.RoleRef)
	}
	if profile.Timeout.AsDuration() != 30*time.Second {
		t.Fatalf("expected timeout to be 30s, got %s", profile.Timeout.AsDuration())
	}
	if len(profile.AllowedTools) != 1 || profile.AllowedTools[0] != "read_file" {
		t.Fatalf("expected allowed tools to be preserved, got %+v", profile.AllowedTools)
	}
	if !profile.SkipPermissionPrompt {
		t.Fatalf("expected SkipPermissionPrompt to be preserved")
	}
	if profile.SandboxConfig == nil || profile.SandboxConfig.Mode != domainpb.SandboxMode_SANDBOX_MODE_OFF {
		t.Fatalf("expected SandboxConfig.Mode=OFF; got %+v", profile.SandboxConfig)
	}
}
