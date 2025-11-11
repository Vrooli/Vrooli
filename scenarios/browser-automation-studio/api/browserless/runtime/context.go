package runtime

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// ExecutionContext stores workflow-scoped variables during execution.
type ExecutionContext struct {
	mu        sync.RWMutex
	variables map[string]any
}

// NewExecutionContext creates an empty execution context.
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{variables: make(map[string]any)}
}

// Set assigns a value to the provided variable name.
func (ctx *ExecutionContext) Set(name string, value any) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("variable name is required")
	}
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	if ctx.variables == nil {
		ctx.variables = make(map[string]any)
	}
	ctx.variables[trimmed] = value
	return nil
}

// Get returns the variable value if present.
func (ctx *ExecutionContext) Get(name string) (any, bool) {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	value, ok := ctx.variables[strings.TrimSpace(name)]
	return value, ok
}

// Delete removes the provided variable.
func (ctx *ExecutionContext) Delete(name string) {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()
	delete(ctx.variables, strings.TrimSpace(name))
}

// Has reports whether the variable exists.
func (ctx *ExecutionContext) Has(name string) bool {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	_, ok := ctx.variables[strings.TrimSpace(name)]
	return ok
}

// GetString returns the variable coerced into a string.
func (ctx *ExecutionContext) GetString(name string) (string, error) {
	value, ok := ctx.Get(name)
	if !ok {
		return "", fmt.Errorf("variable %s is not defined", strings.TrimSpace(name))
	}
	switch typed := value.(type) {
	case string:
		return typed, nil
	case fmt.Stringer:
		return typed.String(), nil
	default:
		return fmt.Sprintf("%v", typed), nil
	}
}

// GetInt coerces the stored value into an integer if possible.
func (ctx *ExecutionContext) GetInt(name string) (int, error) {
	value, ok := ctx.Get(name)
	if !ok {
		return 0, fmt.Errorf("variable %s is not defined", strings.TrimSpace(name))
	}
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float32:
		return int(typed), nil
	case float64:
		return int(typed), nil
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return 0, nil
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, fmt.Errorf("variable %s is not numeric", name)
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("variable %s cannot be converted to int", name)
	}
}

// GetBool coerces the stored value into a boolean if possible.
func (ctx *ExecutionContext) GetBool(name string) (bool, error) {
	value, ok := ctx.Get(name)
	if !ok {
		return false, fmt.Errorf("variable %s is not defined", strings.TrimSpace(name))
	}
	switch typed := value.(type) {
	case bool:
		return typed, nil
	case string:
		trimmed := strings.ToLower(strings.TrimSpace(typed))
		if trimmed == "true" || trimmed == "1" {
			return true, nil
		}
		if trimmed == "false" || trimmed == "0" || trimmed == "" {
			return false, nil
		}
		return false, fmt.Errorf("variable %s cannot be converted to bool", name)
	case int, int32, int64:
		return fmt.Sprint(typed) != "0", nil
	case float32, float64:
		return fmt.Sprint(typed) != "0", nil
	default:
		return false, fmt.Errorf("variable %s cannot be converted to bool", name)
	}
}

// Snapshot returns a copy of the current variables for safe sharing.
func (ctx *ExecutionContext) Snapshot() map[string]any {
	ctx.mu.RLock()
	defer ctx.mu.RUnlock()
	if len(ctx.variables) == 0 {
		return nil
	}
	copy := make(map[string]any, len(ctx.variables))
	for k, v := range ctx.variables {
		copy[k] = v
	}
	return copy
}

var placeholderPattern = regexp.MustCompile(`\{\{([^}]+)\}\}`)

// interpolateString replaces {{var}} placeholders with variable values.
func interpolateString(raw string, ctx *ExecutionContext) (string, []string) {
	if ctx == nil {
		return raw, nil
	}
	missing := make([]string, 0)
	replaced := placeholderPattern.ReplaceAllStringFunc(raw, func(match string) string {
		inner := strings.TrimSpace(strings.Trim(match, "{}"))
		if inner == "" {
			return match
		}
		if value, ok := ctx.Get(inner); ok {
			return fmt.Sprintf("%v", value)
		}
		missing = append(missing, inner)
		return match
	})
	if len(missing) == 0 {
		return replaced, nil
	}
	return replaced, dedupeStrings(missing)
}

func dedupeStrings(values []string) []string {
	if len(values) <= 1 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	sort.Strings(unique)
	return unique
}

// interpolateMap recursively applies interpolation to all string values.
func interpolateMap(input map[string]any, ctx *ExecutionContext) (map[string]any, []string) {
	if len(input) == 0 || ctx == nil {
		return input, nil
	}
	missing := make([]string, 0)
	for key, value := range input {
		switch typed := value.(type) {
		case string:
			interpolated, keys := interpolateString(typed, ctx)
			input[key] = interpolated
			missing = append(missing, keys...)
		case map[string]any:
			updated, keys := interpolateMap(typed, ctx)
			input[key] = updated
			missing = append(missing, keys...)
		case []any:
			updated, keys := interpolateSlice(typed, ctx)
			input[key] = updated
			missing = append(missing, keys...)
		}
	}
	return input, dedupeStrings(missing)
}

func interpolateSlice(values []any, ctx *ExecutionContext) ([]any, []string) {
	if len(values) == 0 || ctx == nil {
		return values, nil
	}
	missing := make([]string, 0)
	for idx, value := range values {
		switch typed := value.(type) {
		case string:
			interpolated, keys := interpolateString(typed, ctx)
			values[idx] = interpolated
			missing = append(missing, keys...)
		case map[string]any:
			updated, keys := interpolateMap(typed, ctx)
			values[idx] = updated
			missing = append(missing, keys...)
		case []any:
			updated, keys := interpolateSlice(typed, ctx)
			values[idx] = updated
			missing = append(missing, keys...)
		}
	}
	return values, dedupeStrings(missing)
}
