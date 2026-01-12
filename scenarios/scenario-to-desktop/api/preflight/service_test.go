package preflight

import (
	"testing"
	"time"
)

// mockTimeProvider provides a fixed time for deterministic testing.
type mockTimeProvider struct {
	now time.Time
}

func (m *mockTimeProvider) Now() time.Time {
	return m.now
}

func TestInMemorySessionStore(t *testing.T) {
	store := NewInMemorySessionStore()

	t.Run("GetNonExistent", func(t *testing.T) {
		session, ok := store.Get("nonexistent")
		if ok || session != nil {
			t.Errorf("expected nil session for nonexistent ID")
		}
	})

	t.Run("StopNonExistent", func(t *testing.T) {
		if store.Stop("nonexistent") {
			t.Errorf("expected Stop to return false for nonexistent session")
		}
	})
}

func TestInMemoryJobStore(t *testing.T) {
	store := NewInMemoryJobStore()

	t.Run("CreateAndGet", func(t *testing.T) {
		job := store.Create()
		if job == nil || job.ID == "" {
			t.Fatalf("expected job to be created with ID")
		}

		retrieved, ok := store.Get(job.ID)
		if !ok {
			t.Fatalf("expected to retrieve created job")
		}
		if retrieved.ID != job.ID {
			t.Errorf("expected job IDs to match")
		}
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		job, ok := store.Get("nonexistent")
		if ok || job != nil {
			t.Errorf("expected nil job for nonexistent ID")
		}
	})

	t.Run("UpdateExisting", func(t *testing.T) {
		job := store.Create()
		store.Update(job.ID, func(j *Job) {
			j.Status = "running"
		})

		updated, ok := store.Get(job.ID)
		if !ok {
			t.Fatalf("expected to retrieve updated job")
		}
		if updated.Status != "running" {
			t.Errorf("expected status 'running', got %q", updated.Status)
		}
	})

	t.Run("SetStep", func(t *testing.T) {
		job := store.Create()
		store.SetStep(job.ID, "step1", "running", "executing")

		updated, _ := store.Get(job.ID)
		step, ok := updated.Steps["step1"]
		if !ok {
			t.Fatalf("expected step1 to exist")
		}
		if step.State != "running" {
			t.Errorf("expected state 'running', got %q", step.State)
		}
	})

	t.Run("Finish", func(t *testing.T) {
		job := store.Create()
		store.Finish(job.ID, "completed", "")

		finished, _ := store.Get(job.ID)
		if finished.Status != "completed" {
			t.Errorf("expected status 'completed', got %q", finished.Status)
		}
		// UpdatedAt should be set when job is finished
		if finished.UpdatedAt.IsZero() {
			t.Errorf("expected UpdatedAt to be set")
		}
	})
}

func TestServiceCreation(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatalf("expected service to be created")
	}
}
