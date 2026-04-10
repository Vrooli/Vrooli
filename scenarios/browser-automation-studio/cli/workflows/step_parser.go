package workflows

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/vrooli/browser-automation-studio/workflow/validator"
)

// validKVKeys is the set of known parameter keys that can be used as key=value pairs.
// This is built from step definitions to avoid treating attribute selectors like
// [data-testid='dashboard'] as key-value pairs.
var validKVKeys map[string]bool

func init() {
	validKVKeys = buildValidKVKeys()
}

// buildValidKVKeys extracts all valid key names from step definitions.
func buildValidKVKeys() map[string]bool {
	keys := make(map[string]bool)

	// Common keys used across steps
	commonKeys := []string{
		"selector", "text", "value", "label", "timeoutMs", "timeout_ms",
	}
	for _, k := range commonKeys {
		keys[k] = true
	}

	// Build from step definitions
	for _, def := range validator.GetStepDefinitions() {
		// Add required KV keys
		for _, kv := range def.RequiredKVs {
			keys[kv.Key] = true
			// Also add snake_case variant
			if snakeKey := toSnakeCase(kv.Key); snakeKey != kv.Key {
				keys[snakeKey] = true
			}
		}
		// Add optional KV keys
		for _, kv := range def.OptionalKVs {
			keys[kv.Key] = true
			// Also add snake_case variant
			if snakeKey := toSnakeCase(kv.Key); snakeKey != kv.Key {
				keys[snakeKey] = true
			}
		}
		// Add positional's MapsTo key
		if def.Positional != nil && def.Positional.MapsTo != "" {
			keys[def.Positional.MapsTo] = true
		}
	}

	// Add scenario-specific keys
	extraKeys := []string{
		"scenario", "path", "url", "waitUntil", "wait_until",
		"assertMode", "assert_mode", "expectedText", "expected_text",
		"fullPage", "full_page", "durationMs", "duration_ms",
		"state", "attribute", "outputKey", "output_key",
		"optionText", "option_text", "optionValue", "option_value",
		"optionIndex", "option_index", "clickCount", "click_count",
		"button", "delay", "clear", "name", "expression", "script", "code",
		"key", "keys", "sourceSelector", "source_selector",
		"targetSelector", "target_selector",
		"workflowId", "workflow_id", "workflowPath", "workflow_path",
	}
	for _, k := range extraKeys {
		keys[k] = true
	}

	return keys
}

// toSnakeCase converts camelCase to snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(r + 32) // lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// isValidKVKey checks if a key is a known parameter name that should be treated
// as a key=value pair. This prevents attribute selectors like [data-testid='x']
// from being incorrectly parsed as key-value pairs.
func isValidKVKey(key string) bool {
	// Check direct match
	if validKVKeys[key] {
		return true
	}
	// Check if it's a nested key (e.g., resilience.maxAttempts)
	if strings.Contains(key, ".") {
		parts := strings.SplitN(key, ".", 2)
		// Allow resilience.* and other known nested patterns
		if parts[0] == "resilience" {
			return true
		}
	}
	return false
}

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

		// Check for key=value, but only if the key is a valid parameter name.
		// This prevents attribute selectors like [data-testid='dashboard'] from
		// being incorrectly parsed as key-value pairs.
		if idx := strings.Index(arg, "="); idx > 0 {
			key := arg[:idx]
			if isValidKVKey(key) {
				value := arg[idx+1:]
				step.KVPairs[key] = value
				continue
			}
		}

		// Not a valid KV pair - treat as positional
		if step.Positional == "" {
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
