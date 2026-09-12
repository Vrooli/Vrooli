package overview

import (
	"fmt"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
)

// GoalScopeSuggestion flags an active goal that has no scoped work. Unlike
// initiatives, goals deliberately have no explicit inter-goal dependency
// graph: their scope is defined by target prerequisite closure.
type GoalScopeSuggestion struct {
	Goal   string `json:"goal"`
	Reason string `json:"reason"`
}

// ConsistencyReport groups drift signals the Portfolio Manager and humans
// should surface but never act on automatically.
type ConsistencyReport struct {
	GoalScopeSuggestions []GoalScopeSuggestion `json:"goal_scope_suggestions"`
}

// maxSuggestions caps the payload to keep overview responses bounded even in
// large repos. Callers should treat the list as sampled when this cap is hit.
const maxSuggestions = 50

// computeGoalScopeSuggestions identifies active goals that no longer contain
// any backlog work. Item dependencies are intentionally not turned into goal
// dependencies because a single item may validly belong to multiple scopes.
func computeGoalScopeSuggestions(_ []backlog.BacklogItem, goalList []goals.GoalWithScope) []GoalScopeSuggestion {
	if len(goalList) == 0 {
		return nil
	}
	out := make([]GoalScopeSuggestion, 0)
	for _, goal := range goalList {
		if goal.Goal.Status == goals.StatusActive && len(goal.Scope.Closure) == 0 {
			out = append(out, GoalScopeSuggestion{Goal: goal.Goal.Name, Reason: fmt.Sprintf("active goal %q has no scoped backlog items", goal.Goal.Name)})
		}
	}

	if len(out) > maxSuggestions {
		out = out[:maxSuggestions]
	}
	return out
}
