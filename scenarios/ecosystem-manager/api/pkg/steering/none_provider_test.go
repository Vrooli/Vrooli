package steering

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestNoneProvider_Strategy(t *testing.T) {
	provider := NewNoneProvider(nil)
	if provider.Strategy() != StrategyNone {
		t.Errorf("Strategy() = %v, want %v", provider.Strategy(), StrategyNone)
	}
}

func TestNoneProvider_GetCurrentMode(t *testing.T) {
	provider := NewNoneProvider(nil)

	task := &tasks.TaskItem{ID: "task-1"}

	mode, err := provider.GetCurrentMode(task)
	if err != nil {
		t.Errorf("GetCurrentMode() error = %v", err)
	}
	if mode != autosteer.ModeProgress {
		t.Errorf("GetCurrentMode() = %v, want %v", mode, autosteer.ModeProgress)
	}
}

func TestNoneProvider_GetCurrentMode_NilTask(t *testing.T) {
	provider := NewNoneProvider(nil)

	mode, err := provider.GetCurrentMode(nil)
	if err != nil {
		t.Errorf("GetCurrentMode() error = %v", err)
	}
	// Even with nil task, returns Progress
	if mode != autosteer.ModeProgress {
		t.Errorf("GetCurrentMode() = %v, want %v", mode, autosteer.ModeProgress)
	}
}

func TestNoneProvider_EnhancePrompt(t *testing.T) {
	enhancer := &mockPromptEnhancer{
		modeSection: "## Progress Section\nDefault progress focus",
	}
	provider := NewNoneProvider(enhancer)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement == nil {
		t.Fatal("EnhancePrompt() returned nil enhancement")
	}
	if enhancement.Section != "## Progress Section\nDefault progress focus" {
		t.Errorf("EnhancePrompt().Section = %v, want progress section", enhancement.Section)
	}
	if enhancement.Source != "none:progress" {
		t.Errorf("EnhancePrompt().Source = %v, want none:progress", enhancement.Source)
	}
}

func TestNoneProvider_EnhancePrompt_NilEnhancer(t *testing.T) {
	provider := NewNoneProvider(nil)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement != nil {
		t.Error("EnhancePrompt() should return nil when promptEnhancer is nil")
	}
}

func TestNoneProvider_EnhancePrompt_EmptySection(t *testing.T) {
	enhancer := &mockPromptEnhancer{
		modeSection: "", // Empty section
	}
	provider := NewNoneProvider(enhancer)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement != nil {
		t.Error("EnhancePrompt() should return nil when section is empty")
	}
}

func TestNoneProvider_AfterExecution(t *testing.T) {
	provider := NewNoneProvider(nil)

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
	if decision.Mode != autosteer.ModeProgress {
		t.Errorf("AfterExecution().Mode = %v, want %v", decision.Mode, autosteer.ModeProgress)
	}
	if !decision.ShouldRequeue {
		t.Error("AfterExecution().ShouldRequeue should be true for none strategy")
	}
	if decision.Exhausted {
		t.Error("AfterExecution().Exhausted should be false (none never exhausts)")
	}
	if decision.Reason != "none_strategy_continues" {
		t.Errorf("AfterExecution().Reason = %v, want none_strategy_continues", decision.Reason)
	}
}

func TestNoneProvider_Initialize(t *testing.T) {
	provider := NewNoneProvider(nil)

	task := &tasks.TaskItem{
		ID: "task-1",
	}

	err := provider.Initialize(task)
	if err != nil {
		t.Errorf("Initialize() error = %v, want nil (no-op)", err)
	}
}

func TestNoneProvider_Reset(t *testing.T) {
	provider := NewNoneProvider(nil)

	err := provider.Reset("task-1")
	if err != nil {
		t.Errorf("Reset() error = %v, want nil (no-op)", err)
	}
}
