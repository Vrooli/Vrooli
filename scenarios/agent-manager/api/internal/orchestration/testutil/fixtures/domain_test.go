package fixtures

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestDomainFixtureOptionsOverrideMutableContractFields(t *testing.T) {
	profile := NewAgentProfile(t, WithAgentProfileName("reviewer"))
	if profile.Name != "reviewer" || profile.ProfileKey != "reviewer" {
		t.Fatalf("profile=%+v", profile)
	}
	task := NewTask(t, WithTaskTitle("title"), WithTaskDescription("description"), WithTaskStatus(domain.TaskStatusRunning))
	if task.Title != "title" || task.Description != "description" || task.Status != domain.TaskStatusRunning {
		t.Fatalf("task=%+v", task)
	}
	run := NewRun(t, task.ID, profile.ID, WithRunStatus(domain.RunStatusRunning))
	if run.Status != domain.RunStatusRunning {
		t.Fatalf("run=%+v", run)
	}
	if event := NewRunEvent(t, run.ID); event.RunID != run.ID {
		t.Fatalf("event=%+v", event)
	}
}

func TestDomainFixturesLinkRunToTaskAndProfile(t *testing.T) {
	profile := NewAgentProfile(t)
	task := NewTask(t)
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
		WithSandboxDeleteOn(domain.SandboxLifecycleTerminal),
	)

	if !cfg.ManualReview {
		t.Fatal("expected ManualReview=true")
	}
	if cfg.GetAutoApply() {
		t.Fatal("expected AutoApply=false")
	}
	if len(cfg.Lifecycle.DeleteOn) != 1 || cfg.Lifecycle.DeleteOn[0] != domain.SandboxLifecycleTerminal {
		t.Fatalf("DeleteOn = %v, want [terminal]", cfg.Lifecycle.DeleteOn)
	}
}
