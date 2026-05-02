package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupPruneTestStore(t *testing.T) (*FileTeamStore, string) {
	t.Helper()
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "store")
	teamDir := filepath.Join(storeDir, "teams", "team-1")
	sharedDir := filepath.Join(teamDir, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	team := newIndependentTestTeam("team-1", "Test")
	data, err := json.MarshalIndent(team, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(teamDir, "team.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return NewFileTeamStore(storeDir, nil), sharedDir
}

func setupPruneTestStoreWithRetention(t *testing.T, retention *RetentionConfig) (*FileTeamStore, string) {
	t.Helper()
	dir := t.TempDir()
	storeDir := filepath.Join(dir, "store")
	teamDir := filepath.Join(storeDir, "teams", "team-1")
	sharedDir := filepath.Join(teamDir, "shared")
	if err := os.MkdirAll(sharedDir, 0o755); err != nil {
		t.Fatal(err)
	}

	team := *newIndependentTestTeam("team-1", "Test")
	team.Retention = retention
	data, err := json.MarshalIndent(team, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(teamDir, "team.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return NewFileTeamStore(storeDir, nil), sharedDir
}

func writeTasks(t *testing.T, sharedDir string, tasks []TeamTask) {
	t.Helper()
	board := TeamTaskBoard{Tasks: tasks}
	data, err := json.Marshal(board)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sharedDir, "tasks.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeDecisions(t *testing.T, sharedDir string, entries []DecisionEntry) {
	t.Helper()
	var content string
	for _, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		content += string(data) + "\n"
	}
	if err := os.WriteFile(filepath.Join(sharedDir, "decisions.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeKnowledge(t *testing.T, sharedDir string, entries []KnowledgeEntry) {
	t.Helper()
	var content string
	for _, e := range entries {
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		content += string(data) + "\n"
	}
	if err := os.WriteFile(filepath.Join(sharedDir, "knowledge.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTasksBack(t *testing.T, s *FileTeamStore) []TeamTask {
	t.Helper()
	board, err := s.GetTaskBoard(context.Background(), "team-1")
	if err != nil {
		t.Fatal(err)
	}
	return board.Tasks
}

func TestPruneSharedState_CompletedTasks(t *testing.T) {
	s, sharedDir := setupPruneTestStoreWithRetention(t, &RetentionConfig{
		Tasks: &TaskRetention{MaxCompleted: 20, MaxAgeDays: 0},
	})

	now := time.Now().UTC()
	var tasks []TeamTask
	for i := 0; i < 30; i++ {
		tasks = append(tasks, TeamTask{
			ID:        fmt.Sprintf("task-%d", i),
			Title:     fmt.Sprintf("Task %d", i),
			Status:    "done",
			UpdatedAt: now.Add(-time.Duration(30-i) * time.Hour).Format(time.RFC3339),
		})
	}
	writeTasks(t, sharedDir, tasks)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}
	if result.TasksRemoved != 10 {
		t.Errorf("expected 10 tasks removed, got %d", result.TasksRemoved)
	}

	remaining := readTasksBack(t, s)
	if len(remaining) != 20 {
		t.Errorf("expected 20 tasks remaining, got %d", len(remaining))
	}
}

func TestPruneSharedState_TaskAgeCutoff(t *testing.T) {
	s, sharedDir := setupPruneTestStoreWithRetention(t, &RetentionConfig{
		Tasks: &TaskRetention{MaxCompleted: 0, MaxAgeDays: 7},
	})

	now := time.Now().UTC()
	tasks := []TeamTask{
		{ID: "old-1", Title: "Old", Status: "done", UpdatedAt: now.Add(-10 * 24 * time.Hour).Format(time.RFC3339)},
		{ID: "old-2", Title: "Old2", Status: "done", UpdatedAt: now.Add(-8 * 24 * time.Hour).Format(time.RFC3339)},
		{ID: "new-1", Title: "New", Status: "done", UpdatedAt: now.Add(-1 * 24 * time.Hour).Format(time.RFC3339)},
	}
	writeTasks(t, sharedDir, tasks)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}
	if result.TasksRemoved != 2 {
		t.Errorf("expected 2 tasks removed, got %d", result.TasksRemoved)
	}

	remaining := readTasksBack(t, s)
	if len(remaining) != 1 {
		t.Errorf("expected 1 task remaining, got %d", len(remaining))
	}
	if remaining[0].ID != "new-1" {
		t.Errorf("expected new-1 to remain, got %s", remaining[0].ID)
	}
}

func TestPruneSharedState_ActiveTasksUntouched(t *testing.T) {
	s, sharedDir := setupPruneTestStoreWithRetention(t, &RetentionConfig{
		Tasks: &TaskRetention{MaxCompleted: 1, MaxAgeDays: 1},
	})

	now := time.Now().UTC()
	tasks := []TeamTask{
		{ID: "active-1", Title: "Active", Status: "todo", UpdatedAt: now.Add(-100 * 24 * time.Hour).Format(time.RFC3339)},
		{ID: "active-2", Title: "InProgress", Status: "in-progress", UpdatedAt: now.Add(-100 * 24 * time.Hour).Format(time.RFC3339)},
		{ID: "active-3", Title: "Blocked", Status: "blocked", UpdatedAt: now.Add(-100 * 24 * time.Hour).Format(time.RFC3339)},
		{ID: "done-1", Title: "Done", Status: "done", UpdatedAt: now.Format(time.RFC3339)},
		{ID: "done-old", Title: "DoneOld", Status: "done", UpdatedAt: now.Add(-100 * 24 * time.Hour).Format(time.RFC3339)},
	}
	writeTasks(t, sharedDir, tasks)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}
	// done-old should be removed (too old + over maxCompleted)
	if result.TasksRemoved != 1 {
		t.Errorf("expected 1 task removed, got %d", result.TasksRemoved)
	}

	remaining := readTasksBack(t, s)
	// 3 active + 1 done = 4
	if len(remaining) != 4 {
		t.Errorf("expected 4 tasks remaining, got %d", len(remaining))
	}
	// Verify all active tasks survive
	activeCount := 0
	for _, task := range remaining {
		if task.Status != "done" {
			activeCount++
		}
	}
	if activeCount != 3 {
		t.Errorf("expected 3 active tasks, got %d", activeCount)
	}
}

func TestPruneSharedState_Decisions(t *testing.T) {
	s, sharedDir := setupPruneTestStoreWithRetention(t, &RetentionConfig{
		Decisions: &EntryRetention{MaxEntries: 50, MaxAgeDays: 0},
	})

	now := time.Now().UTC()
	var entries []DecisionEntry
	for i := 0; i < 80; i++ {
		entries = append(entries, DecisionEntry{
			ID:       fmt.Sprintf("dec-%d", i),
			At:       now.Add(-time.Duration(80-i) * time.Hour).Format(time.RFC3339),
			By:       "agent-1",
			Decision: fmt.Sprintf("Decision %d", i),
		})
	}
	writeDecisions(t, sharedDir, entries)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}
	if result.DecisionsRemoved != 30 {
		t.Errorf("expected 30 decisions removed, got %d", result.DecisionsRemoved)
	}

	// Read back
	remaining, _, err := s.GetDecisions(context.Background(), "team-1", "", "", 0)
	if err != nil {
		t.Fatalf("GetDecisions: %v", err)
	}
	if len(remaining) != 50 {
		t.Errorf("expected 50 decisions remaining, got %d", len(remaining))
	}
	// Verify newest are kept
	if remaining[0].ID != "dec-30" {
		t.Errorf("expected first remaining to be dec-30, got %s", remaining[0].ID)
	}
}

func TestPruneSharedState_DecisionAgeCutoff(t *testing.T) {
	s, sharedDir := setupPruneTestStoreWithRetention(t, &RetentionConfig{
		Decisions: &EntryRetention{MaxEntries: 0, MaxAgeDays: 30},
	})

	now := time.Now().UTC()
	entries := []DecisionEntry{
		{ID: "old", At: now.Add(-60 * 24 * time.Hour).Format(time.RFC3339), By: "a", Decision: "old"},
		{ID: "new", At: now.Add(-1 * 24 * time.Hour).Format(time.RFC3339), By: "a", Decision: "new"},
	}
	writeDecisions(t, sharedDir, entries)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}
	if result.DecisionsRemoved != 1 {
		t.Errorf("expected 1 decision removed, got %d", result.DecisionsRemoved)
	}
}

func TestPruneSharedState_Knowledge(t *testing.T) {
	s, sharedDir := setupPruneTestStoreWithRetention(t, &RetentionConfig{
		Knowledge: &EntryRetention{MaxEntries: 50, MaxAgeDays: 0},
	})

	now := time.Now().UTC()
	var entries []KnowledgeEntry
	for i := 0; i < 80; i++ {
		entries = append(entries, KnowledgeEntry{
			ID:      fmt.Sprintf("know-%d", i),
			At:      now.Add(-time.Duration(80-i) * time.Hour).Format(time.RFC3339),
			By:      "agent-1",
			Topic:   "test",
			Content: fmt.Sprintf("Knowledge %d", i),
		})
	}
	writeKnowledge(t, sharedDir, entries)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}
	if result.KnowledgeRemoved != 30 {
		t.Errorf("expected 30 knowledge removed, got %d", result.KnowledgeRemoved)
	}

	remaining, err := s.GetKnowledge(context.Background(), "team-1", "", "", 0)
	if err != nil {
		t.Fatalf("GetKnowledge: %v", err)
	}
	if len(remaining) != 50 {
		t.Errorf("expected 50 knowledge remaining, got %d", len(remaining))
	}
}

func TestPruneSharedState_EmptyState(t *testing.T) {
	s, _ := setupPruneTestStore(t)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}
	if result.TasksRemoved != 0 || result.DecisionsRemoved != 0 || result.KnowledgeRemoved != 0 {
		t.Errorf("expected all zeros, got tasks=%d decisions=%d knowledge=%d",
			result.TasksRemoved, result.DecisionsRemoved, result.KnowledgeRemoved)
	}
}

func TestPruneSharedState_NilRetention(t *testing.T) {
	// Use default store (no retention set on team) - should use defaults
	s, sharedDir := setupPruneTestStore(t)

	now := time.Now().UTC()
	// Write 30 done tasks - defaults keep 20
	var tasks []TeamTask
	for i := 0; i < 30; i++ {
		tasks = append(tasks, TeamTask{
			ID:        fmt.Sprintf("task-%d", i),
			Title:     fmt.Sprintf("Task %d", i),
			Status:    "done",
			UpdatedAt: now.Add(-time.Duration(i) * time.Hour).Format(time.RFC3339),
		})
	}
	writeTasks(t, sharedDir, tasks)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}
	if result.TasksRemoved != 10 {
		t.Errorf("expected 10 tasks removed with default retention, got %d", result.TasksRemoved)
	}
}

func TestPruneSharedState_CustomRetention(t *testing.T) {
	s, sharedDir := setupPruneTestStoreWithRetention(t, &RetentionConfig{
		Tasks:     &TaskRetention{MaxCompleted: 5, MaxAgeDays: 0},
		Decisions: &EntryRetention{MaxEntries: 10, MaxAgeDays: 0},
		Knowledge: &EntryRetention{MaxEntries: 3, MaxAgeDays: 0},
	})

	now := time.Now().UTC()

	// 15 done tasks
	var tasks []TeamTask
	for i := 0; i < 15; i++ {
		tasks = append(tasks, TeamTask{
			ID:        fmt.Sprintf("task-%d", i),
			Status:    "done",
			UpdatedAt: now.Add(-time.Duration(15-i) * time.Hour).Format(time.RFC3339),
		})
	}
	writeTasks(t, sharedDir, tasks)

	// 20 decisions
	var decisions []DecisionEntry
	for i := 0; i < 20; i++ {
		decisions = append(decisions, DecisionEntry{
			ID: fmt.Sprintf("dec-%d", i),
			At: now.Add(-time.Duration(20-i) * time.Hour).Format(time.RFC3339),
		})
	}
	writeDecisions(t, sharedDir, decisions)

	// 10 knowledge
	var knowledge []KnowledgeEntry
	for i := 0; i < 10; i++ {
		knowledge = append(knowledge, KnowledgeEntry{
			ID: fmt.Sprintf("know-%d", i),
			At: now.Add(-time.Duration(10-i) * time.Hour).Format(time.RFC3339),
		})
	}
	writeKnowledge(t, sharedDir, knowledge)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}
	if result.TasksRemoved != 10 {
		t.Errorf("expected 10 tasks removed, got %d", result.TasksRemoved)
	}
	if result.DecisionsRemoved != 10 {
		t.Errorf("expected 10 decisions removed, got %d", result.DecisionsRemoved)
	}
	if result.KnowledgeRemoved != 7 {
		t.Errorf("expected 7 knowledge removed, got %d", result.KnowledgeRemoved)
	}
}

func TestPruneSharedState_BothLimitsApplied(t *testing.T) {
	s, sharedDir := setupPruneTestStoreWithRetention(t, &RetentionConfig{
		Decisions: &EntryRetention{MaxEntries: 10, MaxAgeDays: 7},
	})

	now := time.Now().UTC()

	// 15 decisions total:
	//   5 very old (20-24 days ago)
	//   4 moderately old (8-11 days ago, beyond 7-day cutoff)
	//   6 recent (0-3 days ago, within 7-day cutoff)
	var entries []DecisionEntry
	for i := 0; i < 5; i++ {
		entries = append(entries, DecisionEntry{
			ID: fmt.Sprintf("ancient-%d", i),
			At: now.Add(-time.Duration(20+i) * 24 * time.Hour).Format(time.RFC3339),
		})
	}
	for i := 0; i < 4; i++ {
		entries = append(entries, DecisionEntry{
			ID: fmt.Sprintf("old-%d", i),
			At: now.Add(-time.Duration(8+i) * 24 * time.Hour).Format(time.RFC3339),
		})
	}
	for i := 0; i < 6; i++ {
		entries = append(entries, DecisionEntry{
			ID: fmt.Sprintf("new-%d", i),
			At: now.Add(-time.Duration(i) * 24 * time.Hour).Format(time.RFC3339),
		})
	}
	writeDecisions(t, sharedDir, entries)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}

	// maxEntries=10 first removes the 5 oldest (ancient-0 through ancient-4).
	// Remaining 10: old-0..old-3 (8-11 days), new-0..new-5 (0-3 days, well within cutoff).
	// maxAgeDays=7 then removes old-0..old-3 (all >7 days old) = 4 more.
	// Total removed: 5 + 4 = 9. Remaining: 6 recent entries.
	if result.DecisionsRemoved != 9 {
		t.Errorf("expected 9 decisions removed, got %d", result.DecisionsRemoved)
	}

	remaining, _, err := s.GetDecisions(context.Background(), "team-1", "", "", 0)
	if err != nil {
		t.Fatalf("GetDecisions: %v", err)
	}
	if len(remaining) != 6 {
		t.Errorf("expected 6 decisions remaining, got %d", len(remaining))
	}
}
