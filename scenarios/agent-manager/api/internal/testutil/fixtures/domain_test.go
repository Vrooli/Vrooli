package fixtures

import (
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestDomainFixturesLinkRunToTaskAndProfile(t *testing.T) {
	profileID := uuid.New()
	taskID := uuid.New()

	profile := NewAgentProfile(t, WithAgentProfileID(profileID))
	task := NewTask(t, WithTaskID(taskID))
	run := NewRun(t, task.ID, profile.ID)

	if run.TaskID != task.ID {
		t.Fatalf("run TaskID = %s, want %s", run.TaskID, task.ID)
	}
	if run.AgentProfileID == nil || *run.AgentProfileID != profile.ID {
		t.Fatalf("run AgentProfileID = %v, want %s", run.AgentProfileID, profile.ID)
	}
	if run.Status != domain.RunStatusPending {
		t.Fatalf("run Status = %s, want %s", run.Status, domain.RunStatusPending)
	}
	if run.Phase != domain.RunPhaseQueued {
		t.Fatalf("run Phase = %s, want %s", run.Phase, domain.RunPhaseQueued)
	}
}

func TestSandboxConfigFixtureAppliesOptions(t *testing.T) {
	cfg := NewSandboxConfig(t,
		WithSandboxManualReview(true),
		WithSandboxAutoApply(false),
		WithSandboxApplyOnFailure(false),
		WithSandboxDeleteOn(domain.SandboxLifecycleTerminal),
	)

	if !cfg.ManualReview {
		t.Fatal("expected ManualReview=true")
	}
	if cfg.GetAutoApply() {
		t.Fatal("expected AutoApply=false")
	}
	if cfg.GetApplyOnFailure() {
		t.Fatal("expected ApplyOnFailure=false")
	}
	if len(cfg.Lifecycle.DeleteOn) != 1 || cfg.Lifecycle.DeleteOn[0] != domain.SandboxLifecycleTerminal {
		t.Fatalf("DeleteOn = %v, want [terminal]", cfg.Lifecycle.DeleteOn)
	}
}
