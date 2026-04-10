package steering

import (
	"errors"
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestProfileProvider_Strategy(t *testing.T) {
	provider := NewProfileProvider(nil)
	if provider.Strategy() != StrategyProfile {
		t.Fatalf("Strategy() = %v, want %v", provider.Strategy(), StrategyProfile)
	}
}

func TestProfileProvider_GetCurrentSet(t *testing.T) {
	provider := NewProfileProvider(&mockAutoSteerIntegration{currentSet: []string{"ux", "test"}})
	set, err := provider.GetCurrentSet(&tasks.TaskItem{ID: "task-1"})
	if err != nil {
		t.Fatalf("GetCurrentSet() error = %v", err)
	}
	if len(set) != 2 || set[0] != "ux" || set[1] != "test" {
		t.Fatalf("GetCurrentSet() = %#v, want [ux test]", set)
	}
}

func TestProfileProvider_AfterExecution(t *testing.T) {
	t.Run("continue", func(t *testing.T) {
		provider := NewProfileProvider(&mockAutoSteerIntegration{shouldContinue: true, currentSet: []string{"test"}})
		decision, err := provider.AfterExecution(&tasks.TaskItem{ID: "task-1"}, "scenario")
		if err != nil {
			t.Fatalf("AfterExecution() error = %v", err)
		}
		if !decision.ShouldRequeue || decision.Exhausted || decision.Reason != "profile_continues" {
			t.Fatalf("AfterExecution() invalid decision: %#v", decision)
		}
	})

	t.Run("exhausted", func(t *testing.T) {
		provider := NewProfileProvider(&mockAutoSteerIntegration{shouldContinue: false, currentSet: []string{"test"}})
		decision, err := provider.AfterExecution(&tasks.TaskItem{ID: "task-1"}, "scenario")
		if err != nil {
			t.Fatalf("AfterExecution() error = %v", err)
		}
		if decision.ShouldRequeue || !decision.Exhausted || decision.Reason != "profile_completed" {
			t.Fatalf("AfterExecution() invalid decision: %#v", decision)
		}
	})

	t.Run("error", func(t *testing.T) {
		provider := NewProfileProvider(&mockAutoSteerIntegration{shouldContinueErr: errors.New("boom")})
		if _, err := provider.AfterExecution(&tasks.TaskItem{ID: "task-1"}, "scenario"); err == nil {
			t.Fatal("AfterExecution() expected error")
		}
	})
}

// mockAutoSteerIntegration is a test double for AutoSteerIntegrationAPI.
type mockAutoSteerIntegration struct {
	shouldContinue     bool
	shouldContinueErr  error
	currentSet         []string
	initializeCalled   bool
	initializeScenario string
	initializeErr      error
}

func (m *mockAutoSteerIntegration) InitializeAutoSteer(task *tasks.TaskItem, scenarioName string) error {
	m.initializeCalled = true
	m.initializeScenario = scenarioName
	return m.initializeErr
}

func (m *mockAutoSteerIntegration) EnhancePrompt(task *tasks.TaskItem, basePrompt string) (string, error) {
	return basePrompt, nil
}

func (m *mockAutoSteerIntegration) ShouldContinueTask(task *tasks.TaskItem, scenarioName string) (bool, error) {
	return m.shouldContinue, m.shouldContinueErr
}

func (m *mockAutoSteerIntegration) GetCurrentSet(task *tasks.TaskItem) ([]string, error) {
	return m.currentSet, nil
}

func (m *mockAutoSteerIntegration) ExecutionOrchestrator() *autosteer.ExecutionOrchestrator {
	return nil
}
