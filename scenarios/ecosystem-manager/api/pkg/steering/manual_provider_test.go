package steering

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestManualProvider_Strategy(t *testing.T) {
	provider := NewManualProvider(nil)
	if provider.Strategy() != StrategyManual {
		t.Errorf("Strategy() = %v, want %v", provider.Strategy(), StrategyManual)
	}
}

func TestManualProvider_GetCurrentMode(t *testing.T) {
	provider := NewManualProvider(nil)

	task := &tasks.TaskItem{
		ID:        "task-1",
		SteerMode: "ux",
	}

	mode, err := provider.GetCurrentMode(task)
	if err != nil {
		t.Errorf("GetCurrentMode() error = %v", err)
	}
	if mode != "ux" {
		t.Errorf("GetCurrentMode() = %v, want ux", mode)
	}
}

func TestManualProvider_GetCurrentMode_InvalidMode(t *testing.T) {
	provider := NewManualProvider(nil)

	task := &tasks.TaskItem{
		ID:        "task-1",
		SteerMode: "invalid_xyz",
	}

	mode, err := provider.GetCurrentMode(task)
	if err != nil {
		t.Errorf("GetCurrentMode() error = %v", err)
	}
	// Should fall back to progress for invalid mode
	if mode != "progress" {
		t.Errorf("GetCurrentMode() = %v, want progress (fallback)", mode)
	}
}

func TestManualProvider_GetCurrentMode_NilTask(t *testing.T) {
	provider := NewManualProvider(nil)

	mode, err := provider.GetCurrentMode(nil)
	if err != nil {
		t.Errorf("GetCurrentMode() error = %v", err)
	}
	// Should fall back to progress for nil task
	if mode != "progress" {
		t.Errorf("GetCurrentMode() = %v, want progress (fallback for nil)", mode)
	}
}

func TestManualProvider_EnhancePrompt(t *testing.T) {
	enhancer := &mockPromptEnhancer{
		modeSection: "## Test Section\nFocus on progress",
	}
	provider := NewManualProvider(enhancer)

	task := &tasks.TaskItem{
		ID:        "task-1",
		SteerMode: "progress",
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement == nil {
		t.Fatal("EnhancePrompt() returned nil enhancement")
	}
	if enhancement.Section != "## Test Section\nFocus on progress" {
		t.Errorf("EnhancePrompt().Section = %v, want test section", enhancement.Section)
	}
	if enhancement.Source != "manual:progress" {
		t.Errorf("EnhancePrompt().Source = %v, want manual:progress", enhancement.Source)
	}
}

func TestManualProvider_EnhancePrompt_InvalidMode(t *testing.T) {
	enhancer := &mockPromptEnhancer{
		modeSection: "## Progress Section\nDefault fallback",
	}
	provider := NewManualProvider(enhancer)

	task := &tasks.TaskItem{
		ID:        "task-1",
		SteerMode: "invalid_mode_xyz",
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement == nil {
		t.Fatal("EnhancePrompt() returned nil, should fall back to progress")
	}
	// Should fall back to progress mode
	if enhancement.Source != "manual:progress" {
		t.Errorf("EnhancePrompt().Source = %v, want manual:progress (fallback)", enhancement.Source)
	}
}

func TestManualProvider_EnhancePrompt_NilEnhancer(t *testing.T) {
	provider := NewManualProvider(nil)

	task := &tasks.TaskItem{
		ID:        "task-1",
		SteerMode: "progress",
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement != nil {
		t.Error("EnhancePrompt() should return nil when promptEnhancer is nil")
	}
}

func TestManualProvider_EnhancePrompt_NilTask(t *testing.T) {
	enhancer := &mockPromptEnhancer{
		modeSection: "## Progress Section",
	}
	provider := NewManualProvider(enhancer)

	enhancement, err := provider.EnhancePrompt(nil)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	// Should default to progress mode for nil task
	if enhancement != nil && enhancement.Source != "manual:progress" {
		t.Errorf("EnhancePrompt().Source = %v, want manual:progress for nil task", enhancement.Source)
	}
}

func TestManualProvider_AfterExecution(t *testing.T) {
	provider := NewManualProvider(nil)

	task := &tasks.TaskItem{
		ID:        "task-1",
		SteerMode: "ux",
	}

	decision, err := provider.AfterExecution(task, "test-scenario")
	if err != nil {
		t.Fatalf("AfterExecution() error = %v", err)
	}
	if decision == nil {
		t.Fatal("AfterExecution() returned nil decision")
	}
	if decision.Mode != "ux" {
		t.Errorf("AfterExecution().Mode = %v, want ux", decision.Mode)
	}
	if !decision.ShouldRequeue {
		t.Error("AfterExecution().ShouldRequeue should be true for manual mode")
	}
	if decision.Exhausted {
		t.Error("AfterExecution().Exhausted should be false (manual never exhausts)")
	}
	if decision.Reason != "manual_mode_continues" {
		t.Errorf("AfterExecution().Reason = %v, want manual_mode_continues", decision.Reason)
	}
}

func TestManualProvider_Initialize(t *testing.T) {
	provider := NewManualProvider(nil)

	task := &tasks.TaskItem{
		ID:        "task-1",
		SteerMode: "progress",
	}

	err := provider.Initialize(task)
	if err != nil {
		t.Errorf("Initialize() error = %v, want nil (no-op)", err)
	}
}

func TestManualProvider_Reset(t *testing.T) {
	provider := NewManualProvider(nil)

	err := provider.Reset("task-1")
	if err != nil {
		t.Errorf("Reset() error = %v, want nil (no-op)", err)
	}
}

// mockPromptEnhancer is a test double for autosteer.PromptEnhancerAPI
type mockPromptEnhancer struct {
	modeSection      string
	autoSteerSection string
}

func (m *mockPromptEnhancer) GenerateModeSection(mode autosteer.SteerMode) string {
	return m.modeSection
}

func (m *mockPromptEnhancer) GenerateAutoSteerSection(state *autosteer.ProfileExecutionState, profile *autosteer.AutoSteerProfile, evaluator autosteer.ConditionEvaluatorAPI) string {
	return m.autoSteerSection
}

func (m *mockPromptEnhancer) GeneratePhaseTransitionMessage(oldPhase, newPhase autosteer.SteerPhase, phaseNumber, totalPhases int) string {
	return ""
}

func (m *mockPromptEnhancer) GenerateCompletionMessage(profile *autosteer.AutoSteerProfile, state *autosteer.ProfileExecutionState) string {
	return ""
}
