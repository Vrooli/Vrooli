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

	task := &tasks.TaskItem{ID: "task-1"}

	mode, err := provider.GetCurrentMode(task)
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

	task := &tasks.TaskItem{ID: "nonexistent"}

	mode, err := provider.GetCurrentMode(task)
	if err != nil {
		t.Fatalf("GetCurrentMode() error = %v", err)
	}
	if mode != "" {
		t.Errorf("GetCurrentMode() = %v, want empty string", mode)
	}
}

func TestQueueProvider_GetCurrentMode_NilTask(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)

	mode, err := provider.GetCurrentMode(nil)
	if err != nil {
		t.Fatalf("GetCurrentMode() error = %v", err)
	}
	if mode != "" {
		t.Errorf("GetCurrentMode() = %v, want empty string for nil task", mode)
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

func TestQueueState_Reset(t *testing.T) {
	state := NewQueueState("task-1", []autosteer.SteerMode{"a", "b", "c"})
	state.Advance()
	state.Advance()

	if state.CurrentIndex != 2 {
		t.Fatalf("Pre-reset CurrentIndex = %d, want 2", state.CurrentIndex)
	}

	state.Reset()

	if state.CurrentIndex != 0 {
		t.Errorf("Reset() should set CurrentIndex to 0, got %d", state.CurrentIndex)
	}
	if state.IsExhausted() {
		t.Error("Reset() should make state not exhausted")
	}
	if state.CurrentMode() != "a" {
		t.Errorf("After Reset(), CurrentMode() = %v, want 'a'", state.CurrentMode())
	}
}

func TestQueueProvider_EnhancePrompt(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	enhancer := &mockPromptEnhancer{modeSection: "## Progress Mode\nFocus on progress"}
	provider := NewQueueProvider(repo, enhancer)

	// Pre-initialize state
	state := NewQueueState("task-1", []autosteer.SteerMode{"progress", "ux", "test"})
	repo.Save(state)

	task := &tasks.TaskItem{
		ID:            "task-1",
		SteeringQueue: []string{"progress", "ux", "test"},
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement == nil {
		t.Fatal("EnhancePrompt() returned nil enhancement")
	}

	// Check source format
	if enhancement.Source != "queue:progress[1/3]" {
		t.Errorf("EnhancePrompt().Source = %v, want queue:progress[1/3]", enhancement.Source)
	}

	// Check section contains mode content and queue progress
	if enhancement.Section == "" {
		t.Error("EnhancePrompt().Section should not be empty")
	}
	if !contains(enhancement.Section, "Queue Progress") {
		t.Error("EnhancePrompt().Section should contain 'Queue Progress'")
	}
	if !contains(enhancement.Section, "Position:") {
		t.Error("EnhancePrompt().Section should contain position info")
	}
}

func TestQueueProvider_EnhancePrompt_InitializesIfNoState(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	enhancer := &mockPromptEnhancer{modeSection: "## Progress Mode\nFocus on progress"}
	provider := NewQueueProvider(repo, enhancer)

	task := &tasks.TaskItem{
		ID:            "task-2",
		SteeringQueue: []string{"progress", "ux"},
	}

	// No state pre-exists - should initialize automatically
	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement == nil {
		t.Fatal("EnhancePrompt() returned nil enhancement after auto-init")
	}

	// Verify state was created
	state, _ := repo.Get("task-2")
	if state == nil {
		t.Error("State should have been auto-initialized")
	}
}

func TestQueueProvider_EnhancePrompt_Exhausted(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	enhancer := &mockPromptEnhancer{modeSection: "## Progress Mode"}
	provider := NewQueueProvider(repo, enhancer)

	// Create exhausted state
	state := NewQueueState("task-1", []autosteer.SteerMode{"progress"})
	state.Advance() // Now exhausted
	repo.Save(state)

	task := &tasks.TaskItem{
		ID:            "task-1",
		SteeringQueue: []string{"progress"},
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement != nil {
		t.Error("EnhancePrompt() should return nil for exhausted queue")
	}
}

func TestQueueProvider_EnhancePrompt_NilEnhancer(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)

	state := NewQueueState("task-1", []autosteer.SteerMode{"progress"})
	repo.Save(state)

	task := &tasks.TaskItem{
		ID:            "task-1",
		SteeringQueue: []string{"progress"},
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement != nil {
		t.Error("EnhancePrompt() should return nil when promptEnhancer is nil")
	}
}

func TestQueueProvider_EnhancePrompt_NilRepo(t *testing.T) {
	enhancer := &mockPromptEnhancer{}
	provider := NewQueueProvider(nil, enhancer)

	task := &tasks.TaskItem{
		ID:            "task-1",
		SteeringQueue: []string{"progress"},
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement != nil {
		t.Error("EnhancePrompt() should return nil when stateRepo is nil")
	}
}

func TestQueueProvider_EnhancePrompt_QueueProgressInfo(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	enhancer := &mockPromptEnhancer{modeSection: "## Mode Section"}
	provider := NewQueueProvider(repo, enhancer)

	tests := []struct {
		name           string
		queue          []autosteer.SteerMode
		currentIndex   int
		wantPosition   string
		wantUpcoming   bool
		wantMoreCount  int
	}{
		{
			name:          "first of three",
			queue:         []autosteer.SteerMode{"progress", "ux", "test"},
			currentIndex:  0,
			wantPosition:  "1/3",
			wantUpcoming:  true,
			wantMoreCount: 0,
		},
		{
			name:          "second of three",
			queue:         []autosteer.SteerMode{"progress", "ux", "test"},
			currentIndex:  1,
			wantPosition:  "2/3",
			wantUpcoming:  true,
			wantMoreCount: 0,
		},
		{
			name:          "last item",
			queue:         []autosteer.SteerMode{"progress", "ux", "test"},
			currentIndex:  2,
			wantPosition:  "3/3",
			wantUpcoming:  false, // No upcoming when on last item
			wantMoreCount: 0,
		},
		{
			name:          "long queue with more items",
			queue:         []autosteer.SteerMode{"a", "b", "c", "d", "e", "f", "g"},
			currentIndex:  0,
			wantPosition:  "1/7",
			wantUpcoming:  true,
			wantMoreCount: 3, // Shows 3 upcoming, then "... and 3 more"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := NewQueueState("task-"+tt.name, tt.queue)
			state.CurrentIndex = tt.currentIndex
			repo.Save(state)

			task := &tasks.TaskItem{
				ID:            "task-" + tt.name,
				SteeringQueue: toStringSlice(tt.queue),
			}

			enhancement, err := provider.EnhancePrompt(task)
			if err != nil {
				t.Fatalf("EnhancePrompt() error = %v", err)
			}
			if enhancement == nil {
				t.Fatal("EnhancePrompt() returned nil")
			}

			// Check position
			if !contains(enhancement.Section, tt.wantPosition) {
				t.Errorf("Section should contain position %s", tt.wantPosition)
			}

			// Check upcoming section presence
			hasUpcoming := contains(enhancement.Section, "Upcoming:")
			if tt.wantUpcoming && !hasUpcoming {
				t.Error("Section should contain 'Upcoming:' section")
			}
			if !tt.wantUpcoming && hasUpcoming {
				t.Error("Section should NOT contain 'Upcoming:' section for last item")
			}

			// Check "... and X more" for long queues
			if tt.wantMoreCount > 0 {
				if !contains(enhancement.Section, "... and") {
					t.Errorf("Section should contain '... and %d more'", tt.wantMoreCount)
				}
			}
		})
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toStringSlice(modes []autosteer.SteerMode) []string {
	result := make([]string, len(modes))
	for i, m := range modes {
		result[i] = string(m)
	}
	return result
}
