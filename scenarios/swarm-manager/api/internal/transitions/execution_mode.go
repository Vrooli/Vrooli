package transitions

import (
	"fmt"
	"strings"
)

// ExecutionModeSliced and ExecutionModeGoal are the only execution modes. This
// is the single place the mode ids are spelled; every package that needs them
// references this set rather than repeating string literals.
const (
	ExecutionModeSliced = "sliced"
	ExecutionModeGoal   = "goal"
)

// ExecutionModeIDs is the ordered set of declared execution modes.
var ExecutionModeIDs = []string{ExecutionModeSliced, ExecutionModeGoal}

// NormalizeExecutionMode validates an execution mode id. Empty selects the
// default (sliced).
func NormalizeExecutionMode(raw string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ExecutionModeSliced, nil
	}
	switch value {
	case ExecutionModeSliced, ExecutionModeGoal:
		return value, nil
	default:
		return "", fmt.Errorf("execution_mode must be %q or %q", ExecutionModeSliced, ExecutionModeGoal)
	}
}
