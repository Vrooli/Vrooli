package workflows

import (
	"fmt"
	"strconv"
	"strings"
)

// StepSpec represents a parsed --step flag.
type StepSpec struct {
	Type       string            // navigate, click, type, etc.
	Positional string            // First non-key=value argument
	KVPairs    map[string]string // key=value pairs
}

// ParseSteps extracts all --step flags from args.
// Returns steps and remaining args (non-step flags).
func ParseSteps(args []string) ([]*StepSpec, []string, error) {
	var steps []*StepSpec
	var remaining []string

	for i := 0; i < len(args); i++ {
		if args[i] == "--step" {
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("--step requires a node type")
			}
			step, consumed, err := parseOneStep(args[i+1:])
			if err != nil {
				return nil, nil, err
			}
			steps = append(steps, step)
			i += consumed // consumed includes the node type and all its arguments
		} else {
			remaining = append(remaining, args[i])
		}
	}

	return steps, remaining, nil
}

// parseOneStep parses a single step starting from the node type.
// Returns the step, number of args consumed (including node type), and any error.
func parseOneStep(args []string) (*StepSpec, int, error) {
	if len(args) == 0 {
		return nil, 0, fmt.Errorf("--step requires a node type")
	}

	step := &StepSpec{
		Type:    args[0],
		KVPairs: make(map[string]string),
	}
	consumed := 1

	for i := 1; i < len(args); i++ {
		arg := args[i]

		// Stop at next flag
		if strings.HasPrefix(arg, "--") {
			break
		}

		consumed++

		// Check for key=value
		if idx := strings.Index(arg, "="); idx > 0 {
			key := arg[:idx]
			value := arg[idx+1:]
			step.KVPairs[key] = value
		} else if step.Positional == "" {
			// First non-kv is positional
			step.Positional = arg
		} else {
			return nil, 0, fmt.Errorf("unexpected argument in step '%s': %s", step.Type, arg)
		}
	}

	return step, consumed, nil
}

// parseValue attempts to convert a string value to its appropriate type.
// Returns int, float64, bool, or string.
func parseValue(s string) any {
	// Try to parse as integer
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	// Try to parse as float
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	// Try to parse as bool
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return s
}

// setNestedValue sets a value in a nested map structure using dot notation.
// For example, setNestedValue(data, "resilience.maxAttempts", 3) sets data["resilience"]["maxAttempts"] = 3
func setNestedValue(data map[string]any, path string, value any) {
	parts := strings.Split(path, ".")
	current := data

	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if _, ok := current[part]; !ok {
			current[part] = make(map[string]any)
		}
		if nested, ok := current[part].(map[string]any); ok {
			current = nested
		} else {
			// Cannot nest further; replace with new map
			newMap := make(map[string]any)
			current[part] = newMap
			current = newMap
		}
	}

	current[parts[len(parts)-1]] = value
}
