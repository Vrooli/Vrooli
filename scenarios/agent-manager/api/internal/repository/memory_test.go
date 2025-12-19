package repository_test

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// [REQ:REQ-P0-001] [REQ:REQ-P0-002] [REQ:REQ-P0-003] Repository tests for profile, task, and run management

func TestMemoryProfileRepository_CRUD(t *testing.T) {
	repo := repository.NewMemoryProfileRepository()
	ctx := context.Background()

	// Create
	profile := &domain.AgentProfile{
		ID:         uuid.New(),
		Name:       "test-profile",
		RunnerType: domain.RunnerTypeClaudeCode,
		Model:      "claude-3-opus",
		MaxTurns:   100,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := repo.Create(ctx, profile); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get by ID
	got, err := repo.Get(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected profile, got nil")
	}
	if got.Name != profile.Name {
		t.Errorf("expected name %s, got %s", profile.Name, got.Name)
	}

	// Get by Name
	got, err = repo.GetByName(ctx, "test-profile")
	if err != nil {
		t.Fatalf("GetByName failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected profile by name, got nil")
	}

	// Update
	profile.MaxTurns = 200
	if err := repo.Update(ctx, profile); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ = repo.Get(ctx, profile.ID)
	if got.MaxTurns != 200 {
		t.Errorf("expected MaxTurns 200, got %d", got.MaxTurns)
	}

	// List
	profiles, err := repo.List(ctx, repository.ListFilter{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(profiles))
	}

	// Delete
	if err := repo.Delete(ctx, profile.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ = repo.Get(ctx, profile.ID)
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestMemoryProfileRepository_DuplicateID(t *testing.T) {
	repo := repository.NewMemoryProfileRepository()
	ctx := context.Background()

	id := uuid.New()
	profile1 := &domain.AgentProfile{ID: id, Name: "profile1"}
	profile2 := &domain.AgentProfile{ID: id, Name: "profile2"}

	repo.Create(ctx, profile1)
	err := repo.Create(ctx, profile2)
	if err == nil {
		t.Error("expected error for duplicate ID")
	}
}

func TestMemoryProfileRepository_DuplicateName(t *testing.T) {
	repo := repository.NewMemoryProfileRepository()
	ctx := context.Background()

	profile1 := &domain.AgentProfile{ID: uuid.New(), Name: "same-name"}
	profile2 := &domain.AgentProfile{ID: uuid.New(), Name: "same-name"}

	repo.Create(ctx, profile1)
	err := repo.Create(ctx, profile2)
	if err == nil {
		t.Error("expected error for duplicate name")
	}
}

func TestMemoryTaskRepository_CRUD(t *testing.T) {
	repo := repository.NewMemoryTaskRepository()
	ctx := context.Background()

	// Create
	task := &domain.Task{
		ID:          uuid.New(),
		Title:       "Test Task",
		Description: "A test task",
		ScopePath:   "src/",
		Status:      domain.TaskStatusQueued,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := repo.Create(ctx, task); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get
	got, err := repo.Get(ctx, task.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil || got.Title != task.Title {
		t.Error("Get returned wrong task")
	}

	// List
	tasks, err := repo.List(ctx, repository.ListFilter{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// ListByStatus
	tasks, err = repo.ListByStatus(ctx, domain.TaskStatusQueued, repository.ListFilter{})
	if err != nil {
		t.Fatalf("ListByStatus failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("expected 1 queued task, got %d", len(tasks))
	}

	// Update
	task.Status = domain.TaskStatusRunning
	if err := repo.Update(ctx, task); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Delete
	if err := repo.Delete(ctx, task.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestMemoryRunRepository_CRUD(t *testing.T) {
	repo := repository.NewMemoryRunRepository()
	ctx := context.Background()

	taskID := uuid.New()
	profileID := uuid.New()

	// Create
	run := &domain.Run{
		ID:             uuid.New(),
		TaskID:         taskID,
		AgentProfileID: &profileID,
		Status:         domain.RunStatusPending,
		RunMode:        domain.RunModeSandboxed,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := repo.Create(ctx, run); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get
	got, err := repo.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil || got.TaskID != taskID {
		t.Error("Get returned wrong run")
	}

	// List
	runs, err := repo.List(ctx, repository.RunListFilter{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("expected 1 run, got %d", len(runs))
	}

	// ListByTask
	runs, err = repo.ListByTask(ctx, taskID, repository.ListFilter{})
	if err != nil {
		t.Fatalf("ListByTask failed: %v", err)
	}
	if len(runs) != 1 {
		t.Errorf("expected 1 run for task, got %d", len(runs))
	}

	// CountByStatus
	count, err := repo.CountByStatus(ctx, domain.RunStatusPending)
	if err != nil {
		t.Fatalf("CountByStatus failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	// Update
	run.Status = domain.RunStatusRunning
	if err := repo.Update(ctx, run); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Delete
	if err := repo.Delete(ctx, run.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestMemoryRunRepository_FilterByTaskID(t *testing.T) {
	repo := repository.NewMemoryRunRepository()
	ctx := context.Background()

	taskID1 := uuid.New()
	taskID2 := uuid.New()

	// Create runs for different tasks
	run1 := &domain.Run{ID: uuid.New(), TaskID: taskID1, Status: domain.RunStatusPending}
	run2 := &domain.Run{ID: uuid.New(), TaskID: taskID1, Status: domain.RunStatusPending}
	run3 := &domain.Run{ID: uuid.New(), TaskID: taskID2, Status: domain.RunStatusPending}

	repo.Create(ctx, run1)
	repo.Create(ctx, run2)
	repo.Create(ctx, run3)

	// Filter by task
	runs, _ := repo.List(ctx, repository.RunListFilter{TaskID: &taskID1})
	if len(runs) != 2 {
		t.Errorf("expected 2 runs for task1, got %d", len(runs))
	}
}

func TestMemoryEventRepository_CRUD(t *testing.T) {
	repo := repository.NewMemoryEventRepository()
	ctx := context.Background()
	runID := uuid.New()

	// Append
	events := []*domain.RunEvent{
		{ID: uuid.New(), EventType: domain.EventTypeLog},
		{ID: uuid.New(), EventType: domain.EventTypeMessage},
	}

	if err := repo.Append(ctx, runID, events...); err != nil {
		t.Fatalf("Append failed: %v", err)
	}

	// Get
	got, err := repo.Get(ctx, runID, -1, 10)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 events, got %d", len(got))
	}

	// Verify sequences
	if got[0].Sequence != 0 || got[1].Sequence != 1 {
		t.Error("sequences not assigned correctly")
	}

	// GetByType
	got, err = repo.GetByType(ctx, runID, []domain.RunEventType{domain.EventTypeLog}, 10)
	if err != nil {
		t.Fatalf("GetByType failed: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 log event, got %d", len(got))
	}

	// Count
	count, err := repo.Count(ctx, runID)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// Delete
	if err := repo.Delete(ctx, runID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	count, _ = repo.Count(ctx, runID)
	if count != 0 {
		t.Errorf("expected count 0 after delete, got %d", count)
	}
}

func TestMemoryCheckpointRepository_CRUD(t *testing.T) {
	repo := repository.NewMemoryCheckpointRepository()
	ctx := context.Background()
	runID := uuid.New()

	// Save
	checkpoint := &domain.RunCheckpoint{
		RunID:           runID,
		Phase:           domain.RunPhaseExecuting,
		StepWithinPhase: 5,
		LastHeartbeat:   time.Now(),
	}

	if err := repo.Save(ctx, checkpoint); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Get
	got, err := repo.Get(ctx, runID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil || got.StepWithinPhase != 5 {
		t.Error("Get returned wrong checkpoint")
	}

	// Heartbeat
	oldTime := got.LastHeartbeat
	time.Sleep(10 * time.Millisecond)
	if err := repo.Heartbeat(ctx, runID); err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	got, _ = repo.Get(ctx, runID)
	if !got.LastHeartbeat.After(oldTime) {
		t.Error("heartbeat should have updated timestamp")
	}

	// ListStale
	stale, err := repo.ListStale(ctx, -1*time.Second) // Everything is stale
	if err != nil {
		t.Fatalf("ListStale failed: %v", err)
	}
	if len(stale) != 1 {
		t.Errorf("expected 1 stale checkpoint, got %d", len(stale))
	}

	// Delete
	if err := repo.Delete(ctx, runID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestMemoryIdempotencyRepository_CRUD(t *testing.T) {
	repo := repository.NewMemoryIdempotencyRepository()
	ctx := context.Background()
	key := "test-key-" + uuid.New().String()

	// Check non-existent
	got, err := repo.Check(ctx, key)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent key")
	}

	// Reserve
	rec, err := repo.Reserve(ctx, key, time.Hour)
	if err != nil {
		t.Fatalf("Reserve failed: %v", err)
	}
	if rec == nil || rec.Status != domain.IdempotencyStatusPending {
		t.Error("Reserve should return pending record")
	}

	// Check reserved
	got, _ = repo.Check(ctx, key)
	if got == nil {
		t.Error("expected record after reserve")
	}

	// Complete
	entityID := uuid.New()
	if err := repo.Complete(ctx, key, entityID, "run", []byte(`{"id":"test"}`)); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	got, _ = repo.Check(ctx, key)
	if got.Status != domain.IdempotencyStatusComplete {
		t.Errorf("expected complete status, got %s", got.Status)
	}
	if got.EntityID == nil || *got.EntityID != entityID {
		t.Error("entity ID not set correctly")
	}
}

func TestMemoryIdempotencyRepository_DuplicateReserve(t *testing.T) {
	repo := repository.NewMemoryIdempotencyRepository()
	ctx := context.Background()
	key := "duplicate-key"

	repo.Reserve(ctx, key, time.Hour)
	_, err := repo.Reserve(ctx, key, time.Hour)
	if err == nil {
		t.Error("expected error for duplicate reserve")
	}
}

func TestMemoryIdempotencyRepository_ExpiredReserve(t *testing.T) {
	repo := repository.NewMemoryIdempotencyRepository()
	ctx := context.Background()
	key := "expired-key"

	// Reserve with very short TTL
	repo.Reserve(ctx, key, 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	// Should be able to reserve again after expiration
	_, err := repo.Reserve(ctx, key, time.Hour)
	if err != nil {
		t.Errorf("should be able to reserve expired key: %v", err)
	}
}

func TestMemoryListFilter_Pagination(t *testing.T) {
	repo := repository.NewMemoryProfileRepository()
	ctx := context.Background()

	// Create 10 profiles
	for i := 0; i < 10; i++ {
		profile := &domain.AgentProfile{ID: uuid.New(), Name: "profile-" + uuid.New().String()}
		repo.Create(ctx, profile)
	}

	// Test offset
	profiles, _ := repo.List(ctx, repository.ListFilter{Offset: 5})
	if len(profiles) != 5 {
		t.Errorf("expected 5 profiles with offset 5, got %d", len(profiles))
	}

	// Test limit
	profiles, _ = repo.List(ctx, repository.ListFilter{Limit: 3})
	if len(profiles) != 3 {
		t.Errorf("expected 3 profiles with limit 3, got %d", len(profiles))
	}

	// Test offset + limit
	profiles, _ = repo.List(ctx, repository.ListFilter{Offset: 2, Limit: 3})
	if len(profiles) != 3 {
		t.Errorf("expected 3 profiles with offset 2 and limit 3, got %d", len(profiles))
	}

	// Test offset beyond range
	profiles, _ = repo.List(ctx, repository.ListFilter{Offset: 100})
	if len(profiles) != 0 {
		t.Errorf("expected 0 profiles with offset beyond range, got %d", len(profiles))
	}
}
