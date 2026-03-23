package execution

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/promptmanager"
)

type stubAgentService struct {
	spawnCalls int
	spawnErr   error
}

func (s *stubAgentService) IsEnabled() bool { return true }

func (s *stubAgentService) SpawnBacklog(_ context.Context, _ agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error) {
	s.spawnCalls++
	if s.spawnErr != nil {
		return agentmanager.RunResult{}, s.spawnErr
	}
	return agentmanager.RunResult{TaskID: "task-1", RunID: "run-1"}, nil
}

func TestQueueAndStartManualExecution(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "test-idea", map[string]any{
		"name":        "test-idea",
		"title":       "Test",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWritePlanFile(t, root, "idea", "test-idea")

	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		RootDir:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "test-idea",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", record.Status)
	}

	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "test-idea", "spec.json"))
	if storedItem["status"] != "queued" {
		t.Fatalf("expected backlog status queued, got %#v", storedItem["status"])
	}

	started, err := service.Start(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if started.Status != StatusStarting {
		t.Fatalf("expected starting status, got %s", started.Status)
	}
	if started.TaskID != "task-1" || started.RunID != "run-1" {
		t.Fatalf("expected task/run IDs set, got task=%s run=%s", started.TaskID, started.RunID)
	}
	if started.PromptTrace == nil {
		t.Fatal("expected prompt trace to be captured")
	}
	if started.PromptTrace.SkillID != "swarm-manager-process-idea" {
		t.Fatalf("expected process idea prompt skill ID, got %q", started.PromptTrace.SkillID)
	}
	if agent.spawnCalls != 1 {
		t.Fatalf("expected 1 spawn call, got %d", agent.spawnCalls)
	}
}

func TestQueueBacklog_UsesPolicyDefaultsWhenModeMissing(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "policy-idea", map[string]any{
		"name":        "policy-idea",
		"title":       "Policy Idea",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWritePlanFile(t, root, "idea", "policy-idea")

	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		RootDir:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		PolicyPath:   filepath.Join(root, ".vrooli", "execution-policy.json"),
		AgentService: agent,
	})
	_, err := service.UpdatePolicy(context.Background(), Policy{
		DefaultMode:         ModeScheduled,
		DefaultDelaySeconds: 600,
	})
	if err != nil {
		t.Fatalf("UpdatePolicy error: %v", err)
	}

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "policy-idea",
		Mode:        "",
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	if record.Mode != ModeScheduled {
		t.Fatalf("expected scheduled mode from policy, got %s", record.Mode)
	}
	if record.Status != StatusScheduled {
		t.Fatalf("expected scheduled status, got %s", record.Status)
	}
	if record.ScheduledAt == "" {
		t.Fatalf("expected scheduled_at to be populated")
	}
}

func TestQueueBacklog_RejectsDelayForNonScheduledModes(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "delay-manual", map[string]any{
		"name":        "delay-manual",
		"title":       "Delay Manual",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})

	service := NewService(ServiceConfig{
		RootDir:   root,
		StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
	})

	_, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind:  "idea",
		BacklogName:  "delay-manual",
		Mode:         ModeManual,
		DelaySeconds: 10,
	})
	if err == nil {
		t.Fatal("expected error when delay_seconds is provided for manual mode")
	}
}

func TestQueueBacklog_RejectsScheduledModeWithoutEffectiveDelay(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "scheduled-no-delay", map[string]any{
		"name":        "scheduled-no-delay",
		"title":       "Scheduled No Delay",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	testPolicyPath := filepath.Join(root, ".vrooli", "execution-policy.json")
	mustWritePolicy(t, testPolicyPath, map[string]any{
		"default_mode":          "scheduled",
		"default_delay_seconds": 0,
	})

	service := NewService(ServiceConfig{
		RootDir:    root,
		StorePath:  filepath.Join(root, ".vrooli", "execution-runs.json"),
		PolicyPath: testPolicyPath,
	})

	_, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "scheduled-no-delay",
		Mode:        ModeScheduled,
	})
	if err == nil {
		t.Fatal("expected error for scheduled mode without effective delay")
	}
}

func TestQueueBacklog_AllowsArchivedIdeas(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "archived-idea", map[string]any{
		"name":        "archived-idea",
		"title":       "Archived Idea",
		"description": "desc",
		"status":      "archived",
		"priority":    3,
		"tags":        []string{},
	})
	mustWritePlanFile(t, root, "idea", "archived-idea")

	service := NewService(ServiceConfig{
		RootDir:   root,
		StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "archived-idea",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", record.Status)
	}
}

func TestQueueBacklog_YOLORollsBackWhenSpawnFails(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "rollback-idea", map[string]any{
		"name":        "rollback-idea",
		"title":       "Rollback Idea",
		"description": "desc",
		"status":      "archived",
		"priority":    3,
		"tags":        []string{},
	})
	mustWritePlanFile(t, root, "idea", "rollback-idea")

	agent := &stubAgentService{spawnErr: errors.New("spawn failed")}
	service := NewService(ServiceConfig{
		RootDir:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})

	_, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "rollback-idea",
		Mode:        ModeYOLO,
	})
	if err == nil {
		t.Fatal("expected queue error when spawn fails")
	}

	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "rollback-idea", "spec.json"))
	if storedItem["status"] != "archived" {
		t.Fatalf("expected archived status restored, got %#v", storedItem["status"])
	}

	records := mustLoadRecords(t, filepath.Join(root, ".vrooli", "execution-runs.json"))
	if len(records) != 0 {
		t.Fatalf("expected rollback to remove execution record, got %d", len(records))
	}
}

func TestCancel_RestoresArchivedStatus(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "archived-cancel", map[string]any{
		"name":          "archived-cancel",
		"title":         "Archived Cancel",
		"description":   "desc",
		"status":        "archived",
		"priority":      3,
		"tags":          []string{},
		"archiveReason": "scenario deleted with archive=true",
	})
	mustWritePlanFile(t, root, "idea", "archived-cancel")

	service := NewService(ServiceConfig{
		RootDir:   root,
		StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "archived-cancel",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}

	_, err = service.Cancel(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}

	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "archived-cancel", "spec.json"))
	if storedItem["status"] != "archived" {
		t.Fatalf("expected archived status after cancel, got %#v", storedItem["status"])
	}
	if storedItem["archiveReason"] != "scenario deleted with archive=true" {
		t.Fatalf("expected archive metadata preserved, got %#v", storedItem["archiveReason"])
	}
}

func TestCancel_RestoresArchivedStatusAfterForcedQueue(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "archived-cancel-forced", map[string]any{
		"name":          "archived-cancel-forced",
		"title":         "Archived Cancel Forced",
		"description":   "desc",
		"status":        "archived",
		"priority":      3,
		"tags":          []string{},
		"archiveReason": "scenario deleted with archive=true",
	})
	mustWritePlanFile(t, root, "idea", "archived-cancel-forced")

	service := NewService(ServiceConfig{
		RootDir:   root,
		StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "archived-cancel-forced",
		Mode:        ModeManual,
		Force:       true,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}

	_, err = service.Cancel(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}

	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "archived-cancel-forced", "spec.json"))
	if storedItem["status"] != "archived" {
		t.Fatalf("expected archived status after cancel, got %#v", storedItem["status"])
	}
}

func TestCancel_ReturnsErrorWhenRestoreFails(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "cancel-restore-error", map[string]any{
		"name":        "cancel-restore-error",
		"title":       "Cancel Restore Error",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWritePlanFile(t, root, "idea", "cancel-restore-error")

	service := NewService(ServiceConfig{
		RootDir:   root,
		StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "cancel-restore-error",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}

	specPath := filepath.Join(root, "ideas", "cancel-restore-error", "spec.json")
	if err := os.Remove(specPath); err != nil {
		t.Fatalf("remove spec for restore failure simulation: %v", err)
	}

	_, err = service.Cancel(context.Background(), record.ExecutionID)
	if err == nil {
		t.Fatal("expected cancel restore error")
	}
	if !strings.Contains(err.Error(), "failed to load backlog item for cancel restore") {
		t.Fatalf("expected restore load error, got %v", err)
	}
}

func TestUpdatePolicy_RejectsInvalidScheduledDefaults(t *testing.T) {
	root := t.TempDir()
	service := NewService(ServiceConfig{
		RootDir:    root,
		StorePath:  filepath.Join(root, ".vrooli", "execution-runs.json"),
		PolicyPath: filepath.Join(root, ".vrooli", "execution-policy.json"),
	})

	_, err := service.UpdatePolicy(context.Background(), Policy{
		DefaultMode:         ModeScheduled,
		DefaultDelaySeconds: 0,
	})
	if err == nil {
		t.Fatal("expected error when scheduled default mode has non-positive delay")
	}
}

func mustWriteBacklogItem(t *testing.T, root, kind, name string, payload map[string]any) {
	t.Helper()
	kindDir := "ideas"
	switch kind {
	case "research":
		kindDir = "research"
	case "fix":
		kindDir = "fix"
	case "execute":
		kindDir = "execute"
	case "chore":
		kindDir = "chore"
	}
	dir := filepath.Join(root, kindDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir backlog item: %v", err)
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "spec.json"), bytes, 0o644); err != nil {
		t.Fatalf("write spec.json: %v", err)
	}
}

// mustWritePlanFile creates a plan.md in the item directory so that
// workshop readiness preflight passes (plan exists with no rounds = manually created plan).
func mustWritePlanFile(t *testing.T, root, kind, name string) {
	t.Helper()
	kindDir := "ideas"
	switch kind {
	case "research":
		kindDir = "research"
	case "fix":
		kindDir = "fix"
	case "execute":
		kindDir = "execute"
	case "chore":
		kindDir = "chore"
	}
	dir := filepath.Join(root, kindDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir for plan.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plan.md"), []byte("# Plan\nManually created plan for testing."), 0o644); err != nil {
		t.Fatalf("write plan.md: %v", err)
	}
}

func mustLoadBacklogItem(t *testing.T, path string) map[string]any {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var value map[string]any
	if err := json.Unmarshal(bytes, &value); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return value
}

func mustWritePolicy(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir policy dir: %v", err)
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal policy payload: %v", err)
	}
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		t.Fatalf("write policy file: %v", err)
	}
}


func TestMapRunStatus_DirectMappings(t *testing.T) {
	tests := []struct {
		input    string
		errorMsg string
		want     Status
		wantMsg  string
	}{
		{"pending", "", StatusStarting, ""},
		{"starting", "", StatusStarting, ""},
		{"running", "", StatusRunning, ""},
		{"needs_review", "", StatusNeedsReview, ""},
		{"complete", "", StatusCompleted, ""},
		{"failed", "boom", StatusFailed, "boom"},
		{"failed", "", StatusFailed, "agent-manager run failed"},
		{"cancelled", "", StatusCanceled, ""},
		{"unspecified", "", StatusRunning, ""},
		{"RUNNING", "", StatusRunning, ""},
		{"unknown-value", "", StatusRunning, ""},
	}
	for _, tc := range tests {
		got, msg := mapRunStatus(tc.input, tc.errorMsg)
		if got != tc.want {
			t.Errorf("mapRunStatus(%q, %q): got %s, want %s", tc.input, tc.errorMsg, got, tc.want)
		}
		if msg != tc.wantMsg {
			t.Errorf("mapRunStatus(%q, %q): msg got %q, want %q", tc.input, tc.errorMsg, msg, tc.wantMsg)
		}
	}
}

type stubInspector struct {
	state agentmanager.RunState
	err   error
}

func (s *stubInspector) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	if s.err != nil {
		return agentmanager.RunState{}, s.err
	}
	return s.state, nil
}

type stubStopper struct {
	stopCalls int
	err       error
}

func (s *stubStopper) StopRun(_ context.Context, _ string) error {
	s.stopCalls++
	return s.err
}

func TestCancel_StartingExecution(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "starting-cancel", map[string]any{
		"name":        "starting-cancel",
		"title":       "Starting Cancel",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWritePlanFile(t, root, "idea", "starting-cancel")

	stopper := &stubStopper{}
	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		RootDir:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	service.stopper = stopper

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "starting-cancel",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	started, err := service.Start(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if started.Status != StatusStarting {
		t.Fatalf("expected starting, got %s", started.Status)
	}

	canceled, err := service.Cancel(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}
	if canceled.Status != StatusCanceled {
		t.Fatalf("expected canceled, got %s", canceled.Status)
	}
	if stopper.stopCalls != 1 {
		t.Fatalf("expected 1 StopRun call, got %d", stopper.stopCalls)
	}
}

func TestCancel_NeedsReviewExecution(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "review-cancel", map[string]any{
		"name":        "review-cancel",
		"title":       "Review Cancel",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWritePlanFile(t, root, "idea", "review-cancel")

	stopper := &stubStopper{}
	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		RootDir:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})
	service.stopper = stopper

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "review-cancel",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	started, err := service.Start(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}

	// Manually set to needs_review to simulate agent-manager transition
	records, idx, _ := service.loadRecordLocked(started.ExecutionID)
	records[idx].Status = StatusNeedsReview
	_ = service.store.Save(records)

	canceled, err := service.Cancel(context.Background(), started.ExecutionID)
	if err != nil {
		t.Fatalf("Cancel error: %v", err)
	}
	if canceled.Status != StatusCanceled {
		t.Fatalf("expected canceled, got %s", canceled.Status)
	}
}

func TestMigrateRecords_OrphanedRunning(t *testing.T) {
	records := []Record{
		{ExecutionID: "ok", Status: StatusRunning, RunID: "run-1"},
		{ExecutionID: "orphan", Status: StatusRunning, RunID: ""},
		{ExecutionID: "done", Status: StatusCompleted},
	}
	migrated := migrateRecords(records)
	if migrated[0].Status != StatusRunning {
		t.Fatalf("expected running with RunID to stay running, got %s", migrated[0].Status)
	}
	if migrated[1].Status != StatusFailed {
		t.Fatalf("expected orphaned running to become failed, got %s", migrated[1].Status)
	}
	if migrated[1].FailureReason != "orphaned execution: no run ID" {
		t.Fatalf("expected orphan failure reason, got %q", migrated[1].FailureReason)
	}
	if migrated[2].Status != StatusCompleted {
		t.Fatalf("expected completed to stay completed, got %s", migrated[2].Status)
	}
}

// TestRefreshRunning_FailedRunSetsBacklogFailed verifies that when an agent-manager
// run transitions to "failed", the backlog item status is set to "failed" (not
// silently reverted to its previous status).
func TestRefreshRunning_FailedRunSetsBacklogFailed(t *testing.T) {
	root := t.TempDir()
	storePath := filepath.Join(root, ".vrooli", "execution-runs.json")

	mustWriteBacklogItem(t, root, "idea", "fail-status", map[string]any{
		"name":        "fail-status",
		"title":       "Fail Status",
		"description": "desc",
		"status":      "in_progress",
		"priority":    3,
		"tags":        []string{},
	})

	// Seed an execution record that looks like a running execution.
	store := NewStore(storePath)
	if err := store.Save([]Record{{
		ExecutionID:    "exec-fail-1",
		BacklogKind:    "idea",
		BacklogName:    "fail-status",
		PreviousStatus: "backlog",
		Status:         StatusRunning,
		Mode:           ModeManual,
		RunID:          "run-fail-1",
		CreatedAt:      "2026-01-28T00:00:00Z",
		UpdatedAt:      "2026-01-28T00:00:00Z",
	}}); err != nil {
		t.Fatalf("save seed record: %v", err)
	}

	// Inspector returns "failed" for the run.
	inspector := &stubInspector{
		state: agentmanager.RunState{
			RunID:      "run-fail-1",
			Status:     "failed",
			ErrorMsg:   "agent crashed",
			FinishedAt: "2026-01-28T01:00:00Z",
		},
	}

	service := NewService(ServiceConfig{
		RootDir:   root,
		StorePath: storePath,
	})
	service.inspector = inspector

	// List triggers refreshRunningLocked which should detect the failed run.
	records, err := service.List(context.Background(), ListFilters{})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Status != StatusFailed {
		t.Fatalf("expected execution status failed, got %s", records[0].Status)
	}

	// Verify the backlog item was set to "failed" (not reverted to "backlog").
	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "fail-status", "spec.json"))
	if storedItem["status"] != "failed" {
		t.Fatalf("expected backlog status 'failed', got %#v", storedItem["status"])
	}
}

// TestRefreshRunning_CanceledRunRestoresBacklogStatus verifies that when a run
// is canceled, the backlog item status is restored to its previous status.
func TestRefreshRunning_CanceledRunRestoresBacklogStatus(t *testing.T) {
	root := t.TempDir()
	storePath := filepath.Join(root, ".vrooli", "execution-runs.json")

	mustWriteBacklogItem(t, root, "idea", "cancel-restore", map[string]any{
		"name":        "cancel-restore",
		"title":       "Cancel Restore",
		"description": "desc",
		"status":      "in_progress",
		"priority":    3,
		"tags":        []string{},
	})

	store := NewStore(storePath)
	if err := store.Save([]Record{{
		ExecutionID:    "exec-cancel-1",
		BacklogKind:    "idea",
		BacklogName:    "cancel-restore",
		PreviousStatus: "ready",
		Status:         StatusRunning,
		Mode:           ModeManual,
		RunID:          "run-cancel-1",
		CreatedAt:      "2026-01-28T00:00:00Z",
		UpdatedAt:      "2026-01-28T00:00:00Z",
	}}); err != nil {
		t.Fatalf("save seed record: %v", err)
	}

	inspector := &stubInspector{
		state: agentmanager.RunState{
			RunID:      "run-cancel-1",
			Status:     "cancelled",
			FinishedAt: "2026-01-28T01:00:00Z",
		},
	}

	service := NewService(ServiceConfig{
		RootDir:   root,
		StorePath: storePath,
	})
	service.inspector = inspector

	records, err := service.List(context.Background(), ListFilters{})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Status != StatusCanceled {
		t.Fatalf("expected execution status canceled, got %s", records[0].Status)
	}

	// Verify the backlog item was restored to previous status "ready" (not "failed").
	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "cancel-restore", "spec.json"))
	if storedItem["status"] != "ready" {
		t.Fatalf("expected backlog status 'ready', got %#v", storedItem["status"])
	}
}

func mustLoadRecords(t *testing.T, path string) []map[string]any {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read records: %v", err)
	}
	var records []map[string]any
	if err := json.Unmarshal(bytes, &records); err != nil {
		t.Fatalf("unmarshal records: %v", err)
	}
	return records
}
