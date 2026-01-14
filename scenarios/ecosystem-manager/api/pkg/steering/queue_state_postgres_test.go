package steering

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
)

// Tests for PostgresQueueStateRepository nil handling

func TestPostgresQueueStateRepository_Get_NilDB(t *testing.T) {
	repo := NewPostgresQueueStateRepository(nil)

	_, err := repo.Get("task-1")
	if err == nil {
		t.Error("Get() should return error when database connection is nil")
	}
}

func TestPostgresQueueStateRepository_Save_NilDB(t *testing.T) {
	repo := NewPostgresQueueStateRepository(nil)

	state := NewQueueState("task-1", []autosteer.SteerMode{"progress"})
	err := repo.Save(state)
	if err == nil {
		t.Error("Save() should return error when database connection is nil")
	}
}

func TestPostgresQueueStateRepository_Save_NilState(t *testing.T) {
	repo := NewPostgresQueueStateRepository(nil)

	err := repo.Save(nil)
	if err == nil {
		t.Error("Save() should return error when state is nil")
	}
}

func TestPostgresQueueStateRepository_Delete_NilDB(t *testing.T) {
	repo := NewPostgresQueueStateRepository(nil)

	err := repo.Delete("task-1")
	if err == nil {
		t.Error("Delete() should return error when database connection is nil")
	}
}

// Tests for InMemoryQueueStateRepository

func TestInMemoryQueueStateRepository_Get_NotFound(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()

	state, err := repo.Get("nonexistent")
	if err != nil {
		t.Errorf("Get() error = %v, want nil for non-existent", err)
	}
	if state != nil {
		t.Error("Get() should return nil for non-existent task")
	}
}

func TestInMemoryQueueStateRepository_Get_Found(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()

	original := NewQueueState("task-1", []autosteer.SteerMode{"progress", "ux"})
	original.CurrentIndex = 1
	repo.Save(original)

	state, err := repo.Get("task-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if state == nil {
		t.Fatal("Get() returned nil for existing task")
	}
	if state.TaskID != "task-1" {
		t.Errorf("Get().TaskID = %v, want task-1", state.TaskID)
	}
	if len(state.Queue) != 2 {
		t.Errorf("Get().Queue length = %d, want 2", len(state.Queue))
	}
	if state.CurrentIndex != 1 {
		t.Errorf("Get().CurrentIndex = %d, want 1", state.CurrentIndex)
	}
}

func TestInMemoryQueueStateRepository_Get_ReturnsCopy(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()

	original := NewQueueState("task-1", []autosteer.SteerMode{"progress", "ux"})
	repo.Save(original)

	// Get the state and modify it
	state1, _ := repo.Get("task-1")
	state1.CurrentIndex = 99
	state1.Queue[0] = "modified"

	// Get again - should be original values
	state2, _ := repo.Get("task-1")
	if state2.CurrentIndex == 99 {
		t.Error("Get() should return a copy, not the original")
	}
	if state2.Queue[0] == "modified" {
		t.Error("Get() should return a copy with independent queue slice")
	}
}

func TestInMemoryQueueStateRepository_Save(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()

	state := NewQueueState("task-1", []autosteer.SteerMode{"progress"})
	err := repo.Save(state)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	retrieved, _ := repo.Get("task-1")
	if retrieved == nil {
		t.Fatal("Save() did not persist state")
	}
	if retrieved.TaskID != "task-1" {
		t.Errorf("Retrieved TaskID = %v, want task-1", retrieved.TaskID)
	}
}

func TestInMemoryQueueStateRepository_Save_NilState(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()

	err := repo.Save(nil)
	if err == nil {
		t.Error("Save() should return error for nil state")
	}
}

func TestInMemoryQueueStateRepository_Save_StoresCopy(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()

	state := NewQueueState("task-1", []autosteer.SteerMode{"progress", "ux"})
	repo.Save(state)

	// Modify original after save
	state.CurrentIndex = 99
	state.Queue[0] = "modified"

	// Retrieved should have original values
	retrieved, _ := repo.Get("task-1")
	if retrieved.CurrentIndex == 99 {
		t.Error("Save() should store a copy, not the original")
	}
	if retrieved.Queue[0] == "modified" {
		t.Error("Save() should store a copy with independent queue slice")
	}
}

func TestInMemoryQueueStateRepository_Save_Update(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()

	// Save initial state
	state1 := NewQueueState("task-1", []autosteer.SteerMode{"progress"})
	repo.Save(state1)

	// Update with new state
	state2 := NewQueueState("task-1", []autosteer.SteerMode{"ux", "test"})
	state2.CurrentIndex = 1
	repo.Save(state2)

	// Retrieved should have updated values
	retrieved, _ := repo.Get("task-1")
	if len(retrieved.Queue) != 2 {
		t.Errorf("Save() update: Queue length = %d, want 2", len(retrieved.Queue))
	}
	if retrieved.CurrentIndex != 1 {
		t.Errorf("Save() update: CurrentIndex = %d, want 1", retrieved.CurrentIndex)
	}
}

func TestInMemoryQueueStateRepository_Delete(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()

	state := NewQueueState("task-1", []autosteer.SteerMode{"progress"})
	repo.Save(state)

	err := repo.Delete("task-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	retrieved, _ := repo.Get("task-1")
	if retrieved != nil {
		t.Error("Delete() did not remove state")
	}
}

func TestInMemoryQueueStateRepository_Delete_NotFound(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()

	// Delete non-existent - should not error
	err := repo.Delete("nonexistent")
	if err != nil {
		t.Errorf("Delete() error = %v, want nil for non-existent task", err)
	}
}

// Note: QueueState helper methods (IsExhausted, CurrentMode, Remaining, Advance, Reset, Position)
// are tested in queue_provider_test.go to avoid duplication.
