package pipeline

import (
	"context"
	"testing"
)

func TestInMemoryStore(t *testing.T) {
	store := NewInMemoryStore()

	t.Run("Save and Get", func(t *testing.T) {
		status := &Status{
			PipelineID:   "pipeline-123",
			ScenarioName: "test-scenario",
			Status:       StatusRunning,
		}
		store.Save(status)

		retrieved, ok := store.Get("pipeline-123")
		if !ok {
			t.Fatalf("expected to retrieve saved status")
		}
		if retrieved.PipelineID != "pipeline-123" {
			t.Errorf("expected pipeline ID 'pipeline-123'")
		}
	})

	t.Run("Get nonexistent", func(t *testing.T) {
		_, ok := store.Get("nonexistent")
		if ok {
			t.Errorf("expected false for nonexistent")
		}
	})

	t.Run("Update existing", func(t *testing.T) {
		status := &Status{
			PipelineID: "pipeline-update",
			Status:     StatusPending,
		}
		store.Save(status)

		updated := store.Update("pipeline-update", func(s *Status) {
			s.Status = StatusCompleted
		})
		if !updated {
			t.Errorf("expected Update to return true")
		}

		retrieved, _ := store.Get("pipeline-update")
		if retrieved.Status != StatusCompleted {
			t.Errorf("expected status 'completed', got %q", retrieved.Status)
		}
	})

	t.Run("Update nonexistent", func(t *testing.T) {
		updated := store.Update("nonexistent", func(s *Status) {
			s.Status = StatusCompleted
		})
		if updated {
			t.Errorf("expected Update to return false for nonexistent")
		}
	})

	t.Run("List", func(t *testing.T) {
		listStore := NewInMemoryStore()
		listStore.Save(&Status{PipelineID: "p1", Status: StatusRunning})
		listStore.Save(&Status{PipelineID: "p2", Status: StatusCompleted})

		all := listStore.List()
		if len(all) != 2 {
			t.Errorf("expected 2 statuses, got %d", len(all))
		}
	})
}

func TestInMemoryCancelManager(t *testing.T) {
	cm := NewInMemoryCancelManager()

	t.Run("Set and Take cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cm.Set("pipeline-1", cancel)

		taken := cm.Take("pipeline-1")
		if taken == nil {
			t.Fatalf("expected to take cancel func")
		}

		// Verify cancel works
		taken()
		select {
		case <-ctx.Done():
			// Expected
		default:
			t.Errorf("expected context to be cancelled")
		}
	})

	t.Run("Take twice", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		cm.Set("pipeline-2", cancel)

		taken := cm.Take("pipeline-2")
		if taken == nil {
			t.Fatalf("expected to take cancel func")
		}

		// Second take should return nil
		taken2 := cm.Take("pipeline-2")
		if taken2 != nil {
			t.Errorf("expected nil on second take")
		}
	})

	t.Run("Take nonexistent", func(t *testing.T) {
		taken := cm.Take("nonexistent")
		if taken != nil {
			t.Errorf("expected nil for nonexistent")
		}
	})

	t.Run("Clear", func(t *testing.T) {
		_, cancel := context.WithCancel(context.Background())
		cm.Set("pipeline-3", cancel)
		cm.Clear("pipeline-3")

		taken := cm.Take("pipeline-3")
		if taken != nil {
			t.Errorf("expected nil after clear")
		}
	})
}

func TestUUIDGenerator(t *testing.T) {
	gen := NewUUIDGenerator()

	id1 := gen.Generate()
	id2 := gen.Generate()

	if id1 == "" {
		t.Errorf("expected non-empty ID")
	}
	if id1 == id2 {
		t.Errorf("expected unique IDs")
	}
}

func TestRealTimeProvider(t *testing.T) {
	tp := NewRealTimeProvider()

	now := tp.Now()
	if now == 0 {
		t.Errorf("expected non-zero unix timestamp")
	}

	// Verify it returns a reasonable timestamp (after year 2020)
	if now < 1577836800 {
		t.Errorf("expected timestamp after 2020, got %d", now)
	}
}
