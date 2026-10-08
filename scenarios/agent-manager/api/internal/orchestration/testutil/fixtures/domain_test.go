package fixtures

import (
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestDomainFixtureOptionsOverrideEveryMutableContractField(t *testing.T) {
	profileID, taskID, runID, sandboxID, eventID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	profile := NewAgentProfile(t, WithAgentProfileID(profileID), WithAgentProfileName("reviewer"), WithAgentProfileRole("analysis"), WithAgentProfileSandboxConfig(&domain.SandboxConfig{}))
	if profile.ID != profileID || profile.Name != "reviewer" || profile.ProfileKey != "reviewer" || profile.RoleRef != "analysis" || profile.SandboxConfig == nil {
		t.Fatalf("profile=%+v", profile)
	}
	task := NewTask(t, WithTaskID(taskID), WithTaskTitle("title"), WithTaskDescription("description"), WithTaskScope("pkg", "/workspace"), WithTaskStatus(domain.TaskStatusRunning))
	if task.ID != taskID || task.Title != "title" || task.Description != "description" || task.ScopePath != "pkg" || task.ProjectRoot != "/workspace" || task.Status != domain.TaskStatusRunning {
		t.Fatalf("task=%+v", task)
	}
	run := NewRun(t, taskID, profileID, WithRunID(runID), WithRunStatus(domain.RunStatusRunning), WithRunPhase(domain.RunPhaseExecuting), WithRunMode(domain.RunModeInPlace), WithRunSandboxConfig(&domain.SandboxConfig{}), WithRunSandboxID(sandboxID), WithRunConversationID("conversation"))
	if run.ID != runID || run.Status != domain.RunStatusRunning || run.Phase != domain.RunPhaseExecuting || run.RunMode != domain.RunModeInPlace || run.SandboxID == nil || *run.SandboxID != sandboxID || run.ConversationID != "conversation" {
		t.Fatalf("run=%+v", run)
	}
	when := time.Now().UTC().Round(0)
	event := NewRunEvent(t, runID, WithRunEventID(eventID), WithRunEventSequence(7), WithRunEventTimestamp(when), WithRunEventPayload(domain.EventTypeStatus, &domain.StatusEventData{NewStatus: string(domain.RunStatusRunning)}))
	if event.ID != eventID || event.Sequence != 7 || !event.Timestamp.Equal(when) || event.EventType != domain.EventTypeStatus {
		t.Fatalf("event=%+v", event)
	}
}

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
