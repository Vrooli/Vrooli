package agentmanager

import (
	"testing"
	"time"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestDefaultProfileConfigHasExpectedSafetyDefaults(t *testing.T) {
	cfg := DefaultProfileConfig()

	if cfg.RunnerType != domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE {
		t.Fatalf("expected Claude Code runner, got %v", cfg.RunnerType)
	}
	if cfg.ModelPreset != domainpb.ModelPreset_MODEL_PRESET_SMART {
		t.Fatalf("expected smart model preset, got %v", cfg.ModelPreset)
	}
	if cfg.TimeoutSeconds != 900 {
		t.Fatalf("expected 900 second timeout, got %d", cfg.TimeoutSeconds)
	}
	if cfg.SkipPermissions {
		t.Fatal("expected permission prompts to remain enabled by default")
	}
	if cfg.SandboxMode != domainpb.SandboxMode_SANDBOX_MODE_PROTECTED {
		t.Fatalf("expected SandboxMode=PROTECTED so test-genie runs always reach the workspace-sandbox merged dir; got %v", cfg.SandboxMode)
	}
}

func TestBuildProfileAndDefaultProfileRef(t *testing.T) {
	svc := NewAgentService(Config{
		ProfileName: "Test Genie Agent",
		ProfileKey:  "test-genie",
		Timeout:     5 * time.Second,
		Enabled:     true,
	})

	profile := svc.buildProfile(&ProfileConfig{
		RunnerType:      domainpb.RunnerType_RUNNER_TYPE_CLAUDE_CODE,
		Model:           "gpt-5.4",
		ModelPreset:     domainpb.ModelPreset_MODEL_PRESET_SMART,
		MaxTurns:        12,
		TimeoutSeconds:  45,
		AllowedTools:    []string{"Read", "Write"},
		SkipPermissions: true,
		SandboxMode:     domainpb.SandboxMode_SANDBOX_MODE_PROTECTED,
	})

	if profile.Name != "Test Genie Agent" {
		t.Fatalf("expected profile name to be propagated, got %q", profile.Name)
	}
	if profile.ProfileKey != "test-genie" {
		t.Fatalf("expected profile key to be propagated, got %q", profile.ProfileKey)
	}
	if profile.Timeout.AsDuration() != 45*time.Second {
		t.Fatalf("expected timeout to be converted to duration, got %s", profile.Timeout.AsDuration())
	}
	if !profile.SkipPermissionPrompt {
		t.Fatal("expected SkipPermissionPrompt to mirror the config")
	}
	if profile.SandboxConfig == nil || profile.SandboxConfig.Mode != domainpb.SandboxMode_SANDBOX_MODE_PROTECTED {
		t.Fatalf("expected SandboxConfig.Mode=PROTECTED to mirror cfg.SandboxMode, got %+v", profile.SandboxConfig)
	}

	ref := svc.defaultProfileRef()
	if ref.ProfileKey != "test-genie" {
		t.Fatalf("expected default profile ref to use profile key, got %q", ref.ProfileKey)
	}
	if ref.Defaults == nil || ref.Defaults.Name != "Test Genie Agent" {
		t.Fatal("expected default profile ref to include generated defaults")
	}
}

func TestMapRunStatusRoundTrip(t *testing.T) {
	cases := map[domainpb.RunStatus]string{
		domainpb.RunStatus_RUN_STATUS_PENDING:      "pending",
		domainpb.RunStatus_RUN_STATUS_STARTING:     "pending",
		domainpb.RunStatus_RUN_STATUS_RUNNING:      "running",
		domainpb.RunStatus_RUN_STATUS_NEEDS_REVIEW: "running",
		domainpb.RunStatus_RUN_STATUS_COMPLETE:     "completed",
		domainpb.RunStatus_RUN_STATUS_FAILED:       "failed",
		domainpb.RunStatus_RUN_STATUS_CANCELLED:    "stopped",
		domainpb.RunStatus_RUN_STATUS_UNSPECIFIED:  "unknown",
	}

	for input, expected := range cases {
		if got := MapRunStatus(input); got != expected {
			t.Fatalf("MapRunStatus(%v) = %q, want %q", input, got, expected)
		}
	}

	if got := MapStatusToRun("completed"); got != domainpb.RunStatus_RUN_STATUS_COMPLETE {
		t.Fatalf("expected completed to map back to RUN_STATUS_COMPLETE, got %v", got)
	}
	if got := MapStatusToRun("bogus"); got != domainpb.RunStatus_RUN_STATUS_UNSPECIFIED {
		t.Fatalf("expected unknown status to map to unspecified, got %v", got)
	}
}
