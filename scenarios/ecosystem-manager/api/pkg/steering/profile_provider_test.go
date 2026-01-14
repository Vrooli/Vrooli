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
		t.Errorf("Strategy() = %v, want %v", provider.Strategy(), StrategyProfile)
	}
}

func TestProfileProvider_GetCurrentMode(t *testing.T) {
	integration := &mockAutoSteerIntegration{currentMode: "ux"}
	provider := NewProfileProvider(integration)

	task := &tasks.TaskItem{ID: "task-1"}

	mode, err := provider.GetCurrentMode(task)
	if err != nil {
		t.Fatalf("GetCurrentMode() error = %v", err)
	}
	if mode != "ux" {
		t.Errorf("GetCurrentMode() = %v, want ux", mode)
	}
}

func TestProfileProvider_GetCurrentMode_NilIntegration(t *testing.T) {
	provider := NewProfileProvider(nil)

	task := &tasks.TaskItem{ID: "task-1"}

	mode, err := provider.GetCurrentMode(task)
	if err != nil {
		t.Fatalf("GetCurrentMode() error = %v", err)
	}
	if mode != "" {
		t.Errorf("GetCurrentMode() = %v, want empty for nil integration", mode)
	}
}

func TestProfileProvider_GetCurrentMode_NilTask(t *testing.T) {
	integration := &mockAutoSteerIntegration{currentMode: "ux"}
	provider := NewProfileProvider(integration)

	mode, err := provider.GetCurrentMode(nil)
	if err != nil {
		t.Fatalf("GetCurrentMode() error = %v", err)
	}
	if mode != "" {
		t.Errorf("GetCurrentMode() = %v, want empty for nil task", mode)
	}
}

func TestProfileProvider_EnhancePrompt(t *testing.T) {
	// Note: EnhancePrompt requires a real ExecutionOrchestrator which returns
	// a concrete type that can't be mocked. This test verifies the nil orchestrator
	// path returns nil gracefully. Full integration tests should cover the orchestrator path.
	integration := &mockAutoSteerIntegration{}
	provider := NewProfileProvider(integration)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	// With nil orchestrator, returns nil enhancement
	if enhancement != nil {
		t.Error("EnhancePrompt() with nil orchestrator should return nil enhancement")
	}
}

func TestProfileProvider_EnhancePrompt_NilIntegration(t *testing.T) {
	provider := NewProfileProvider(nil)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement != nil {
		t.Error("EnhancePrompt() should return nil when integration is nil")
	}
}

func TestProfileProvider_EnhancePrompt_NilOrchestrator(t *testing.T) {
	// Test that nil orchestrator returns nil enhancement gracefully
	integration := &mockAutoSteerIntegration{}
	provider := NewProfileProvider(integration)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement != nil {
		t.Error("EnhancePrompt() with nil orchestrator should return nil")
	}
}

func TestProfileProvider_AfterExecution_Continue(t *testing.T) {
	integration := &mockAutoSteerIntegration{
		shouldContinue: true,
		currentMode:    autosteer.ModeTest,
	}
	provider := NewProfileProvider(integration)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	decision, err := provider.AfterExecution(task, "test-scenario")
	if err != nil {
		t.Fatalf("AfterExecution() error = %v", err)
	}
	if decision == nil {
		t.Fatal("AfterExecution() returned nil decision")
	}
	if !decision.ShouldRequeue {
		t.Error("AfterExecution().ShouldRequeue should be true when profile continues")
	}
	if decision.Exhausted {
		t.Error("AfterExecution().Exhausted should be false when profile continues")
	}
	if decision.Reason != "profile_continues" {
		t.Errorf("AfterExecution().Reason = %v, want profile_continues", decision.Reason)
	}
}

func TestProfileProvider_AfterExecution_Exhausted(t *testing.T) {
	integration := &mockAutoSteerIntegration{
		shouldContinue: false,
		currentMode:    autosteer.ModeTest,
	}
	provider := NewProfileProvider(integration)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	decision, err := provider.AfterExecution(task, "test-scenario")
	if err != nil {
		t.Fatalf("AfterExecution() error = %v", err)
	}
	if decision == nil {
		t.Fatal("AfterExecution() returned nil decision")
	}
	if decision.ShouldRequeue {
		t.Error("AfterExecution().ShouldRequeue should be false when profile exhausted")
	}
	if !decision.Exhausted {
		t.Error("AfterExecution().Exhausted should be true when profile completed")
	}
	if decision.Reason != "profile_completed" {
		t.Errorf("AfterExecution().Reason = %v, want profile_completed", decision.Reason)
	}
}

func TestProfileProvider_AfterExecution_NilIntegration(t *testing.T) {
	provider := NewProfileProvider(nil)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	decision, err := provider.AfterExecution(task, "test-scenario")
	if err != nil {
		t.Fatalf("AfterExecution() error = %v", err)
	}
	if decision == nil {
		t.Fatal("AfterExecution() returned nil decision")
	}
	if decision.ShouldRequeue {
		t.Error("AfterExecution().ShouldRequeue should be false for nil integration")
	}
	if !decision.Exhausted {
		t.Error("AfterExecution().Exhausted should be true for nil integration")
	}
	if decision.Reason != "no_integration" {
		t.Errorf("AfterExecution().Reason = %v, want no_integration", decision.Reason)
	}
}

func TestProfileProvider_AfterExecution_Error(t *testing.T) {
	integration := &mockAutoSteerIntegration{
		shouldContinueErr: errors.New("evaluation error"),
	}
	provider := NewProfileProvider(integration)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	_, err := provider.AfterExecution(task, "test-scenario")
	if err == nil {
		t.Error("AfterExecution() should return error when integration fails")
	}
}

func TestProfileProvider_Initialize(t *testing.T) {
	integration := &mockAutoSteerIntegration{}
	provider := NewProfileProvider(integration)

	task := &tasks.TaskItem{
		ID:     "task-1",
		Target: "test-scenario",
	}

	err := provider.Initialize(task)
	if err != nil {
		t.Errorf("Initialize() error = %v", err)
	}
	if !integration.initializeCalled {
		t.Error("Initialize() should call integration.InitializeAutoSteer")
	}
	if integration.initializeScenario != "test-scenario" {
		t.Errorf("Initialize() scenario = %v, want test-scenario", integration.initializeScenario)
	}
}

func TestProfileProvider_Initialize_UsesTargets(t *testing.T) {
	integration := &mockAutoSteerIntegration{}
	provider := NewProfileProvider(integration)

	task := &tasks.TaskItem{
		ID:      "task-1",
		Target:  "", // Empty target
		Targets: []string{"first-scenario", "second-scenario"},
	}

	err := provider.Initialize(task)
	if err != nil {
		t.Errorf("Initialize() error = %v", err)
	}
	if integration.initializeScenario != "first-scenario" {
		t.Errorf("Initialize() scenario = %v, want first-scenario", integration.initializeScenario)
	}
}

func TestProfileProvider_Initialize_NilIntegration(t *testing.T) {
	provider := NewProfileProvider(nil)

	task := &tasks.TaskItem{
		ID:     "task-1",
		Target: "test-scenario",
	}

	err := provider.Initialize(task)
	if err != nil {
		t.Errorf("Initialize() error = %v, want nil for nil integration", err)
	}
}

func TestProfileProvider_Reset(t *testing.T) {
	// Note: Reset requires a real ExecutionOrchestrator which returns
	// a concrete type that can't be mocked. This test verifies the nil orchestrator
	// path returns nil gracefully. Full integration tests should cover the orchestrator path.
	integration := &mockAutoSteerIntegration{}
	provider := NewProfileProvider(integration)

	err := provider.Reset("task-1")
	// With nil orchestrator, should return nil (no error)
	if err != nil {
		t.Errorf("Reset() with nil orchestrator error = %v, want nil", err)
	}
}

func TestProfileProvider_Reset_NilIntegration(t *testing.T) {
	provider := NewProfileProvider(nil)

	err := provider.Reset("task-1")
	if err != nil {
		t.Errorf("Reset() error = %v, want nil for nil integration", err)
	}
}

// mockAutoSteerIntegration is a test double for AutoSteerIntegrationAPI
type mockAutoSteerIntegration struct {
	shouldContinue     bool
	shouldContinueErr  error
	currentMode        autosteer.SteerMode
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

func (m *mockAutoSteerIntegration) GetCurrentMode(task *tasks.TaskItem) (autosteer.SteerMode, error) {
	return m.currentMode, nil
}

func (m *mockAutoSteerIntegration) ExecutionOrchestrator() *autosteer.ExecutionOrchestrator {
	// Note: ExecutionOrchestrator returns a concrete type that can't be easily mocked.
	// Tests that need orchestrator functionality should use integration tests.
	return nil
}
