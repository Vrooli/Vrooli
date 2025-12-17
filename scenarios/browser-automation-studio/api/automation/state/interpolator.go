package state

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Interpolator handles variable substitution in workflow instructions.
// It supports both ${var} and {{var}} syntax with namespace and fallback support.
type Interpolator struct {
	state *ExecutionState
}

// NewInterpolator creates a new Interpolator for the given state.
func NewInterpolator(state *ExecutionState) *Interpolator {
	return &Interpolator{state: state}
}

// InterpolateInstruction performs variable substitution on instruction
// params/strings using ${var} tokens. Strings, maps, and slices are traversed
// recursively so nested params get interpolated.
func (i *Interpolator) InterpolateInstruction(instr contracts.CompiledInstruction) contracts.CompiledInstruction {
	if i.state == nil {
		return instr
	}
	i.state.mu.RLock()
	storeLen := len(i.state.store)
	i.state.mu.RUnlock()
	if storeLen == 0 {
		return instr
	}

	if instr.Params != nil {
		if params, ok := i.interpolateValue(instr.Params).(map[string]any); ok {
			instr.Params = params
		}
	}
	if instr.Context != nil {
		if ctx, ok := i.interpolateValue(instr.Context).(map[string]any); ok {
			instr.Context = ctx
		}
	}
	if instr.Action != nil {
		if updated := i.interpolateActionDefinition(instr.Action); updated != nil {
			instr.Action = updated
		}
	}
	return instr
}

// InterpolatePlanStep performs variable substitution on a plan step.
func (i *Interpolator) InterpolatePlanStep(step contracts.PlanStep) contracts.PlanStep {
	if i.state == nil {
		return step
	}
	i.state.mu.RLock()
	storeLen := len(i.state.store)
	i.state.mu.RUnlock()
	if storeLen == 0 {
		return step
	}

	if step.Params != nil {
		if params, ok := i.interpolateValue(step.Params).(map[string]any); ok {
			step.Params = params
		}
	}
	if step.Context != nil {
		if ctx, ok := i.interpolateValue(step.Context).(map[string]any); ok {
			step.Context = ctx
		}
	}
	if step.Action != nil {
		if updated := i.interpolateActionDefinition(step.Action); updated != nil {
			step.Action = updated
		}
	}
	return step
}

// InterpolateString performs string interpolation, always returning a string.
// Supports both ${...} and {{...}} template syntax with namespace and fallback support.
func (i *Interpolator) InterpolateString(s string) string {
	if i.state == nil {
		return s
	}
	// Support both ${var} and {{var}} template syntax
	if !strings.Contains(s, "${") && !strings.Contains(s, "{{") {
		return s
	}

	out := s
	maxIterations := 100 // Prevent infinite loops
	for iter := 0; iter < maxIterations; iter++ {
		// Look for both ${...} and {{...}} patterns
		dollarStart := strings.Index(out, "${")
		braceStart := strings.Index(out, "{{")

		var start int
		var prefix string
		var suffix string

		// Pick whichever comes first (or -1 if neither exists)
		if dollarStart != -1 && (braceStart == -1 || dollarStart < braceStart) {
			start = dollarStart
			prefix = "${"
			suffix = "}"
		} else if braceStart != -1 {
			start = braceStart
			prefix = "{{"
			suffix = "}}"
		} else {
			break
		}

		// Find matching closing bracket, handling nested brackets
		end := findMatchingClose(out[start+len(prefix):], suffix)
		if end == -1 {
			break
		}
		end = start + len(prefix) + end
		token := out[start+len(prefix) : end]

		if resolved, ok := i.resolveTokenWithFallback(token); ok {
			out = out[:start] + stringify(resolved) + out[end+len(suffix):]
		} else {
			// Drop unresolved token to avoid infinite loops
			out = out[:start] + out[end+len(suffix):]
		}
	}
	return out
}

// InterpolateValue performs variable substitution recursively on any value.
// For strings, it supports type preservation when the entire value is a single interpolation.
func (i *Interpolator) InterpolateValue(v any) any {
	return i.interpolateValue(v)
}

func (i *Interpolator) interpolateValue(v any) any {
	switch typed := v.(type) {
	case string:
		return i.interpolateStringTyped(typed)
	case map[string]any:
		clone := make(map[string]any, len(typed))
		for key, val := range typed {
			clone[key] = i.interpolateValue(val)
		}
		return clone
	case []any:
		out := make([]any, len(typed))
		for idx, val := range typed {
			out[idx] = i.interpolateValue(val)
		}
		return out
	default:
		return v
	}
}

// interpolateStringTyped performs interpolation with type preservation.
// If the string is a single interpolation (e.g., "${@store/count}"), the original
// type is preserved. If it contains mixed content, the result is always a string.
func (i *Interpolator) interpolateStringTyped(s string) any {
	// Check if this is a single interpolation that should preserve type
	if isSingleInterpolation(s) {
		token := extractSingleToken(s)
		if resolved, ok := i.resolveTokenWithFallback(token); ok {
			return resolved
		}
		return "" // Return empty string for unresolved single interpolation
	}

	// For mixed content or no interpolation, return string
	return i.InterpolateString(s)
}

func (i *Interpolator) interpolateActionDefinition(action *basactions.ActionDefinition) *basactions.ActionDefinition {
	if action == nil || i.state == nil {
		return action
	}
	i.state.mu.RLock()
	storeLen := len(i.state.store)
	i.state.mu.RUnlock()
	if storeLen == 0 {
		return action
	}

	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(action)
	if err != nil {
		return action
	}

	var doc any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return action
	}

	updated := i.interpolateValue(doc)
	updatedMap, ok := updated.(map[string]any)
	if !ok || updatedMap == nil {
		return action
	}
	updatedBytes, err := json.Marshal(updatedMap)
	if err != nil {
		return action
	}

	var out basactions.ActionDefinition
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(updatedBytes, &out); err != nil {
		return action
	}

	return proto.Clone(&out).(*basactions.ActionDefinition)
}

// resolveTokenWithFallback resolves a token that may contain fallback chains.
// Syntax: @namespace/path|@namespace/path2|"default"
// Returns the first defined value or (nil, false) if all are undefined.
func (i *Interpolator) resolveTokenWithFallback(token string) (any, bool) {
	alternatives := parseFallbackChain(token)
	for _, alt := range alternatives {
		if val, ok := i.resolveAlternative(alt); ok {
			return val, true
		}
	}
	return nil, false
}

// resolveAlternative resolves a single alternative (namespace reference or literal).
func (i *Interpolator) resolveAlternative(alt string) (any, bool) {
	alt = strings.TrimSpace(alt)
	if alt == "" {
		return nil, false
	}

	// Check for literal string value (quoted)
	if (strings.HasPrefix(alt, "\"") && strings.HasSuffix(alt, "\"")) ||
		(strings.HasPrefix(alt, "'") && strings.HasSuffix(alt, "'")) {
		return alt[1 : len(alt)-1], true
	}

	// Check for literal boolean
	if alt == "true" {
		return true, true
	}
	if alt == "false" {
		return false, true
	}

	// Check for literal number
	if num, err := strconv.Atoi(alt); err == nil {
		return num, true
	}
	if f, err := strconv.ParseFloat(alt, 64); err == nil {
		return f, true
	}

	// Check for namespace reference (@namespace/path)
	if strings.HasPrefix(alt, "@") {
		return i.resolveNamespacedToken(alt)
	}

	// Legacy: resolve against flat state (backward compatibility)
	return i.state.Resolve(alt)
}

// resolveNamespacedToken resolves a @namespace/path reference.
func (i *Interpolator) resolveNamespacedToken(token string) (any, bool) {
	if i.state == nil {
		return nil, false
	}

	// Strip leading @
	token = strings.TrimPrefix(token, "@")

	// Split into namespace and path
	slashIdx := strings.Index(token, "/")
	if slashIdx == -1 {
		// No path, just namespace - return entire namespace map
		if ns := i.state.GetNamespace(token); ns != nil {
			return ns, true
		}
		return nil, false
	}

	namespace := token[:slashIdx]
	path := token[slashIdx+1:]

	return i.state.ResolveNamespaced(namespace, path)
}

// isSingleInterpolation returns true if the string contains exactly one interpolation
// that spans the entire value (after trimming).
func isSingleInterpolation(s string) bool {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		// Check there's no nested ${
		inner := s[2 : len(s)-1]
		return !strings.Contains(inner, "${") && !strings.Contains(inner, "{{")
	}
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") {
		inner := s[2 : len(s)-2]
		return !strings.Contains(inner, "${") && !strings.Contains(inner, "{{")
	}
	return false
}

// extractSingleToken extracts the token from a single interpolation string.
func extractSingleToken(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "${") && strings.HasSuffix(s, "}") {
		return s[2 : len(s)-1]
	}
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") {
		return s[2 : len(s)-2]
	}
	return s
}

// parseFallbackChain splits a token by | into alternatives, respecting quoted strings.
func parseFallbackChain(token string) []string {
	var alternatives []string
	var current strings.Builder
	inQuote := rune(0)
	escaped := false

	for _, r := range token {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if inQuote != 0 {
			current.WriteRune(r)
			if r == inQuote {
				inQuote = 0
			}
			continue
		}
		if r == '"' || r == '\'' {
			inQuote = r
			current.WriteRune(r)
			continue
		}
		if r == '|' {
			alternatives = append(alternatives, strings.TrimSpace(current.String()))
			current.Reset()
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		alternatives = append(alternatives, strings.TrimSpace(current.String()))
	}
	return alternatives
}

// findMatchingClose finds the closing bracket, handling nested brackets.
func findMatchingClose(s, suffix string) int {
	depth := 0
	for idx := 0; idx < len(s); idx++ {
		if strings.HasPrefix(s[idx:], "${") || strings.HasPrefix(s[idx:], "{{") {
			depth++
		}
		if strings.HasPrefix(s[idx:], suffix) {
			if depth == 0 {
				return idx
			}
			depth--
		}
	}
	return -1
}

func stringify(val any) string {
	switch t := val.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case fmt.Stringer:
		return t.String()
	case map[string]any:
		// Special handling for extracted_data artifacts: {"value": actualValue}
		// Unwrap the value to avoid injecting JSON objects into templates
		if len(t) == 1 {
			if innerVal, ok := t["value"]; ok {
				return stringify(innerVal) // Recursively stringify the inner value
			}
		}
		// JSON marshal for other maps
		if b, err := json.Marshal(t); err == nil {
			return string(b)
		}
		return fmt.Sprint(t)
	default:
		if b, err := json.Marshal(t); err == nil {
			return string(b)
		}
		return fmt.Sprint(t)
	}
}

// EvaluateExpression supports simple boolean expressions such as:
// - true/false literals
// - ${var} == 3
// - ${var} != "ok"
// - ${count} > 1
// Whitespace around tokens is allowed. Only a single binary comparison is supported today.
func (i *Interpolator) EvaluateExpression(expr string) (bool, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return false, false
	}

	// Literal booleans
	if strings.EqualFold(expr, "true") {
		return true, true
	}
	if strings.EqualFold(expr, "false") {
		return false, true
	}

	tokens := tokenizeExpression(expr)
	if len(tokens) != 3 {
		return false, false
	}

	left := i.resolveOperand(tokens[0])
	right := i.resolveOperand(tokens[2])
	return CompareValues(left, right, tokens[1]), true
}

func (i *Interpolator) resolveOperand(token string) any {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(token, "${") && strings.HasSuffix(token, "}") {
		path := strings.TrimSuffix(strings.TrimPrefix(token, "${"), "}")
		if v, ok := i.state.Resolve(path); ok {
			return v
		}
		return nil
	}
	if unquoted, ok := unquote(token); ok {
		return unquoted
	}
	if num, err := strconv.Atoi(token); err == nil {
		return num
	}
	if f, err := strconv.ParseFloat(token, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(token); err == nil {
		return b
	}
	return token
}

func unquote(v string) (string, bool) {
	if len(v) < 2 {
		return v, false
	}
	start, end := v[0], v[len(v)-1]
	if (start == '"' && end == '"') || (start == '\'' && end == '\'') {
		return v[1 : len(v)-1], true
	}
	return v, false
}

// tokenizeExpression splits a simple binary expression into [lhs, op, rhs].
func tokenizeExpression(expr string) []string {
	var tokens []string
	var current strings.Builder
	inQuote := rune(0)

	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	for _, r := range expr {
		switch {
		case inQuote != 0:
			current.WriteRune(r)
			if r == inQuote {
				inQuote = 0
			}
		case r == '"' || r == '\'':
			inQuote = r
			current.WriteRune(r)
		case unicode.IsSpace(r):
			flush()
		case strings.ContainsRune("=<>!", r):
			flush()
			current.WriteRune(r)
			flush()
		default:
			current.WriteRune(r)
		}
	}
	flush()

	// Rejoin multi-char operators if they were split.
	if len(tokens) >= 3 {
		reconstructed := make([]string, 0, len(tokens))
		for idx := 0; idx < len(tokens); idx++ {
			switch tokens[idx] {
			case "=", "!", ">", "<":
				if idx+1 < len(tokens) && tokens[idx+1] == "=" {
					reconstructed = append(reconstructed, tokens[idx]+"=")
					idx++
					continue
				}
			}
			reconstructed = append(reconstructed, tokens[idx])
		}
		tokens = reconstructed
	}

	return tokens
}

// CompareValues compares two values using the specified operator.
func CompareValues(current any, expected any, op string) bool {
	switch op {
	case "", "eq", "==":
		return fmt.Sprint(current) == fmt.Sprint(expected)
	case "ne", "!=":
		return fmt.Sprint(current) != fmt.Sprint(expected)
	}

	// Numeric comparisons
	curFloat, curOK := ToFloat(current)
	expFloat, expOK := ToFloat(expected)
	if curOK && expOK {
		switch op {
		case "gt", ">":
			return curFloat > expFloat
		case "gte", ">=":
			return curFloat >= expFloat
		case "lt", "<":
			return curFloat < expFloat
		case "lte", "<=":
			return curFloat <= expFloat
		}
	}
	return false
}

// ToFloat converts a value to float64 if possible.
func ToFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case float64:
		return t, true
	case float32:
		return float64(t), true
	}
	return 0, false
}
