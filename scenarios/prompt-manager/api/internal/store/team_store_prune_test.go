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
	return NewFileTeamStore(storeDir, storeDir, nil), sharedDir
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
	return NewFileTeamStore(storeDir, storeDir, nil), sharedDir
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

func TestPruneSharedState_EmptyState(t *testing.T) {
	s, _ := setupPruneTestStore(t)

	result, err := s.PruneSharedState(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("PruneSharedState: %v", err)
	}
	if result.TasksRemoved != 0 {
		t.Errorf("expected zero tasks removed, got %d", result.TasksRemoved)
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
