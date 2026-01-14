package steering

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestRegistry_DetermineStrategy(t *testing.T) {
	tests := []struct {
		name     string
		task     *tasks.TaskItem
		expected SteeringStrategy
	}{
		{
			name:     "nil task returns none",
			task:     nil,
			expected: StrategyNone,
		},
		{
			name: "profile ID takes priority",
			task: &tasks.TaskItem{
				AutoSteerProfileID: "profile-123",
				SteerMode:          "ux",
				SteeringQueue:      []string{"test", "refactor"},
			},
			expected: StrategyProfile,
		},
		{
			name: "queue takes priority over manual",
			task: &tasks.TaskItem{
				SteerMode:     "ux",
				SteeringQueue: []string{"test", "refactor"},
			},
			expected: StrategyQueue,
		},
		{
			name: "manual mode when no profile or queue",
			task: &tasks.TaskItem{
				SteerMode: "ux",
			},
			expected: StrategyManual,
		},
		{
			name: "none when no steering configured",
			task: &tasks.TaskItem{
				ID: "task-1",
			},
			expected: StrategyNone,
		},
		{
			name: "whitespace profile ID treated as empty",
			task: &tasks.TaskItem{
				AutoSteerProfileID: "   ",
				SteerMode:          "test",
			},
			expected: StrategyManual,
		},
		{
			name: "whitespace steer mode treated as empty",
			task: &tasks.TaskItem{
				SteerMode: "   ",
			},
			expected: StrategyNone,
		},
		{
			name: "empty queue treated as no queue",
			task: &tasks.TaskItem{
				SteerMode:     "ux",
				SteeringQueue: []string{},
			},
			expected: StrategyManual,
		},
	}

	registry := NewRegistry(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.DetermineStrategy(tt.task)
			if result != tt.expected {
				t.Errorf("DetermineStrategy() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRegistry_GetProvider(t *testing.T) {
	mockNone := &mockProvider{strategy: StrategyNone}
	mockManual := &mockProvider{strategy: StrategyManual}
	mockQueue := &mockProvider{strategy: StrategyQueue}

	registry := NewRegistry(map[SteeringStrategy]SteeringProvider{
		StrategyNone:   mockNone,
		StrategyManual: mockManual,
		StrategyQueue:  mockQueue,
	})

	tests := []struct {
		name     string
		task     *tasks.TaskItem
		expected SteeringProvider
	}{
		{
			name:     "returns none provider for nil task",
			task:     nil,
			expected: mockNone,
		},
		{
			name: "returns manual provider for manual task",
			task: &tasks.TaskItem{
				SteerMode: "ux",
			},
			expected: mockManual,
		},
		{
			name: "returns queue provider for queue task",
			task: &tasks.TaskItem{
				SteeringQueue: []string{"test", "refactor"},
			},
			expected: mockQueue,
		},
		{
			name: "returns none provider for unconfigured task",
			task: &tasks.TaskItem{
				ID: "task-1",
			},
			expected: mockNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := registry.GetProvider(tt.task)
			if result != tt.expected {
				t.Errorf("GetProvider() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRegistry_GetProvider_FallbackToNone(t *testing.T) {
	mockNone := &mockProvider{strategy: StrategyNone}

	// Registry with only none provider
	registry := NewRegistry(map[SteeringStrategy]SteeringProvider{
		StrategyNone: mockNone,
	})

	// Task requesting profile (not registered)
	task := &tasks.TaskItem{
		AutoSteerProfileID: "profile-123",
	}

	result := registry.GetProvider(task)
	if result != mockNone {
		t.Errorf("GetProvider() should fallback to none provider, got %v", result)
	}
}

func TestRegistry_HasProvider(t *testing.T) {
	mockNone := &mockProvider{strategy: StrategyNone}

	registry := NewRegistry(map[SteeringStrategy]SteeringProvider{
		StrategyNone: mockNone,
	})

	if !registry.HasProvider(StrategyNone) {
		t.Error("HasProvider(StrategyNone) should return true")
	}

	if registry.HasProvider(StrategyProfile) {
		t.Error("HasProvider(StrategyProfile) should return false")
	}
}

func TestRegistry_RegisterProvider(t *testing.T) {
	registry := NewRegistry(nil)

	mockManual := &mockProvider{strategy: StrategyManual}
	registry.RegisterProvider(StrategyManual, mockManual)

	if !registry.HasProvider(StrategyManual) {
		t.Error("Provider should be registered")
	}

	result := registry.GetProvider(&tasks.TaskItem{SteerMode: "ux"})
	if result != mockManual {
		t.Errorf("GetProvider() = %v, want %v", result, mockManual)
	}
}

// mockProvider is a simple mock for testing
type mockProvider struct {
	strategy SteeringStrategy
}

func (m *mockProvider) Strategy() SteeringStrategy { return m.strategy }
func (m *mockProvider) GetCurrentMode(task *tasks.TaskItem) (autosteer.SteerMode, error) {
	return "", nil
}
func (m *mockProvider) EnhancePrompt(task *tasks.TaskItem) (*PromptEnhancement, error) {
	return nil, nil
}
func (m *mockProvider) AfterExecution(task *tasks.TaskItem, scenarioName string) (*SteeringDecision, error) {
	return nil, nil
}
func (m *mockProvider) Initialize(task *tasks.TaskItem) error { return nil }
func (m *mockProvider) Reset(taskID string) error             { return nil }
