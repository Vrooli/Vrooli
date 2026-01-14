package steering

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestQueueProvider_Strategy(t *testing.T) {
	provider := NewQueueProvider(nil, nil)
	if provider.Strategy() != StrategyQueue {
		t.Errorf("Strategy() = %v, want %v", provider.Strategy(), StrategyQueue)
	}
}

func TestQueueProvider_Initialize(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)

	task := &tasks.TaskItem{
		ID:            "task-1",
		SteeringQueue: []string{"progress", "ux", "test"},
	}

	err := provider.Initialize(task)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Verify state was created
	state, err := repo.Get("task-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if state == nil {
		t.Fatal("State should exist after initialization")
	}
	if len(state.Queue) != 3 {
		t.Errorf("Queue length = %d, want 3", len(state.Queue))
	}
	if state.CurrentIndex != 0 {
		t.Errorf("CurrentIndex = %d, want 0", state.CurrentIndex)
	}
}

func TestQueueProvider_Initialize_AlreadyExists(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)

	// Pre-populate state
	existingState := NewQueueState("task-1", []autosteer.SteerMode{"refactor"})
	existingState.CurrentIndex = 0
	repo.Save(existingState)

	task := &tasks.TaskItem{
		ID:            "task-1",
		SteeringQueue: []string{"progress", "ux", "test"},
	}

	// Initialize should not overwrite
	err := provider.Initialize(task)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	state, _ := repo.Get("task-1")
	if len(state.Queue) != 1 {
		t.Errorf("Queue should not be overwritten, got length %d", len(state.Queue))
	}
}

func TestQueueProvider_Initialize_EmptyQueue(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)

	task := &tasks.TaskItem{
		ID:            "task-1",
		SteeringQueue: []string{},
	}

	err := provider.Initialize(task)
	if err == nil {
		t.Error("Initialize() should fail for empty queue")
	}
}

func TestQueueProvider_GetCurrentMode(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)

	state := NewQueueState("task-1", []autosteer.SteerMode{"progress", "ux", "test"})
	repo.Save(state)

	mode, err := provider.GetCurrentMode("task-1")
	if err != nil {
		t.Fatalf("GetCurrentMode() error = %v", err)
	}
	if mode != "progress" {
		t.Errorf("GetCurrentMode() = %v, want progress", mode)
	}
}

func TestQueueProvider_GetCurrentMode_NoState(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)

	mode, err := provider.GetCurrentMode("nonexistent")
	if err != nil {
		t.Fatalf("GetCurrentMode() error = %v", err)
	}
	if mode != "" {
		t.Errorf("GetCurrentMode() = %v, want empty string", mode)
	}
}

func TestQueueProvider_AfterExecution_Advance(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)

	state := NewQueueState("task-1", []autosteer.SteerMode{"progress", "ux", "test"})
	repo.Save(state)

	task := &tasks.TaskItem{ID: "task-1"}

	decision, err := provider.AfterExecution(task, "test-scenario")
	if err != nil {
		t.Fatalf("AfterExecution() error = %v", err)
	}

	if decision.ShouldRequeue != true {
		t.Errorf("ShouldRequeue = %v, want true", decision.ShouldRequeue)
	}
	if decision.Exhausted != false {
		t.Errorf("Exhausted = %v, want false", decision.Exhausted)
	}
	if decision.Mode != "ux" {
		t.Errorf("Mode = %v, want ux (next mode)", decision.Mode)
	}

	// Verify state was updated
	state, _ = repo.Get("task-1")
	if state.CurrentIndex != 1 {
		t.Errorf("CurrentIndex = %d, want 1", state.CurrentIndex)
	}
}

func TestQueueProvider_AfterExecution_Exhausted(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)

	// Queue with only one item
	state := NewQueueState("task-1", []autosteer.SteerMode{"progress"})
	repo.Save(state)

	task := &tasks.TaskItem{ID: "task-1"}

	decision, err := provider.AfterExecution(task, "test-scenario")
	if err != nil {
		t.Fatalf("AfterExecution() error = %v", err)
	}

	if decision.ShouldRequeue != false {
		t.Errorf("ShouldRequeue = %v, want false", decision.ShouldRequeue)
	}
	if decision.Exhausted != true {
		t.Errorf("Exhausted = %v, want true", decision.Exhausted)
	}
	if decision.Reason != "queue_exhausted" {
		t.Errorf("Reason = %v, want queue_exhausted", decision.Reason)
	}
}

func TestQueueProvider_Reset(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)

	state := NewQueueState("task-1", []autosteer.SteerMode{"progress"})
	repo.Save(state)

	err := provider.Reset("task-1")
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	state, _ = repo.Get("task-1")
	if state != nil {
		t.Error("State should be deleted after Reset()")
	}
}

func TestQueueState_Advance(t *testing.T) {
	state := NewQueueState("task-1", []autosteer.SteerMode{"a", "b", "c"})

	if state.CurrentIndex != 0 {
		t.Fatalf("Initial CurrentIndex = %d, want 0", state.CurrentIndex)
	}

	hasMore := state.Advance()
	if !hasMore {
		t.Error("Advance() should return true when more items exist")
	}
	if state.CurrentIndex != 1 {
		t.Errorf("CurrentIndex = %d, want 1", state.CurrentIndex)
	}

	hasMore = state.Advance()
	if !hasMore {
		t.Error("Advance() should return true when more items exist")
	}
	if state.CurrentIndex != 2 {
		t.Errorf("CurrentIndex = %d, want 2", state.CurrentIndex)
	}

	hasMore = state.Advance()
	if hasMore {
		t.Error("Advance() should return false when exhausted")
	}
	if state.CurrentIndex != 3 {
		t.Errorf("CurrentIndex = %d, want 3", state.CurrentIndex)
	}
}

func TestQueueState_IsExhausted(t *testing.T) {
	state := NewQueueState("task-1", []autosteer.SteerMode{"a"})

	if state.IsExhausted() {
		t.Error("Should not be exhausted initially")
	}

	state.Advance()

	if !state.IsExhausted() {
		t.Error("Should be exhausted after advancing past last item")
	}
}

func TestQueueState_CurrentMode(t *testing.T) {
	state := NewQueueState("task-1", []autosteer.SteerMode{"a", "b"})

	if state.CurrentMode() != "a" {
		t.Errorf("CurrentMode() = %v, want a", state.CurrentMode())
	}

	state.Advance()

	if state.CurrentMode() != "b" {
		t.Errorf("CurrentMode() = %v, want b", state.CurrentMode())
	}

	state.Advance()

	if state.CurrentMode() != "" {
		t.Errorf("CurrentMode() = %v, want empty string when exhausted", state.CurrentMode())
	}
}

func TestQueueState_Position(t *testing.T) {
	state := NewQueueState("task-1", []autosteer.SteerMode{"a", "b", "c"})

	if state.Position() != "1/3" {
		t.Errorf("Position() = %v, want 1/3", state.Position())
	}

	state.Advance()

	if state.Position() != "2/3" {
		t.Errorf("Position() = %v, want 2/3", state.Position())
	}

	state.Advance()
	state.Advance()

	if state.Position() != "done" {
		t.Errorf("Position() = %v, want done", state.Position())
	}
}

func TestQueueState_Remaining(t *testing.T) {
	state := NewQueueState("task-1", []autosteer.SteerMode{"a", "b", "c"})

	if state.Remaining() != 3 {
		t.Errorf("Remaining() = %d, want 3", state.Remaining())
	}

	state.Advance()

	if state.Remaining() != 2 {
		t.Errorf("Remaining() = %d, want 2", state.Remaining())
	}

	state.Advance()
	state.Advance()

	if state.Remaining() != 0 {
		t.Errorf("Remaining() = %d, want 0", state.Remaining())
	}
}
