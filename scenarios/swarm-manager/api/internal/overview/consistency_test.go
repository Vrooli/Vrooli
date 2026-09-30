package overview

import (
	"testing"

	"swarm-manager/internal/goals"
)

func TestComputeGoalScopeSuggestions_EmptyScope(t *testing.T) {
	suggestions := computeGoalScopeSuggestions(nil, []goals.GoalWithScope{
		{Goal: goals.Goal{Name: "empty", Status: goals.StatusActive}},
		{Goal: goals.Goal{Name: "archived", Status: goals.StatusArchived}},
	})
	if len(suggestions) != 1 {
		t.Fatalf("expected one suggestion, got %+v", suggestions)
	}
	if suggestions[0].Goal != "empty" {
		t.Errorf("expected empty goal suggestion, got %+v", suggestions[0])
	}
}

func TestComputeGoalScopeSuggestions_ClosureHasWork(t *testing.T) {
	suggestions := computeGoalScopeSuggestions(nil, []goals.GoalWithScope{
		{Goal: goals.Goal{Name: "ready", Status: goals.StatusActive}, Scope: goals.Scope{Closure: []string{"execute/task"}}},
	})
	if len(suggestions) != 0 {
		t.Errorf("expected no suggestions, got %+v", suggestions)
	}
}
