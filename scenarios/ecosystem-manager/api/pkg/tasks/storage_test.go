package tasks

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveQueueItemRemovesDuplicates(t *testing.T) {
	tmp := t.TempDir()
	for _, status := range queueStatuses {
		if err := os.MkdirAll(filepath.Join(tmp, status), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	storage := NewStorage(tmp)

	// Seed an existing copy in the failed queue.
	duplicatePath := filepath.Join(tmp, "failed", "task-123.yaml")
	if err := os.WriteFile(duplicatePath, []byte("id: task-123\nstatus: failed\n"), 0o644); err != nil {
		t.Fatalf("seed duplicate: %v", err)
	}

	item := TaskItem{ID: "task-123", Status: "pending"}
	if err := storage.SaveQueueItem(item, "pending"); err != nil {
		t.Fatalf("SaveQueueItem: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "pending", "task-123.yaml")); err != nil {
		t.Fatalf("expected pending copy: %v", err)
	}

	if _, err := os.Stat(duplicatePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected failed copy removed, got err=%v", err)
	}
}

func TestCleanupDuplicatesKeepsNewestCopy(t *testing.T) {
	tmp := t.TempDir()
	for _, status := range queueStatuses {
		if err := os.MkdirAll(filepath.Join(tmp, status), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	storage := NewStorage(tmp)

	newerPath := filepath.Join(tmp, "pending", "task-abc.yaml")
	olderPath := filepath.Join(tmp, "failed", "task-abc.yaml")

	if err := os.WriteFile(olderPath, []byte("id: task-abc\nstatus: failed\n"), 0o644); err != nil {
		t.Fatalf("write older: %v", err)
	}
	// Set older timestamp.
	olderTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(olderPath, olderTime, olderTime); err != nil {
		t.Fatalf("chtimes older: %v", err)
	}

	if err := os.WriteFile(newerPath, []byte("id: task-abc\nstatus: pending\n"), 0o644); err != nil {
		t.Fatalf("write newer: %v", err)
	}

	if err := storage.CleanupDuplicates(); err != nil {
		t.Fatalf("CleanupDuplicates: %v", err)
	}

	if _, err := os.Stat(newerPath); err != nil {
		t.Fatalf("expected newer copy retained: %v", err)
	}

	if _, err := os.Stat(olderPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected older copy removed, err=%v", err)
	}
}

func TestMoveTaskTo(t *testing.T) {
	tmp := t.TempDir()
	for _, status := range queueStatuses {
		if err := os.MkdirAll(filepath.Join(tmp, status), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	storage := NewStorage(tmp)

	t.Run("SuccessfulMove", func(t *testing.T) {
		task := TaskItem{ID: "task-move-1", Type: "resource", Status: "pending"}
		if err := storage.SaveQueueItem(task, "pending"); err != nil {
			t.Fatalf("SaveQueueItem: %v", err)
		}

		movedTask, previousStatus, err := storage.MoveTaskTo("task-move-1", "in-progress")
		if err != nil {
			t.Fatalf("MoveTaskTo: %v", err)
		}

		if previousStatus != "pending" {
			t.Errorf("Expected previousStatus=pending, got %s", previousStatus)
		}

		if movedTask.ID != "task-move-1" {
			t.Errorf("Expected task ID task-move-1, got %s", movedTask.ID)
		}

		// Verify file moved
		if _, err := os.Stat(filepath.Join(tmp, "in-progress", "task-move-1.yaml")); err != nil {
			t.Fatalf("Expected task in in-progress: %v", err)
		}
		if _, err := os.Stat(filepath.Join(tmp, "pending", "task-move-1.yaml")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("Expected task removed from pending")
		}
	})

	t.Run("NoOpWhenAlreadyInStatus", func(t *testing.T) {
		task := TaskItem{ID: "task-noop", Type: "resource", Status: "completed"}
		if err := storage.SaveQueueItem(task, "completed"); err != nil {
			t.Fatalf("SaveQueueItem: %v", err)
		}

		_, previousStatus, err := storage.MoveTaskTo("task-noop", "completed")
		if err != nil {
			t.Fatalf("MoveTaskTo: %v", err)
		}

		if previousStatus != "completed" {
			t.Errorf("Expected noop to report current status")
		}
	})

	t.Run("ErrorWhenTaskNotFound", func(t *testing.T) {
		_, _, err := storage.MoveTaskTo("nonexistent-task", "completed")
		if err == nil {
			t.Fatal("Expected error for nonexistent task")
		}
	})
}

func TestGetTaskByID(t *testing.T) {
	tmp := t.TempDir()
	for _, status := range queueStatuses {
		if err := os.MkdirAll(filepath.Join(tmp, status), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	storage := NewStorage(tmp)

	t.Run("ExactFilenameMatch", func(t *testing.T) {
		task := TaskItem{ID: "exact-match", Type: "resource", Status: "pending"}
		if err := storage.SaveQueueItem(task, "pending"); err != nil {
			t.Fatalf("SaveQueueItem: %v", err)
		}

		found, status, err := storage.GetTaskByID("exact-match")
		if err != nil {
			t.Fatalf("GetTaskByID: %v", err)
		}

		if found.ID != "exact-match" {
			t.Errorf("Expected ID exact-match, got %s", found.ID)
		}
		if status != "pending" {
			t.Errorf("Expected status pending, got %s", status)
		}
	})

	t.Run("TimestampPrefixMatch", func(t *testing.T) {
		task := TaskItem{ID: "resource-generator-test-20250110-143000", Type: "resource", Status: "in-progress"}
		if err := storage.SaveQueueItem(task, "in-progress"); err != nil {
			t.Fatalf("SaveQueueItem: %v", err)
		}

		// Search using full ID
		found, status, err := storage.GetTaskByID("resource-generator-test-20250110-143000")
		if err != nil {
			t.Fatalf("GetTaskByID: %v", err)
		}

		if found.ID != "resource-generator-test-20250110-143000" {
			t.Errorf("Expected full ID, got %s", found.ID)
		}
		if status != "in-progress" {
			t.Errorf("Expected status in-progress, got %s", status)
		}
	})

	t.Run("InternalIDMatch", func(t *testing.T) {
		// Create a file with different filename than internal ID
		task := TaskItem{ID: "internal-id-123", Type: "scenario", Status: "completed"}
		if err := storage.SaveQueueItem(task, "completed"); err != nil {
			t.Fatalf("SaveQueueItem: %v", err)
		}

		// Rename file to simulate mismatch
		oldPath := filepath.Join(tmp, "completed", "internal-id-123.yaml")
		newPath := filepath.Join(tmp, "completed", "different-filename.yaml")
		if err := os.Rename(oldPath, newPath); err != nil {
			t.Fatalf("Rename: %v", err)
		}

		// Should still find by internal ID
		found, status, err := storage.GetTaskByID("internal-id-123")
		if err != nil {
			t.Fatalf("GetTaskByID: %v", err)
		}

		if found.ID != "internal-id-123" {
			t.Errorf("Expected ID internal-id-123, got %s", found.ID)
		}
		if status != "completed" {
			t.Errorf("Expected status completed, got %s", status)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, _, err := storage.GetTaskByID("does-not-exist")
		if err == nil {
			t.Fatal("Expected error for nonexistent task")
		}
	})
}

func TestNormalizeTaskItemDoesNotForceAutoRequeue(t *testing.T) {
	storage := NewStorage(t.TempDir())
	item := TaskItem{
		Title:                "Manual toggle respected",
		Status:               "completed",
		ProcessorAutoRequeue: false,
	}

	raw := map[string]any{
		"title": "Manual toggle respected",
		// Intentionally omit processor_auto_requeue to ensure we don't default it to true.
	}

	_ = storage.normalizeTaskItem(&item, "completed", raw)

	if item.ProcessorAutoRequeue {
		t.Fatalf("expected ProcessorAutoRequeue to remain false when omitted in raw data")
	}
}

func TestNormalizeTargets(t *testing.T) {
	tests := []struct {
		name          string
		primary       string
		targets       []string
		wantTargets   []string
		wantCanonical string
	}{
		{
			name:          "SinglePrimaryOnly",
			primary:       "redis",
			targets:       nil,
			wantTargets:   []string{"redis"},
			wantCanonical: "redis",
		},
		{
			name:          "ArrayOnly",
			primary:       "",
			targets:       []string{"postgres", "redis"},
			wantTargets:   []string{"postgres", "redis"},
			wantCanonical: "postgres",
		},
		{
			name:          "BothWithDuplicates",
			primary:       "redis",
			targets:       []string{"redis", "postgres"},
			wantTargets:   []string{"redis", "postgres"},
			wantCanonical: "redis",
		},
		{
			name:          "EmptyStringsFiltered",
			primary:       "",
			targets:       []string{"", "  ", "postgres", ""},
			wantTargets:   []string{"postgres"},
			wantCanonical: "postgres",
		},
		{
			name:          "CaseInsensitiveDuplication",
			primary:       "Redis",
			targets:       []string{"redis", "REDIS"},
			wantTargets:   []string{"Redis"},
			wantCanonical: "Redis",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTargets, gotCanonical := NormalizeTargets(tt.primary, tt.targets)

			if len(gotTargets) != len(tt.wantTargets) {
				t.Errorf("Target count mismatch: got %d, want %d", len(gotTargets), len(tt.wantTargets))
			}

			for i, want := range tt.wantTargets {
				if i >= len(gotTargets) || gotTargets[i] != want {
					t.Errorf("Target[%d] = %q, want %q", i, gotTargets[i], want)
				}
			}

			if gotCanonical != tt.wantCanonical {
				t.Errorf("Canonical = %q, want %q", gotCanonical, tt.wantCanonical)
			}
		})
	}
}

func TestFindActiveTargetTask(t *testing.T) {
	tmp := t.TempDir()
	for _, status := range queueStatuses {
		if err := os.MkdirAll(filepath.Join(tmp, status), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	storage := NewStorage(tmp)

	t.Run("FindsExistingActiveTask", func(t *testing.T) {
		task := TaskItem{
			ID:        "improver-redis-1",
			Type:      "resource",
			Operation: "improver",
			Target:    "redis",
			Targets:   []string{"redis"},
			Status:    "in-progress",
		}
		if err := storage.SaveQueueItem(task, "in-progress"); err != nil {
			t.Fatalf("SaveQueueItem: %v", err)
		}

		found, status, err := storage.FindActiveTargetTask("resource", "improver", "redis")
		if err != nil {
			t.Fatalf("FindActiveTargetTask: %v", err)
		}

		if found == nil {
			t.Fatal("Expected to find active task")
		}
		if found.ID != "improver-redis-1" {
			t.Errorf("Expected ID improver-redis-1, got %s", found.ID)
		}
		if status != "in-progress" {
			t.Errorf("Expected status in-progress, got %s", status)
		}
	})

	t.Run("ReturnsNilWhenNoMatch", func(t *testing.T) {
		found, _, err := storage.FindActiveTargetTask("resource", "improver", "nonexistent")
		if err != nil {
			t.Fatalf("FindActiveTargetTask: %v", err)
		}

		if found != nil {
			t.Errorf("Expected nil, got task %s", found.ID)
		}
	})

	t.Run("IgnoresCompletedTasks", func(t *testing.T) {
		completedTask := TaskItem{
			ID:        "improver-postgres-1",
			Type:      "resource",
			Operation: "improver",
			Target:    "postgres",
			Targets:   []string{"postgres"},
			Status:    "completed",
		}
		if err := storage.SaveQueueItem(completedTask, "completed"); err != nil {
			t.Fatalf("SaveQueueItem: %v", err)
		}

		found, _, err := storage.FindActiveTargetTask("resource", "improver", "postgres")
		if err != nil {
			t.Fatalf("FindActiveTargetTask: %v", err)
		}

		if found != nil {
			t.Errorf("Expected nil (completed tasks are not active), got %s", found.ID)
		}
	})
}

func TestMoveTaskDoesNotDeadlock(t *testing.T) {
	tmp := t.TempDir()
	for _, status := range queueStatuses {
		if err := os.MkdirAll(filepath.Join(tmp, status), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	storage := NewStorage(tmp)
	task := TaskItem{ID: "deadlock-test", Type: "scenario", Status: "pending"}
	if err := storage.SaveQueueItem(task, "pending"); err != nil {
		t.Fatalf("SaveQueueItem: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- storage.MoveTask(task.ID, "pending", "completed")
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("MoveTask returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("MoveTask timed out (possible deadlock)")
	}

	if _, status, err := storage.GetTaskByID(task.ID); err != nil || status != "completed" {
		t.Fatalf("expected task moved to completed, status=%s err=%v", status, err)
	}
}
