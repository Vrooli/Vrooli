package steering

import (
	"testing"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

func TestQueueProvider_Strategy(t *testing.T) {
	provider := NewQueueProvider(nil, nil)
	if provider.Strategy() != StrategyQueue {
		t.Fatalf("Strategy() = %v, want %v", provider.Strategy(), StrategyQueue)
	}
}

func TestQueueProvider_InitializeAndGetCurrentSet(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)
	task := &tasks.TaskItem{ID: "task-1", SteeringQueue: [][]string{{"progress"}, {"ux", "test"}}}

	if err := provider.Initialize(task); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	state, err := repo.Get(task.ID)
	if err != nil || state == nil {
		t.Fatalf("repo.Get() = (%v, %v)", state, err)
	}
	if state.QueueLength != 2 || state.CurrentIndex != 0 {
		t.Fatalf("state = %#v, want queue_length=2 current_index=0", state)
	}

	set, err := provider.GetCurrentSet(task)
	if err != nil {
		t.Fatalf("GetCurrentSet() error = %v", err)
	}
	if len(set) != 1 || set[0] != "progress" {
		t.Fatalf("GetCurrentSet() = %#v, want [progress]", set)
	}
}

func TestQueueProvider_AfterExecution_AdvanceAndExhaust(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	provider := NewQueueProvider(repo, nil)
	task := &tasks.TaskItem{ID: "task-1", SteeringQueue: [][]string{{"progress"}, {"ux"}}}
	if err := provider.Initialize(task); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	decision, err := provider.AfterExecution(task, "scenario")
	if err != nil {
		t.Fatalf("AfterExecution() error = %v", err)
	}
	if !decision.ShouldRequeue || decision.Exhausted {
		t.Fatalf("first decision invalid: %#v", decision)
	}
	if len(decision.SkillSet) != 1 || decision.SkillSet[0] != "ux" {
		t.Fatalf("first decision skill_set = %#v, want [ux]", decision.SkillSet)
	}

	decision, err = provider.AfterExecution(task, "scenario")
	if err != nil {
		t.Fatalf("AfterExecution() error = %v", err)
	}
	if decision.ShouldRequeue || !decision.Exhausted || decision.Reason != "queue_exhausted" {
		t.Fatalf("second decision invalid: %#v", decision)
	}
}

func TestQueueProvider_EnhancePrompt_UsesCurrentQueueSet(t *testing.T) {
	repo := NewInMemoryQueueStateRepository()
	enhancer := &mockPromptEnhancer{skillSetSection: "<skills>xml</skills>"}
	provider := NewQueueProvider(repo, enhancer)
	task := &tasks.TaskItem{ID: "task-1", SteeringQueue: [][]string{{"progress"}, {"ux"}}}

	if err := provider.Initialize(task); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	enhancement, err := provider.EnhancePrompt(task)
	if err != nil {
		t.Fatalf("EnhancePrompt() error = %v", err)
	}
	if enhancement == nil {
		t.Fatal("EnhancePrompt() returned nil")
	}
	if enhancement.Source != "queue:progress[1/2]" {
		t.Fatalf("EnhancePrompt().Source = %q, want queue:progress[1/2]", enhancement.Source)
	}
	if enhancement.Section == "" {
		t.Fatal("EnhancePrompt().Section should not be empty")
	}
}

func TestQueueState_Helpers(t *testing.T) {
	state := NewQueueState("task-1", 3)
	if state.Position() != "1/3" || state.Remaining() != 3 || state.IsExhausted() {
		t.Fatalf("initial state unexpected: %#v", state)
	}
	if !state.Advance() || state.Position() != "2/3" {
		t.Fatalf("state after first advance unexpected: %#v", state)
	}
	if !state.Advance() || state.Position() != "3/3" {
		t.Fatalf("state after second advance unexpected: %#v", state)
	}
	if state.Advance() || !state.IsExhausted() || state.Position() != "done" || state.Remaining() != 0 {
		t.Fatalf("state after exhaustion unexpected: %#v", state)
	}
	state.Reset()
	if state.CurrentIndex != 0 || state.IsExhausted() {
		t.Fatalf("state after reset unexpected: %#v", state)
	}
}
