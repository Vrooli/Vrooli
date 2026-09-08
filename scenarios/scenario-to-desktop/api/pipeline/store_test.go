package pipeline

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryStoreDelete(t *testing.T) {
	store := NewInMemoryStore()
	status := &Status{PipelineID: "test-id", Status: StatusRunning}
	store.Save(status)

	t.Run("delete existing", func(t *testing.T) {
		store.Delete("test-id")
		_, ok := store.Get("test-id")
		if ok {
			t.Error("expected status to be deleted")
		}
	})

	t.Run("delete nonexistent", func(t *testing.T) {
		// Should not panic
		store.Delete("nonexistent")
	})
}

func TestInMemoryStoreCleanup(t *testing.T) {
	store := NewInMemoryStore()
	now := time.Now()

	// Create statuses with different completion times
	oldCompleted := &Status{
		PipelineID:  "old-completed",
		Status:      StatusCompleted,
		CompletedAt: now.Add(-25 * time.Hour).Unix(),
	}
	recentCompleted := &Status{
		PipelineID:  "recent-completed",
		Status:      StatusCompleted,
		CompletedAt: now.Add(-1 * time.Hour).Unix(),
	}
	running := &Status{
		PipelineID: "running",
		Status:     StatusRunning,
	}
	oldFailed := &Status{
		PipelineID:  "old-failed",
		Status:      StatusFailed,
		CompletedAt: now.Add(-48 * time.Hour).Unix(),
	}

	store.Save(oldCompleted)
	store.Save(recentCompleted)
	store.Save(running)
	store.Save(oldFailed)

	// Cleanup statuses older than 24 hours (use unix timestamp)
	cutoffTime := now.Add(-24 * time.Hour).Unix()
	store.Cleanup(cutoffTime)

	// Old completed and old failed should be deleted (completed before cutoff)
	if _, ok := store.Get("old-completed"); ok {
		t.Error("expected old-completed to be cleaned up")
	}
	if _, ok := store.Get("old-failed"); ok {
		t.Error("expected old-failed to be cleaned up")
	}

	// Recent completed and running should remain
	if _, ok := store.Get("recent-completed"); !ok {
		t.Error("expected recent-completed to remain")
	}
	if _, ok := store.Get("running"); !ok {
		t.Error("expected running to remain")
	}
}

func TestInMemoryStoreUpdateStage(t *testing.T) {
	store := NewInMemoryStore()
	status := &Status{
		PipelineID: "test-id",
		Status:     StatusRunning,
		Stages:     make(map[string]*StageResult),
	}
	store.Save(status)

	t.Run("update existing stage", func(t *testing.T) {
		result := &StageResult{
			Stage:  "build",
			Status: StatusRunning,
		}
		store.UpdateStage("test-id", "build", result)

		updated, _ := store.Get("test-id")
		if updated.Stages["build"].Status != StatusRunning {
			t.Error("expected stage to be updated")
		}
	})

	t.Run("update nonexistent pipeline", func(t *testing.T) {
		result := &StageResult{Stage: "test", Status: StatusRunning}
		// Should not panic
		store.UpdateStage("nonexistent", "test", result)
	})
}

func TestInMemoryStoreReadsAreIndependentSnapshots(t *testing.T) {
	store := NewInMemoryStore()
	store.Save(&Status{
		PipelineID: "snapshot-test",
		Status:     StatusRunning,
		StageOrder: []string{"build"},
		Stages: map[string]*StageResult{
			"build": {Stage: "build", Status: StatusRunning, Logs: []string{"started"}},
		},
		Config: &Config{Platforms: []string{"linux/amd64"}},
	})

	read, ok := store.Get("snapshot-test")
	if !ok {
		t.Fatal("expected stored status")
	}
	read.Status = StatusFailed
	read.StageOrder[0] = "smoketest"
	read.Stages["build"].Status = StatusFailed
	read.Stages["build"].Logs[0] = "mutated"
	read.Config.Platforms[0] = "windows/amd64"

	stored, ok := store.Get("snapshot-test")
	if !ok {
		t.Fatal("expected stored status after read mutation")
	}
	if stored.Status != StatusRunning || stored.StageOrder[0] != "build" || stored.Stages["build"].Status != StatusRunning || stored.Stages["build"].Logs[0] != "started" || stored.Config.Platforms[0] != "linux/amd64" {
		t.Fatalf("read mutation leaked into store: %#v", stored)
	}
}

func TestCancelManagerOperations(t *testing.T) {
	cm := NewInMemoryCancelManager()

	t.Run("set and take", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		cm.Set("pipeline-1", cancel)

		cancelFn := cm.Take("pipeline-1")
		if cancelFn == nil {
			t.Error("expected Take to return non-nil for set pipeline")
		}

		// Should return nil on second take
		cancelFn2 := cm.Take("pipeline-1")
		if cancelFn2 != nil {
			t.Error("expected Take to return nil after already taken")
		}
	})

	t.Run("take nonexistent", func(t *testing.T) {
		cancelFn := cm.Take("nonexistent")
		if cancelFn != nil {
			t.Error("expected Take to return nil for nonexistent pipeline")
		}
	})

	t.Run("clear", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		cm.Set("pipeline-2", cancel)
		cm.Clear("pipeline-2")

		cancelFn := cm.Take("pipeline-2")
		if cancelFn != nil {
			t.Error("expected Take to return nil after Clear")
		}
	})
}

// Note: TestUUIDGenerator and TestRealTimeProvider are in orchestrator_test.go
