package steering

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestManualProvider_Strategy(t *testing.T) {
	provider := NewManualProvider(nil)
	if provider.Strategy() != StrategyManual {
		t.Fatalf("Strategy() = %v, want %v", provider.Strategy(), StrategyManual)
	}
}

func TestManualProvider_GetCurrentSet_ValidAndFallback(t *testing.T) {
	provider := NewManualProvider(nil)

	t.Run("valid set", func(t *testing.T) {
		set, err := provider.GetCurrentSet(&tasks.TaskItem{SteerSet: []string{"ux", "test"}})
		if err != nil {
			t.Fatalf("GetCurrentSet() error = %v", err)
		}
		if len(set) != 2 || set[0] != "ux" || set[1] != "test" {
			t.Fatalf("GetCurrentSet() = %#v, want [ux test]", set)
		}
	})

	t.Run("invalid values filtered", func(t *testing.T) {
		set, err := provider.GetCurrentSet(&tasks.TaskItem{SteerSet: []string{"invalid_xyz", "progress"}})
		if err != nil {
			t.Fatalf("GetCurrentSet() error = %v", err)
		}
		if len(set) != 1 || set[0] != "progress" {
			t.Fatalf("GetCurrentSet() = %#v, want [progress]", set)
		}
	})

	t.Run("fallback progress", func(t *testing.T) {
		set, err := provider.GetCurrentSet(nil)
		if err != nil {
			t.Fatalf("GetCurrentSet() error = %v", err)
		}
		if len(set) != 1 || set[0] != string(autosteer.ModeProgress) {
			t.Fatalf("GetCurrentSet() = %#v, want [progress]", set)
		}
	})
}

func TestManualProvider_EnhancePrompt(t *testing.T) {
	enhancer := &mockPromptEnhancer{skillSetSection: "## Test Section\nFocus"}
	provider := NewManualProvider(enhancer)

	enhancement, err := provider.EnhancePrompt(&tasks.TaskItem{ID: "task-1", SteerSet: []string{"progress", "ux"}})
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement == nil {
		t.Fatal("EnhancePrompt() returned nil")
	}
	if enhancement.Source != "manual:progress,ux" {
		t.Fatalf("EnhancePrompt().Source = %q, want manual:progress,ux", enhancement.Source)
	}
	if enhancement.Section == "" {
		t.Fatal("EnhancePrompt().Section should not be empty")
	}
}

func TestManualProvider_AfterExecution(t *testing.T) {
	provider := NewManualProvider(nil)
	decision, err := provider.AfterExecution(&tasks.TaskItem{ID: "task-1", SteerSet: []string{"ux"}}, "scenario")
	if err != nil {
		t.Fatalf("AfterExecution() error = %v", err)
	}
	if !decision.ShouldRequeue || decision.Exhausted {
		t.Fatalf("AfterExecution() invalid decision: %#v", decision)
	}
	if len(decision.SkillSet) != 1 || decision.SkillSet[0] != "ux" {
		t.Fatalf("AfterExecution().SkillSet = %#v, want [ux]", decision.SkillSet)
	}
	if decision.Reason != "manual_set_continues" {
		t.Fatalf("AfterExecution().Reason = %q", decision.Reason)
	}
}

func TestManualProvider_NilEnhancer(t *testing.T) {
	provider := NewManualProvider(nil)
	enhancement, err := provider.EnhancePrompt(&tasks.TaskItem{SteerSet: []string{"progress"}})
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement != nil {
		t.Fatal("EnhancePrompt() expected nil when no enhancer")
	}
}

// mockPromptEnhancer is a test double for autosteer.PromptEnhancerAPI.
type mockPromptEnhancer struct {
	skillSetSection  string
	autoSteerSection string
}

func (m *mockPromptEnhancer) GenerateSkillSetSection(skillIDs []string, withScope bool, scope string) string {
	return m.skillSetSection
}

func (m *mockPromptEnhancer) GenerateControllerSection(state *autosteer.ProfileExecutionState, profile *autosteer.AutoSteerProfile) string {
	return m.autoSteerSection
}
