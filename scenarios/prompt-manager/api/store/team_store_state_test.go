package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"prompt-manager/teamconfig"
)

func setupStateTestStore(t *testing.T) *FileTeamStore {
	t.Helper()
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "store")
	if err := os.MkdirAll(filepath.Join(storeDir, "teams", "team-1", "members", "agent-1"), 0o755); err != nil {
		t.Fatal(err)
	}
	team := newIndependentTestTeam("team-1", "Test")
	team.Kind = KindTeam
	team.SchemaVersion = CurrentSchemaVersion
	team.Timestamps = NewTimestamps()
	if err := SaveJSON(filepath.Join(storeDir, "teams", "team-1", "team.json"), team); err != nil {
		t.Fatal(err)
	}
	return NewFileTeamStore(storeDir, storeDir, nil)
}

func TestGetSetLastHandoff(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	if err := s.SetLastHandoff(ctx, "team-1", "agent-1", "**Status**: Done"); err != nil {
		t.Fatalf("SetLastHandoff: %v", err)
	}

	got, err := s.GetLastHandoff(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatalf("GetLastHandoff: %v", err)
	}
	if got != "**Status**: Done" {
		t.Errorf("expected '**Status**: Done', got: %s", got)
	}
}

func TestGetLastHandoffMissing(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	got, err := s.GetLastHandoff(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatalf("GetLastHandoff: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty, got: %s", got)
	}
}

func TestAppendAndGetHandoffHistory(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	entries := []HandoffEntry{
		{AgentID: "agent-1", RunID: "run-1", Timestamp: "2025-01-01T00:00:00Z", Content: "First"},
		{AgentID: "agent-2", RunID: "run-2", Timestamp: "2025-01-01T01:00:00Z", Content: "Second"},
		{AgentID: "agent-1", RunID: "run-3", Timestamp: "2025-01-01T02:00:00Z", Content: "Third"},
	}
	for i := range entries {
		if err := s.AppendHandoffHistory(ctx, "team-1", &entries[i]); err != nil {
			t.Fatalf("AppendHandoffHistory: %v", err)
		}
	}

	// Get all
	all, err := s.GetHandoffHistory(ctx, "team-1", "", 0)
	if err != nil {
		t.Fatalf("GetHandoffHistory all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(all))
	}

	// Filter by agent
	agent1, err := s.GetHandoffHistory(ctx, "team-1", "agent-1", 0)
	if err != nil {
		t.Fatalf("GetHandoffHistory filtered: %v", err)
	}
	if len(agent1) != 2 {
		t.Fatalf("expected 2 entries for agent-1, got %d", len(agent1))
	}

	// Limit
	limited, err := s.GetHandoffHistory(ctx, "team-1", "", 2)
	if err != nil {
		t.Fatalf("GetHandoffHistory limited: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("expected 2 entries with limit, got %d", len(limited))
	}
	if limited[0].Content != "Third" {
		t.Errorf("expected 'Third' first in limited results (newest-first), got: %s", limited[0].Content)
	}
	if limited[1].Content != "Second" {
		t.Errorf("expected 'Second' second in limited results (newest-first), got: %s", limited[1].Content)
	}
}

func TestListHeartbeatAttemptsIncludesPreRunLastExecution(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	cfg := &HeartbeatConfig{
		Enabled:    true,
		Schedule:   "0 * * * *",
		ProfileKey: "prompt-manager-heartbeat",
		LastExecution: &HeartbeatExecResult{
			StartedAt: "2026-05-06T12:00:00Z",
			EndedAt:   "2026-05-06T12:00:01Z",
			Status:    HeartbeatStatusFailed,
			Error:     "creating run: validation error",
		},
	}
	if err := s.SetHeartbeatConfig(ctx, "team-1", "agent-1", cfg); err != nil {
		t.Fatalf("SetHeartbeatConfig: %v", err)
	}

	attempts, total, err := s.ListHeartbeatAttempts(ctx, "", "", HeartbeatStatusFailed, "prompt-manager-heartbeat", 10, 0)
	if err != nil {
		t.Fatalf("ListHeartbeatAttempts: %v", err)
	}
	if total != 1 || len(attempts) != 1 {
		t.Fatalf("expected one derived attempt, total=%d len=%d", total, len(attempts))
	}
	if attempts[0].Phase != "pre_run_failure" || attempts[0].Error == "" {
		t.Fatalf("unexpected derived attempt: %+v", attempts[0])
	}
}

func TestClearHandoffHistory(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	entries := []HandoffEntry{
		{AgentID: "agent-1", RunID: "r1", Timestamp: "2025-01-01T00:00:00Z", Content: "A1"},
		{AgentID: "agent-2", RunID: "r2", Timestamp: "2025-01-01T01:00:00Z", Content: "A2"},
		{AgentID: "agent-1", RunID: "r3", Timestamp: "2025-01-01T02:00:00Z", Content: "A1b"},
	}
	for i := range entries {
		if err := s.AppendHandoffHistory(ctx, "team-1", &entries[i]); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	// Clear by agent
	if err := s.ClearHandoffHistory(ctx, "team-1", "agent-1"); err != nil {
		t.Fatalf("ClearHandoffHistory agent-1: %v", err)
	}
	remaining, err := s.GetHandoffHistory(ctx, "team-1", "", 0)
	if err != nil {
		t.Fatalf("GetHandoffHistory after agent clear: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("expected 1 entry after agent clear, got %d", len(remaining))
	}
	if remaining[0].AgentID != "agent-2" {
		t.Errorf("expected agent-2, got %s", remaining[0].AgentID)
	}

	// Clear all
	if err := s.ClearHandoffHistory(ctx, "team-1", ""); err != nil {
		t.Fatalf("ClearHandoffHistory all: %v", err)
	}
	all, err := s.GetHandoffHistory(ctx, "team-1", "", 0)
	if err != nil {
		t.Fatalf("GetHandoffHistory after full clear: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 entries after full clear, got %d", len(all))
	}

	// Idempotent: clearing again should not error
	if err := s.ClearHandoffHistory(ctx, "team-1", ""); err != nil {
		t.Errorf("expected idempotent clear, got error: %v", err)
	}
}

func TestClearLastHandoff(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	if err := s.SetLastHandoff(ctx, "team-1", "agent-1", "test content"); err != nil {
		t.Fatalf("SetLastHandoff: %v", err)
	}

	if err := s.ClearLastHandoff(ctx, "team-1", "agent-1"); err != nil {
		t.Fatalf("ClearLastHandoff: %v", err)
	}

	content, err := s.GetLastHandoff(ctx, "team-1", "agent-1")
	if err != nil {
		t.Fatalf("GetLastHandoff after clear: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty after clear, got: %s", content)
	}

	// Idempotent
	if err := s.ClearLastHandoff(ctx, "team-1", "agent-1"); err != nil {
		t.Errorf("expected idempotent clear, got error: %v", err)
	}
}

func TestGetTaskBoardEmpty(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	board, err := s.GetTaskBoard(ctx, "team-1")
	if err != nil {
		t.Fatalf("GetTaskBoard: %v", err)
	}
	if len(board.Tasks) != 0 {
		t.Errorf("expected empty board, got %d tasks", len(board.Tasks))
	}
}

func TestSaveAndGetTaskBoard(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	board := &TeamTaskBoard{
		Tasks: []TeamTask{
			{ID: "task-1", Title: "Test task", Status: "todo", Priority: "P2", CreatedBy: "agent-1"},
		},
	}
	if err := s.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatalf("SaveTaskBoard: %v", err)
	}

	got, err := s.GetTaskBoard(ctx, "team-1")
	if err != nil {
		t.Fatalf("GetTaskBoard: %v", err)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got.Tasks))
	}
	if got.Tasks[0].Title != "Test task" {
		t.Errorf("expected 'Test task', got: %s", got.Tasks[0].Title)
	}
}

func TestGetTask(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	board := &TeamTaskBoard{
		Tasks: []TeamTask{
			{ID: "task-1", Title: "First"},
			{ID: "task-2", Title: "Second"},
		},
	}
	if err := s.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	task, err := s.GetTask(ctx, "team-1", "task-2")
	if err != nil {
		t.Fatalf("GetTask: %v", err)
	}
	if task == nil {
		t.Fatal("expected task, got nil")
	}
	if task.Title != "Second" {
		t.Errorf("expected 'Second', got: %s", task.Title)
	}

	// Not found
	missing, err := s.GetTask(ctx, "team-1", "task-999")
	if err != nil {
		t.Fatalf("GetTask missing: %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for missing task, got: %v", missing)
	}
}

func TestUpdateTask(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	board := &TeamTaskBoard{
		Tasks: []TeamTask{
			{ID: "task-1", Title: "Original", Status: "todo"},
		},
	}
	if err := s.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	err := s.UpdateTask(ctx, "team-1", "task-1", func(task *TeamTask) {
		task.Status = "done"
		task.Title = "Updated"
	})
	if err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}

	task, err := s.GetTask(ctx, "team-1", "task-1")
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != "done" {
		t.Errorf("expected 'done', got: %s", task.Status)
	}
	if task.Title != "Updated" {
		t.Errorf("expected 'Updated', got: %s", task.Title)
	}
}

func TestDeleteTask(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	board := &TeamTaskBoard{
		Tasks: []TeamTask{
			{ID: "task-1", Title: "Keep"},
			{ID: "task-2", Title: "Delete"},
		},
	}
	if err := s.SaveTaskBoard(ctx, "team-1", board); err != nil {
		t.Fatal(err)
	}

	if err := s.DeleteTask(ctx, "team-1", "task-2"); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}

	got, err := s.GetTaskBoard(ctx, "team-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tasks) != 1 {
		t.Fatalf("expected 1 task after delete, got %d", len(got.Tasks))
	}
	if got.Tasks[0].ID != "task-1" {
		t.Errorf("expected task-1, got: %s", got.Tasks[0].ID)
	}
}

func TestUpdatePersistsRuntimeAndDecisionMode(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	updates := newIndependentTestTeam("ignored", "ignored")
	updates.Runtime.Mode = teamconfig.RuntimeModeSingleProcess
	updates.Coordination.Pattern = teamconfig.CoordinationPatternLeaderLed
	updates.Coordination.LeadAgentID = "lead"
	updates.Coordination.ReportingMode = teamconfig.ReportingModeLeader
	updates.Coordination.MessagingMode = teamconfig.MessagingModeInSession
	updates.Execution.QueuePolicy = teamconfig.QueuePolicySerialized
	updates.Execution.MaxConcurrentRuns = 1
	updates.DecisionMode = teamconfig.DecisionModeApproval
	if err := s.Update(ctx, "team-1", updates); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := s.Get(ctx, "team-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Runtime.Mode != teamconfig.RuntimeModeSingleProcess {
		t.Errorf("expected runtime.mode %q, got %q", teamconfig.RuntimeModeSingleProcess, got.Runtime.Mode)
	}
	if got.DecisionMode != teamconfig.DecisionModeApproval {
		t.Errorf("expected decisionMode %q, got %q", teamconfig.DecisionModeApproval, got.DecisionMode)
	}
}

func TestAppendAndGetDecisions(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	decisions := []DecisionEntry{
		{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Use JWT", Rationale: "Stateless", Context: "auth"},
		{ID: "dec-2", At: "2025-01-01T01:00:00Z", By: "agent-2", Decision: "Use Redis", Rationale: "Fast", Context: "cache"},
		{ID: "dec-3", At: "2025-01-01T02:00:00Z", By: "agent-1", Decision: "Add rate limit", Rationale: "Safety", Context: "auth"},
	}
	for i := range decisions {
		if err := s.AppendDecision(ctx, "team-1", &decisions[i]); err != nil {
			t.Fatalf("AppendDecision: %v", err)
		}
	}

	// Get all
	all, _, err := s.GetDecisions(ctx, "team-1", "", "", 0)
	if err != nil {
		t.Fatalf("GetDecisions all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3, got %d", len(all))
	}

	// Filter by context
	auth, _, err := s.GetDecisions(ctx, "team-1", "auth", "", 0)
	if err != nil {
		t.Fatalf("GetDecisions filtered: %v", err)
	}
	if len(auth) != 2 {
		t.Fatalf("expected 2 auth decisions, got %d", len(auth))
	}

	// Limit
	limited, _, err := s.GetDecisions(ctx, "team-1", "", "", 1)
	if err != nil {
		t.Fatalf("GetDecisions limited: %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("expected 1, got %d", len(limited))
	}
}

func TestUpdateDecision(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	entry := &DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Original", Rationale: "Because"}
	if err := s.AppendDecision(ctx, "team-1", entry); err != nil {
		t.Fatalf("AppendDecision: %v", err)
	}

	err := s.UpdateDecision(ctx, "team-1", "dec-1", func(d *DecisionEntry) {
		d.Decision = "Updated"
		d.Rationale = "New reason"
		d.Status = "accepted"
	})
	if err != nil {
		t.Fatalf("UpdateDecision: %v", err)
	}

	all, _, err := s.GetDecisions(ctx, "team-1", "", "", 0)
	if err != nil {
		t.Fatalf("GetDecisions: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1, got %d", len(all))
	}
	if all[0].Decision != "Updated" {
		t.Errorf("expected 'Updated', got: %s", all[0].Decision)
	}
	if all[0].Rationale != "New reason" {
		t.Errorf("expected 'New reason', got: %s", all[0].Rationale)
	}
	if all[0].Status != "accepted" {
		t.Errorf("expected 'accepted', got: %s", all[0].Status)
	}
}

func TestUpdateDecisionNotFound(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	entry := &DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "Test"}
	if err := s.AppendDecision(ctx, "team-1", entry); err != nil {
		t.Fatalf("AppendDecision: %v", err)
	}

	err := s.UpdateDecision(ctx, "team-1", "dec-999", func(d *DecisionEntry) {
		d.Decision = "Updated"
	})
	if err == nil {
		t.Fatal("expected error for non-existent decision")
	}
}

func TestDeleteDecision(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	entries := []DecisionEntry{
		{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Keep", Rationale: "Yes"},
		{ID: "dec-2", At: "2025-01-01T01:00:00Z", By: "agent-1", Decision: "Delete", Rationale: "No"},
	}
	for i := range entries {
		if err := s.AppendDecision(ctx, "team-1", &entries[i]); err != nil {
			t.Fatalf("AppendDecision: %v", err)
		}
	}

	if err := s.DeleteDecision(ctx, "team-1", "dec-2"); err != nil {
		t.Fatalf("DeleteDecision: %v", err)
	}

	all, _, err := s.GetDecisions(ctx, "team-1", "", "", 0)
	if err != nil {
		t.Fatalf("GetDecisions: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 after delete, got %d", len(all))
	}
	if all[0].ID != "dec-1" {
		t.Errorf("expected dec-1, got: %s", all[0].ID)
	}
}

func TestDeleteDecisionNotFound(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	entry := &DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Test", Rationale: "Test"}
	if err := s.AppendDecision(ctx, "team-1", entry); err != nil {
		t.Fatalf("AppendDecision: %v", err)
	}

	err := s.DeleteDecision(ctx, "team-1", "dec-999")
	if err == nil {
		t.Fatal("expected error for non-existent decision")
	}
}

func TestDeleteDecisionOnly(t *testing.T) {
	s := setupStateTestStore(t)
	ctx := context.Background()

	entry := &DecisionEntry{ID: "dec-1", At: "2025-01-01T00:00:00Z", By: "agent-1", Decision: "Only", Rationale: "One"}
	if err := s.AppendDecision(ctx, "team-1", entry); err != nil {
		t.Fatalf("AppendDecision: %v", err)
	}

	if err := s.DeleteDecision(ctx, "team-1", "dec-1"); err != nil {
		t.Fatalf("DeleteDecision: %v", err)
	}

	all, _, err := s.GetDecisions(ctx, "team-1", "", "", 0)
	if err != nil {
		t.Fatalf("GetDecisions: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 after deleting only decision, got %d", len(all))
	}
}
