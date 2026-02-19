package steering

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestNoneProvider_Strategy(t *testing.T) {
	provider := NewNoneProvider(nil)
	if provider.Strategy() != StrategyNone {
		t.Fatalf("Strategy() = %v, want %v", provider.Strategy(), StrategyNone)
	}
}

func TestNoneProvider_GetCurrentSet(t *testing.T) {
	provider := NewNoneProvider(nil)
	set, err := provider.GetCurrentSet(&tasks.TaskItem{ID: "task-1"})
	if err != nil {
		t.Fatalf("GetCurrentSet() error = %v", err)
	}
	if len(set) != 1 || set[0] != "progress" {
		t.Fatalf("GetCurrentSet() = %#v, want [progress]", set)
	}
}

func TestNoneProvider_EnhancePrompt(t *testing.T) {
	enhancer := &mockPromptEnhancer{skillSetSection: "## Progress Section"}
	provider := NewNoneProvider(enhancer)

	enhancement, err := provider.EnhancePrompt(&tasks.TaskItem{ID: "task-1"})
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement == nil || enhancement.Source != "none:progress" {
		t.Fatalf("EnhancePrompt() = %#v", enhancement)
	}
}

func TestNoneProvider_AfterExecution(t *testing.T) {
	provider := NewNoneProvider(nil)
	decision, err := provider.AfterExecution(&tasks.TaskItem{ID: "task-1"}, "scenario")
	if err != nil {
		t.Fatalf("AfterExecution() error = %v", err)
	}
	if !decision.ShouldRequeue || decision.Exhausted {
		t.Fatalf("AfterExecution() invalid decision: %#v", decision)
	}
	if len(decision.SkillSet) != 1 || decision.SkillSet[0] != "progress" {
		t.Fatalf("AfterExecution().SkillSet = %#v", decision.SkillSet)
	}
}
