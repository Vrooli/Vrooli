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
	"swarm-manager/internal/handoff"
	"swarm-manager/internal/promptmanager"
)

// stubPolicyProvider implements PolicyProvider for tests.
type stubPolicyProvider struct {
	policy Policy
}

func (s *stubPolicyProvider) LoadPolicy() (Policy, error) {
	return s.policy, nil
}

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

type snapshotAgentService struct {
	stubAgentService
	runStateCalls int
}

func (s *snapshotAgentService) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	s.runStateCalls++
	return agentmanager.RunState{Status: "completed", FinishedAt: "2026-05-14T00:00:00Z"}, nil
}

func TestListSnapshotDoesNotProcessActiveExecutions(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	agent := &snapshotAgentService{}
	svc := NewService(ServiceConfig{
		DataRoot:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: agent,
	})
	if err := svc.store.Save([]Record{{
		ExecutionID: "exec-1",
		BacklogKind: "execute",
		BacklogName: "slow-graph",
		Status:      StatusRunning,
		RunID:       "run-1",
		CreatedAt:   "2026-05-14T00:00:00Z",
	}}); err != nil {
		t.Fatalf("save executions: %v", err)
	}

	records, err := svc.ListSnapshot(context.Background(), ListFilters{})
	if err != nil {
		t.Fatalf("ListSnapshot: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("ListSnapshot returned %d records, want 1", len(records))
	}
	if records[0].Status != StatusRunning {
		t.Fatalf("snapshot status = %q, want persisted running", records[0].Status)
	}
	if agent.runStateCalls != 0 {
		t.Fatalf("ListSnapshot called GetRunState %d times, want 0", agent.runStateCalls)
	}
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
	mustWriteDeliverableFile(t, root, "idea", "test-idea")

	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		DataRoot:      root,
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
	if started.PromptTrace.Purpose != "process" {
		t.Fatalf("expected purpose 'process', got %q", started.PromptTrace.Purpose)
	}
	if !strings.Contains(started.PromptTrace.Prompt, "<execution-context>") {
		t.Fatal("expected prompt to contain <execution-context> tag")
	}
	if !strings.Contains(started.PromptTrace.Prompt, "<implementation-plan path=\"plan.md\">") {
		t.Fatal("expected prompt to contain implementation plan tag")
	}
	if !strings.Contains(started.PromptTrace.Prompt, "Manually created plan for testing") {
		t.Fatal("expected prompt to contain plan.md content")
	}
	if !strings.Contains(started.PromptTrace.Prompt, "<idea-handoff>") {
		t.Fatal("expected prompt to contain idea handoff metadata")
	}
	for _, handoffFile := range []string{
		filepath.Join(root, "ideas", "test-idea", "handoff", "brief.md"),
		filepath.Join(root, "ideas", "test-idea", "handoff", "manifest.json"),
		filepath.Join(root, "ideas", "test-idea", "handoff", "source-index.json"),
	} {
		if _, err := os.Stat(handoffFile); err != nil {
			t.Fatalf("expected handoff file %s to exist: %v", handoffFile, err)
		}
	}
	if agent.spawnCalls != 1 {
		t.Fatalf("expected 1 spawn call, got %d", agent.spawnCalls)
	}
}

func TestQueueAndStartManualExecution_ResearchUsesConclusionDeliverable(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "research", "test-research", map[string]any{
		"name":        "test-research",
		"title":       "Research Test",
		"description": "desc",
		"status":      "backlog",
		"priority":    3,
		"tags":        []string{},
	})
	mustWriteDeliverableFile(t, root, "research", "test-research")

	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		DataRoot:      root,
		StorePath:    filepath.Join(root, ".vrooli", "execution-runs.json"),
		AgentService: agent,
		PromptClient: &promptmanager.MockClient{Result: "test prompt"},
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "research",
		BacklogName: "test-research",
		Mode:        ModeManual,
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}

	started, err := service.Start(context.Background(), record.ExecutionID)
	if err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if started.PromptTrace == nil {
		t.Fatal("expected prompt trace to be captured")
	}
	if !strings.Contains(started.PromptTrace.Prompt, "<research-conclusion path=\"conclusion.md\">") {
		t.Fatal("expected prompt to contain research conclusion tag")
	}
	if !strings.Contains(started.PromptTrace.Prompt, "Manually created conclusion for testing") {
		t.Fatal("expected prompt to contain conclusion.md content")
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
	mustWriteDeliverableFile(t, root, "idea", "policy-idea")

	service := NewService(ServiceConfig{
		DataRoot:   root,
		StorePath: filepath.Join(root, ".vrooli", "execution-runs.json"),
		PolicyProvider: &stubPolicyProvider{policy: Policy{
			DefaultMode: ModeManual,
		}},
	})

	record, err := service.QueueBacklog(context.Background(), CreateRequest{
		BacklogKind: "idea",
		BacklogName: "policy-idea",
		Mode:        "",
	})
	if err != nil {
		t.Fatalf("QueueBacklog error: %v", err)
	}
	if record.Mode != ModeManual {
		t.Fatalf("expected manual mode from policy, got %s", record.Mode)
	}
	if record.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", record.Status)
	}
}

func TestQueueBacklog_AllowsArchivedIdeas(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "archived-idea", map[string]any{
		"name":        "archived-idea",
		"title":       "Archived Idea",
		"description": "desc",
		"status":      "ready",
		"priority":    3,
		"tags":        []string{},
		"archived_at": "2025-01-01T00:00:00Z",
	})
	mustWriteDeliverableFile(t, root, "idea", "archived-idea")

	service := NewService(ServiceConfig{
		DataRoot:   root,
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
		"status":      "ready",
		"priority":    3,
		"tags":        []string{},
		"archived_at": "2025-01-01T00:00:00Z",
	})
	mustWriteDeliverableFile(t, root, "idea", "rollback-idea")

	agent := &stubAgentService{spawnErr: errors.New("spawn failed")}
	service := NewService(ServiceConfig{
		DataRoot:      root,
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
	if storedItem["status"] != "ready" {
		t.Fatalf("expected ready status restored, got %#v", storedItem["status"])
	}

	records := mustLoadRecords(t, filepath.Join(root, ".vrooli", "execution-runs.json"))
	if len(records) != 0 {
		t.Fatalf("expected rollback to remove execution record, got %d", len(records))
	}
}

func TestCancel_RestoresArchivedIdeaStatus(t *testing.T) {
	root := t.TempDir()
	mustWriteBacklogItem(t, root, "idea", "archived-cancel", map[string]any{
		"name":          "archived-cancel",
		"title":         "Archived Cancel",
		"description":   "desc",
		"status":        "ready",
		"priority":      3,
		"tags":          []string{},
		"archived_at":   "2025-01-01T00:00:00Z",
		"archiveReason": "scenario deleted with archive=true",
	})
	mustWriteDeliverableFile(t, root, "idea", "archived-cancel")

	service := NewService(ServiceConfig{
		DataRoot:   root,
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
	if storedItem["status"] != "ready" {
		t.Fatalf("expected ready status after cancel, got %#v", storedItem["status"])
	}
	if storedItem["archived_at"] != "2025-01-01T00:00:00Z" {
		t.Fatalf("expected archived_at preserved, got %#v", storedItem["archived_at"])
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
		"status":        "ready",
		"priority":      3,
		"tags":          []string{},
		"archived_at":   "2025-01-01T00:00:00Z",
		"archiveReason": "scenario deleted with archive=true",
	})
	mustWriteDeliverableFile(t, root, "idea", "archived-cancel-forced")

	service := NewService(ServiceConfig{
		DataRoot:   root,
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
	if storedItem["status"] != "ready" {
		t.Fatalf("expected ready status after cancel, got %#v", storedItem["status"])
	}
	if storedItem["archived_at"] != "2025-01-01T00:00:00Z" {
		t.Fatalf("expected archived_at preserved after cancel, got %#v", storedItem["archived_at"])
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
	mustWriteDeliverableFile(t, root, "idea", "cancel-restore-error")

	service := NewService(ServiceConfig{
		DataRoot:   root,
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
	if !strings.Contains(err.Error(), "backlog status restore failed") {
		t.Fatalf("expected restore error, got %v", err)
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

// mustWriteDeliverableFile creates the primary workshop artifact in the item
// directory so that workshop readiness preflight passes (deliverable exists
// with no rounds = manually created artifact).
func mustWriteDeliverableFile(t *testing.T, root, kind, name string) {
	t.Helper()
	kindDir := "ideas"
	deliverablePath := "plan.md"
	switch kind {
	case "research":
		kindDir = "research"
		deliverablePath = "conclusion.md"
	case "fix":
		kindDir = "fix"
	case "execute":
		kindDir = "execute"
	case "chore":
		kindDir = "chore"
	}
	dir := filepath.Join(root, kindDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir for deliverable: %v", err)
	}
	content := "# Plan\nManually created plan for testing."
	if deliverablePath == "conclusion.md" {
		content = "# Conclusion\nManually created conclusion for testing."
	}
	if err := os.WriteFile(filepath.Join(dir, deliverablePath), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", deliverablePath, err)
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
		tracker := &runTracker{}
		got, msg := mapRunStatus(tc.input, tc.errorMsg, tracker, 5)
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
	mustWriteDeliverableFile(t, root, "idea", "starting-cancel")

	stopper := &stubStopper{}
	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		DataRoot:      root,
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
	mustWriteDeliverableFile(t, root, "idea", "review-cancel")

	stopper := &stubStopper{}
	agent := &stubAgentService{}
	service := NewService(ServiceConfig{
		DataRoot:      root,
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

// TestRefreshRunning_FailedRunSetsBacklogInReview verifies that when an
// agent-manager run transitions to "failed", the backlog item lands in
// in_review (not terminal) so the review agent can document the failure and
// the user decides the terminal state via review-decide. The execution
// record itself still records StatusFailed.
func TestRefreshRunning_FailedRunSetsBacklogInReview(t *testing.T) {
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
		DataRoot:   root,
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

	// Verify the backlog item landed in "in_review" — terminal transitions
	// are user-only (plan §W1); the review agent will document the failure.
	storedItem := mustLoadBacklogItem(t, filepath.Join(root, "ideas", "fail-status", "spec.json"))
	if storedItem["status"] != "in_review" {
		t.Fatalf("expected backlog status 'in_review', got %#v", storedItem["status"])
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
		DataRoot:   root,
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

func TestIsFinalizationEligible(t *testing.T) {
	tests := []struct {
		name     string
		record   Record
		expected bool
	}{
		{name: "default process run", record: Record{}, expected: true},
		{name: "fixup run", record: Record{PromptTrace: &PromptTrace{Purpose: "fixup"}}, expected: true},
		{name: "followup run", record: Record{PromptTrace: &PromptTrace{Purpose: "followup"}}, expected: true},
		{name: "custom run", record: Record{PromptTrace: &PromptTrace{Purpose: "custom"}}, expected: true},
		{name: "research run excluded", record: Record{PromptTrace: &PromptTrace{Purpose: "research"}}, expected: false},
		{name: "archive run excluded", record: Record{ArchiveContext: &ArchiveContext{ScenarioName: "web-console"}}, expected: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if actual := isFinalizationEligible(tc.record); actual != tc.expected {
				t.Fatalf("expected eligible=%t, got %t", tc.expected, actual)
			}
		})
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

func TestRecordToProto_MapsFinalization(t *testing.T) {
	record := Record{
		ExecutionID:       "exec-review-1",
		BacklogKind:       "idea",
		BacklogName:       "reviewed-idea",
		Status:            StatusNeedsFixup,
		Mode:              ModeYOLO,
		ParentExecutionID: "parent-exec-0",
		FixupAttempt:      2,
		Finalization: &Finalization{
			Eligible:                true,
			Status:                  FinalizationStatusCompleted,
			Phase:                   FinalizationPhaseCompleted,
			ScopeSource:             FinalizationScopeSandboxDiff,
			StartedAt:               "2026-03-24T10:00:00Z",
			CompletedAt:             "2026-03-24T12:00:00Z",
			AggregateClassification: FinalizationAggregateNeedsWork,
			AggregateSummary:        "Tests failing",
			AffectedScenarios:       []string{"web-console"},
			Warnings: []FinalizationWarning{{
				Code:      "health_retry",
				Message:   "restarted twice",
				Retryable: true,
				CreatedAt: "2026-03-24T11:00:00Z",
			}},
			Scenarios: []ScenarioFinalization{
				{
					ScenarioName: "web-console",
					ChangedPaths: []string{"scenarios/web-console/ui/src/App.tsx"},
					Restart: RestartResult{
						Status:     FinalizationStatusCompleted,
						Attempts:   2,
						StartedAt:  "2026-03-24T10:00:00Z",
						FinishedAt: "2026-03-24T10:01:00Z",
					},
					Health: HealthCheckResult{
						Status:         FinalizationStatusCompleted,
						ScenarioStatus: "running",
						HealthStatus:   "healthy",
						SchemaValid:    true,
						Details:        "scenario is healthy",
						CheckedAt:      "2026-03-24T10:02:00Z",
					},
					Review: ScenarioReviewStep{
						Status: FinalizationStatusCompleted,
						JobID:  "review-job-1",
						Result: &ReviewResult{
							JobID:          "review-job-1",
							Classification: "needs_work",
							Summary:        "Tests failing",
							ReviewedAt:     "2026-03-24T12:00:00Z",
							Dimensions: []ReviewDimension{
								{Name: "tests", Status: "red", Details: "3 tests failing"},
								{Name: "lint", Status: "green"},
							},
						},
					},
				},
			},
		},
		CreatedAt: "2026-03-24T00:00:00Z",
		UpdatedAt: "2026-03-24T01:00:00Z",
	}
	pb := recordToProto(record)

	if pb.ParentExecutionId == nil || *pb.ParentExecutionId != "parent-exec-0" {
		t.Fatalf("expected parent_execution_id parent-exec-0, got %v", pb.ParentExecutionId)
	}
	if pb.FixupAttempt != 2 {
		t.Fatalf("expected fixup_attempt 2, got %d", pb.FixupAttempt)
	}
	if pb.Finalization == nil {
		t.Fatal("expected finalization to be set")
	}
	if pb.Finalization.AggregateClassification != "needs_work" {
		t.Fatalf("expected aggregate classification needs_work, got %s", pb.Finalization.AggregateClassification)
	}
	if pb.Finalization.AggregateSummary == nil || *pb.Finalization.AggregateSummary != "Tests failing" {
		t.Fatalf("expected aggregate summary 'Tests failing', got %v", pb.Finalization.AggregateSummary)
	}
	if len(pb.Finalization.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario finalization, got %d", len(pb.Finalization.Scenarios))
	}
	if len(pb.Finalization.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(pb.Finalization.Warnings))
	}
	scenario := pb.Finalization.Scenarios[0]
	if scenario.Review == nil || scenario.Review.Result == nil {
		t.Fatal("expected scenario review result to be set")
	}
	dim0 := scenario.Review.Result.Dimensions[0]
	if dim0.Name != "tests" || dim0.Status != "red" {
		t.Fatalf("expected tests/red, got %s/%s", dim0.Name, dim0.Status)
	}
	if dim0.Details == nil || *dim0.Details != "3 tests failing" {
		t.Fatalf("expected details '3 tests failing', got %v", dim0.Details)
	}
	dim1 := scenario.Review.Result.Dimensions[1]
	if dim1.Name != "lint" || dim1.Status != "green" {
		t.Fatalf("expected lint/green, got %s/%s", dim1.Name, dim1.Status)
	}
}

// --- stubReviewClient for testing ---

type stubReviewClient struct {
	triggerJobID string
	triggerErr   error
	pollResult   *ReviewResult
	pollDone     bool
	pollErr      error
	pingErr      error
}

func (s *stubReviewClient) TriggerReview(_ context.Context, _ ReviewRequest) (string, error) {
	return s.triggerJobID, s.triggerErr
}

func (s *stubReviewClient) PollReview(_ context.Context, _ string) (*ReviewResult, bool, error) {
	return s.pollResult, s.pollDone, s.pollErr
}

func (s *stubReviewClient) Ping(_ context.Context) error {
	return s.pingErr
}

func TestTriggerReview_CompletedExecution(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "exec.json"))
	records := []Record{{
		ExecutionID: "exec-tr-1",
		BacklogKind: "execute",
		BacklogName: "my-feature",
		Status:      StatusCompleted,
		Mode:        ModeYOLO,
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}}
	if err := store.Save(records); err != nil {
		t.Fatal(err)
	}

	// Create a backlog spec with acceptance_allow so scenario extraction works.
	specDir := filepath.Join(dir, "execute", "my-feature")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := `{"name":"my-feature","acceptance_allow":["scenarios/web-console/**"]}`
	if err := os.WriteFile(filepath.Join(specDir, "spec.json"), []byte(spec), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &Service{
		dataRoot:       dir,
		repoRoot:       dir,
		store:          store,
		reviewClient:   &stubReviewClient{triggerJobID: "job-new"},
		policyProvider: &defaultPolicyProvider{},
	}

	result, err := svc.TriggerReview(context.Background(), "exec-tr-1")
	if err != nil {
		t.Fatalf("TriggerReview failed: %v", err)
	}
	if result.Status != StatusValidating {
		t.Fatalf("expected status validating, got %s", result.Status)
	}
	if result.Finalization == nil {
		t.Fatal("expected finalization to be initialized")
	}
	if result.Finalization.Status != FinalizationStatusPending {
		t.Fatalf("expected pending finalization status, got %s", result.Finalization.Status)
	}
	if result.Finalization.Phase != FinalizationPhaseScopeDetection {
		t.Fatalf("expected scope_detection phase, got %s", result.Finalization.Phase)
	}
}

func TestTriggerReview_WrongStatus(t *testing.T) {
	for _, status := range []Status{StatusRunning, StatusPending, StatusStarting, StatusValidating} {
		t.Run(string(status), func(t *testing.T) {
			dir := t.TempDir()
			store := NewStore(filepath.Join(dir, "exec.json"))
			records := []Record{{
				ExecutionID: "exec-tr-2",
				BacklogKind: "execute",
				BacklogName: "test",
				Status:      status,
				RunID:       "run-placeholder", // Prevent migrateRecords from changing running→failed
				Mode:        ModeYOLO,
				CreatedAt:   nowRFC3339(),
				UpdatedAt:   nowRFC3339(),
			}}
			if err := store.Save(records); err != nil {
				t.Fatal(err)
			}

			svc := &Service{
				dataRoot:       dir,
				repoRoot:       dir,
				store:          store,
				reviewClient:   &stubReviewClient{triggerJobID: "job-x"},
				policyProvider: &defaultPolicyProvider{},
			}

			_, err := svc.TriggerReview(context.Background(), "exec-tr-2")
			if err == nil {
				t.Fatal("expected error for non-terminal execution")
			}
			if !strings.Contains(err.Error(), "cannot trigger post-run checks") {
				t.Fatalf("expected status validation error, got: %v", err)
			}
		})
	}
}

func TestTriggerReview_MissingExecution(t *testing.T) {
	svc := &Service{
		store:          NewStore(filepath.Join(t.TempDir(), "exec.json")),
		policyProvider: &defaultPolicyProvider{},
	}
	_, err := svc.TriggerReview(context.Background(), "exec-x")
	if err == nil {
		t.Fatal("expected error when execution is missing")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got: %v", err)
	}
}

func TestRecordToProto_MapsLegacyReviewFieldsIntoFinalization(t *testing.T) {
	record := Record{
		ExecutionID:            "exec-new-fields",
		BacklogKind:            "execute",
		BacklogName:            "test",
		Status:                 StatusCompleted,
		Mode:                   ModeYOLO,
		LegacyReviewSkipReason: "GCT unavailable: connection refused",
		LegacyReviewStartedAt:  "2026-03-24T10:00:00Z",
		CreatedAt:              "2026-03-24T00:00:00Z",
		UpdatedAt:              "2026-03-24T01:00:00Z",
	}
	pb := recordToProto(record)

	if pb.Finalization == nil {
		t.Fatal("expected finalization to be synthesized")
	}
	if pb.Finalization.SkipReason == nil || *pb.Finalization.SkipReason != "GCT unavailable: connection refused" {
		t.Fatalf("expected skip reason to be mapped, got %v", pb.Finalization.SkipReason)
	}
	if pb.Finalization.StartedAt == nil || *pb.Finalization.StartedAt != "2026-03-24T10:00:00Z" {
		t.Fatalf("expected started_at to be mapped, got %v", pb.Finalization.StartedAt)
	}
}

func TestRecordToProto_OmitsEmptyFinalization(t *testing.T) {
	record := Record{
		ExecutionID: "exec-empty-fields",
		BacklogKind: "execute",
		BacklogName: "test",
		Status:      StatusCompleted,
		Mode:        ModeYOLO,
		CreatedAt:   "2026-03-24T00:00:00Z",
		UpdatedAt:   "2026-03-24T01:00:00Z",
	}
	pb := recordToProto(record)

	if pb.Finalization != nil {
		t.Fatalf("expected nil finalization for empty record, got %v", pb.Finalization)
	}
}

// --- buildExecutionPrompt tests ---

func TestBuildExecutionPrompt_ProcessRun(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "idea",
		Name:               "video-studio",
		Title:              "Video Studio",
		ItemFolder:         "/path/to/ideas/video-studio",
		RunType:            "process",
		DeliverablePath:    "plan.md",
		DeliverableContent: "# Plan\nBuild a video editor.",
		IdeaHandoff: &handoff.Package{
			Dir:             "/path/to/ideas/video-studio/handoff",
			BriefPath:       "/path/to/ideas/video-studio/handoff/brief.md",
			ManifestPath:    "/path/to/ideas/video-studio/handoff/manifest.json",
			SourceIndexPath: "/path/to/ideas/video-studio/handoff/source-index.json",
			BriefMarkdown:   "# Idea Execution Handoff\n",
		},
	})

	// Execution context tag present with metadata.
	if !strings.Contains(prompt, "<execution-context>") || !strings.Contains(prompt, "</execution-context>") {
		t.Error("missing execution-context tags")
	}
	if !strings.Contains(prompt, "Backlog item: idea/video-studio") {
		t.Error("missing backlog item line")
	}
	if !strings.Contains(prompt, "Title: Video Studio") {
		t.Error("missing title line")
	}
	if !strings.Contains(prompt, "Item folder: /path/to/ideas/video-studio") {
		t.Error("missing item folder line")
	}
	if !strings.Contains(prompt, "Run type: process") {
		t.Error("missing run type line")
	}

	// Plan tag present with content.
	if !strings.Contains(prompt, "<implementation-plan path=\"plan.md\">") || !strings.Contains(prompt, "</implementation-plan>") {
		t.Error("missing implementation-plan tags")
	}
	if !strings.Contains(prompt, "Build a video editor.") {
		t.Error("missing plan content")
	}
	if !strings.Contains(prompt, "<idea-handoff>") || !strings.Contains(prompt, "<idea-handoff-brief path=\"/path/to/ideas/video-studio/handoff/brief.md\">") {
		t.Error("missing idea handoff tags")
	}
	if !strings.Contains(prompt, "Use brief.md as the ecosystem-manager task notes") {
		t.Error("missing downstream handoff instruction")
	}

	// No review or follow-up tags for a process run.
	if strings.Contains(prompt, "<review-feedback>") {
		t.Error("process run should not have review-feedback tag")
	}
	if strings.Contains(prompt, "<follow-up-context>") {
		t.Error("process run should not have follow-up-context tag")
	}
}

func TestBuildExecutionPrompt_FixupRun(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "fix",
		Name:               "login-crash",
		Title:              "Fix Login Crash",
		ItemFolder:         "/path/to/fix/login-crash",
		RunType:            "fixup",
		DeliverablePath:    "plan.md",
		DeliverableContent: "# Plan\nFix the nil pointer.",
		ReviewFeedback:     "Tests still failing.\n- test_coverage (red): Missing edge case test",
	})

	if !strings.Contains(prompt, "Run type: fixup") {
		t.Error("missing fixup run type")
	}

	// Review feedback tag present.
	if !strings.Contains(prompt, "<review-feedback>") || !strings.Contains(prompt, "</review-feedback>") {
		t.Error("missing review-feedback tags")
	}
	if !strings.Contains(prompt, "Tests still failing.") {
		t.Error("missing review summary in prompt")
	}
	if !strings.Contains(prompt, "Missing edge case test") {
		t.Error("missing review dimension detail")
	}

	// Plan still included.
	if !strings.Contains(prompt, "<implementation-plan path=\"plan.md\">") {
		t.Error("fixup run should still include implementation plan")
	}
	if !strings.Contains(prompt, "Fix the nil pointer.") {
		t.Error("missing plan content in fixup prompt")
	}
}

func TestBuildExecutionPrompt_FollowUpRun(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "execute",
		Name:               "dependency-update",
		Title:              "Update Dependencies",
		ItemFolder:         "/path/to/execute/dependency-update",
		RunType:            "followup",
		DeliverablePath:    "plan.md",
		DeliverableContent: "# Plan\nUpdate all Go deps.",
		FollowUpNote:       "Focus on the swarm-manager scenario only.",
	})

	if !strings.Contains(prompt, "Run type: followup") {
		t.Error("missing followup run type")
	}

	// Follow-up context tag present.
	if !strings.Contains(prompt, "<follow-up-context>") || !strings.Contains(prompt, "</follow-up-context>") {
		t.Error("missing follow-up-context tags")
	}
	if !strings.Contains(prompt, "Focus on the swarm-manager scenario only.") {
		t.Error("missing follow-up note content")
	}

	// Plan still included.
	if !strings.Contains(prompt, "Update all Go deps.") {
		t.Error("missing plan content")
	}
}

func TestBuildExecutionPrompt_NoPlan(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:       "chore",
		Name:       "cleanup",
		Title:      "Clean Up",
		ItemFolder: "/path/to/chore/cleanup",
		RunType:    "process",
	})

	if !strings.Contains(prompt, "<execution-context>") {
		t.Error("should still have execution context")
	}
	if strings.Contains(prompt, "<implementation-plan>") {
		t.Error("should not have implementation-plan tag when plan is empty")
	}
}

func TestBuildExecutionPrompt_EmptyOptionalSections(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "idea",
		Name:               "test",
		ItemFolder:         "/tmp/test",
		RunType:            "process",
		DeliverablePath:    "plan.md",
		DeliverableContent: "plan content",
		ReviewFeedback:     "",
		FollowUpNote:       "   ",
	})

	if strings.Contains(prompt, "<review-feedback>") {
		t.Error("empty review feedback should not produce tag")
	}
	if strings.Contains(prompt, "<follow-up-context>") {
		t.Error("whitespace-only follow-up note should not produce tag")
	}
}

func TestBuildExecutionPrompt_NoTitle(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "fix",
		Name:               "bug",
		ItemFolder:         "/tmp/fix/bug",
		RunType:            "process",
		DeliverablePath:    "plan.md",
		DeliverableContent: "fix it",
	})

	if strings.Contains(prompt, "Title:") {
		t.Error("should not include Title line when title is empty")
	}
}

func TestBuildExecutionPrompt_SuggestedSkills(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "execute",
		Name:               "refactor-api",
		Title:              "Refactor API",
		ItemFolder:         "/tmp/execute/refactor-api",
		RunType:            "process",
		DeliverablePath:    "plan.md",
		DeliverableContent: "# Plan\nRefactor.",
		SuggestedSkills:    []string{"refactor", "screaming-architecture-audit"},
	})

	if !strings.Contains(prompt, "<suggested-skills>") || !strings.Contains(prompt, "</suggested-skills>") {
		t.Error("missing suggested-skills tags")
	}
	if !strings.Contains(prompt, "prompt-manager skill read refactor") {
		t.Error("missing refactor skill in suggested-skills")
	}
	if !strings.Contains(prompt, "prompt-manager skill read screaming-architecture-audit") {
		t.Error("missing screaming-architecture-audit skill in suggested-skills")
	}
}

func TestBuildExecutionPrompt_NoSuggestedSkills(t *testing.T) {
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               "fix",
		Name:               "bug-fix",
		ItemFolder:         "/tmp/fix/bug-fix",
		RunType:            "process",
		DeliverablePath:    "plan.md",
		DeliverableContent: "fix it",
	})

	if strings.Contains(prompt, "<suggested-skills>") {
		t.Error("should not include suggested-skills when none provided")
	}
}

func TestBuildFinalizationFeedback_NilResult(t *testing.T) {
	if got := buildFinalizationFeedback(nil); got != "" {
		t.Errorf("expected empty string for nil result, got %q", got)
	}
}

func TestBuildFinalizationFeedback_WithDimensions(t *testing.T) {
	result := &Finalization{
		AggregateSummary: "Needs work.",
		Warnings: []FinalizationWarning{{
			Code:      "health_retry",
			Message:   "restarted twice",
			Retryable: true,
			CreatedAt: "2026-03-24T01:00:00Z",
		}},
		Scenarios: []ScenarioFinalization{{
			ScenarioName: "web-console",
			Review: ScenarioReviewStep{
				Result: &ReviewResult{
					Summary: "Needs work.",
					Dimensions: []ReviewDimension{
						{Name: "tests", Status: "red", Details: "3 tests failing"},
						{Name: "docs", Status: "green", Details: "OK"},
						{Name: "lint", Status: "yellow", Details: "2 warnings"},
					},
				},
			},
		}},
	}
	got := buildFinalizationFeedback(result)

	if !strings.Contains(got, "Needs work.") {
		t.Error("missing summary")
	}
	if !strings.Contains(got, "web-console tests (red): 3 tests failing") {
		t.Error("missing red dimension")
	}
	if strings.Contains(got, "docs (green)") {
		t.Error("green dimensions should be excluded")
	}
	if !strings.Contains(got, "warning [health_retry]: restarted twice") {
		t.Error("missing warning")
	}
	if !strings.Contains(got, "web-console lint (yellow): 2 warnings") {
		t.Error("missing yellow dimension")
	}
}

// --- checkReviewAgentEnabled tests ---

type errPolicyProvider struct {
	err error
}

func (p *errPolicyProvider) LoadPolicy() (Policy, error) {
	return Policy{}, p.err
}

func TestCheckReviewAgentEnabled_Enabled(t *testing.T) {
	svc := &Service{
		policyProvider: &stubPolicyProvider{policy: Policy{ReviewAgentEnabled: true}},
	}
	enabled, reason := svc.checkReviewAgentEnabled()
	if !enabled {
		t.Fatal("expected enabled=true")
	}
	if reason != "" {
		t.Fatalf("expected empty reason, got %q", reason)
	}
}

func TestCheckReviewAgentEnabled_Disabled(t *testing.T) {
	svc := &Service{
		policyProvider: &stubPolicyProvider{policy: Policy{ReviewAgentEnabled: false}},
	}
	enabled, reason := svc.checkReviewAgentEnabled()
	if enabled {
		t.Fatal("expected enabled=false")
	}
	if reason != finalizationWarningEvidenceSkippedDisabled {
		t.Fatalf("expected %q, got %q", finalizationWarningEvidenceSkippedDisabled, reason)
	}
}

func TestCheckReviewAgentEnabled_PolicyLoadError(t *testing.T) {
	svc := &Service{
		policyProvider: &errPolicyProvider{err: errors.New("disk full")},
	}
	enabled, reason := svc.checkReviewAgentEnabled()
	if enabled {
		t.Fatal("expected enabled=false on policy load error")
	}
	if reason != finalizationWarningEvidenceSkippedPolicyErr {
		t.Fatalf("expected %q, got %q", finalizationWarningEvidenceSkippedPolicyErr, reason)
	}
}

func TestEvidenceSkipMessage(t *testing.T) {
	svc := &Service{}
	msg := svc.evidenceSkipMessage(finalizationWarningEvidenceSkippedDisabled)
	if !strings.Contains(msg, "disabled in settings") {
		t.Fatalf("expected settings hint, got %q", msg)
	}
	msg = svc.evidenceSkipMessage(finalizationWarningEvidenceSkippedPolicyErr)
	if !strings.Contains(msg, "Could not load settings") {
		t.Fatalf("expected policy error hint, got %q", msg)
	}
	msg = svc.evidenceSkipMessage("unknown_code")
	if msg != "Evidence gathering was skipped." {
		t.Fatalf("expected fallback message, got %q", msg)
	}
}
