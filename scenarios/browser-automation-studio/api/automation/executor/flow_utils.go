package executor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// flow_utils.go holds small helpers for graph traversal and value coercion.

// executionState provides namespace-aware variable management for workflow execution.
// It separates runtime state (@store/), input parameters (@params/), and environment
// configuration (@env/) into distinct namespaces with different mutability semantics.
type executionState struct {
	store  map[string]any // @store/ - mutable runtime state, writable via setVariable/storeResult
	params map[string]any // @params/ - read-only input parameters from execution request or subflow call
	env    map[string]any // @env/ - read-only environment configuration from project settings
}

// newExecutionState creates a new executionState with the given initial values.
func newExecutionState(initialStore, initialParams, env map[string]any) *executionState {
	state := &executionState{
		store:  make(map[string]any),
		params: make(map[string]any),
		env:    make(map[string]any),
	}
	for k, v := range initialStore {
		state.store[k] = v
	}
	for k, v := range initialParams {
		state.params[k] = v
	}
	for k, v := range env {
		state.env[k] = v
	}
	return state
}

// getNamespace returns the appropriate map for the given namespace.
func (s *executionState) getNamespace(namespace string) map[string]any {
	if s == nil {
		return nil
	}
	switch namespace {
	case "store":
		return s.store
	case "params":
		return s.params
	case "env":
		return s.env
	default:
		return nil
	}
}

// resolveNamespaced looks up a value in the specified namespace using dot notation.
// Returns (value, true) if found, (nil, false) otherwise.
func (s *executionState) resolveNamespaced(namespace, path string) (any, bool) {
	if s == nil {
		return nil, false
	}
	ns := s.getNamespace(namespace)
	if ns == nil {
		return nil, false
	}
	return resolvePath(ns, path)
}

// setStore sets a value in the @store/ namespace (the only mutable namespace).
func (s *executionState) setStore(key string, value any) {
	if s == nil {
		return
	}
	if s.store == nil {
		s.store = make(map[string]any)
	}
	s.store[key] = value
}

// mergeStore merges the given map into the @store/ namespace.
func (s *executionState) mergeStore(vars map[string]any) {
	if s == nil || vars == nil {
		return
	}
	for k, v := range vars {
		s.setStore(k, v)
	}
}

// copyStore returns a copy of the @store/ namespace.
func (s *executionState) copyStore() map[string]any {
	if s == nil {
		return make(map[string]any)
	}
	return copyMap(s.store)
}

// copyParams returns a copy of the @params/ namespace.
func (s *executionState) copyParams() map[string]any {
	if s == nil {
		return make(map[string]any)
	}
	return copyMap(s.params)
}

// copyEnv returns a copy of the @env/ namespace.
func (s *executionState) copyEnv() map[string]any {
	if s == nil {
		return make(map[string]any)
	}
	return copyMap(s.env)
}

// resolvePath resolves a dot-notation path against a map.
func resolvePath(m map[string]any, path string) (any, bool) {
	if m == nil || path == "" {
		return nil, false
	}

	// Try direct lookup first
	if v, ok := m[path]; ok {
		return v, true
	}

	// Handle dot notation
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, false
	}

	current, ok := m[parts[0]]
	if !ok {
		return nil, false
	}

	for _, part := range parts[1:] {
		switch val := current.(type) {
		case map[string]any:
			current, ok = val[part]
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(val) {
				return nil, false
			}
			current = val[idx]
			ok = true
		default:
			return nil, false
		}
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// flowState is the legacy state container used for backward compatibility.
// New code should use executionState for namespace-aware variable management.
type flowState struct {
	vars         map[string]any
	nextIndex    int
	entryChecked bool

	// Namespace-aware state (optional, for migration)
	namespaced *executionState
}

func newFlowState(seed map[string]any) *flowState {
	state := &flowState{vars: map[string]any{}}
	for k, v := range seed {
		state.vars[k] = v
	}
	return state
}

// newFlowStateWithNamespaces creates a flowState with namespace-aware state management.
// The legacy vars map is populated from initialStore for backward compatibility.
func newFlowStateWithNamespaces(initialStore, initialParams, env map[string]any) *flowState {
	state := &flowState{
		vars:       make(map[string]any),
		namespaced: newExecutionState(initialStore, initialParams, env),
	}
	// Populate legacy vars from store for backward compatibility
	for k, v := range initialStore {
		state.vars[k] = v
	}
	return state
}

// hasNamespaces returns true if this flowState has namespace-aware state.
func (s *flowState) hasNamespaces() bool {
	return s != nil && s.namespaced != nil
}

func (s *flowState) markEntryChecked() {
	if s == nil {
		return
	}
	s.entryChecked = true
}

func (s *flowState) hasCheckedEntry() bool {
	if s == nil {
		return false
	}
	return s.entryChecked
}

func (s *flowState) get(key string) (any, bool) {
	if s == nil {
		return nil, false
	}
	v, ok := s.vars[key]
	return v, ok
}

// resolve supports dot/index path resolution (e.g., user.name, items.0) and
// falls back to a direct lookup for raw keys.
func (s *flowState) resolve(path string) (any, bool) {
	if s == nil {
		return nil, false
	}
	if v, ok := s.get(path); ok {
		return v, true
	}

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, false
	}

	current, ok := s.get(parts[0])
	if !ok {
		return nil, false
	}

	for _, part := range parts[1:] {
		switch val := current.(type) {
		case map[string]any:
			current, ok = val[part]
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(val) {
				return nil, false
			}
			current = val[idx]
			ok = true
		default:
			return nil, false
		}
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func (s *flowState) set(key string, value any) {
	if s == nil {
		return
	}
	if s.vars == nil {
		s.vars = map[string]any{}
	}
	s.vars[key] = value
	// Also update namespaced store if available
	if s.namespaced != nil {
		s.namespaced.setStore(key, value)
	}
}

// setNamespaced sets a value in the specified namespace.
// Only @store/ is writable; attempts to write to @params/ or @env/ are silently ignored.
func (s *flowState) setNamespaced(namespace, key string, value any) {
	if s == nil {
		return
	}
	if namespace != "store" {
		return // Only store is writable
	}
	s.set(key, value)
}

func (s *flowState) merge(vars map[string]any) {
	if s == nil || vars == nil {
		return
	}
	for k, v := range vars {
		s.set(k, v)
	}
}

// mergeNamespacedStore merges vars into the @store/ namespace.
func (s *flowState) mergeNamespacedStore(vars map[string]any) {
	if s == nil || vars == nil {
		return
	}
	for k, v := range vars {
		s.set(k, v)
	}
}

// getNamespacedState returns the underlying executionState, creating one if needed.
func (s *flowState) getNamespacedState() *executionState {
	if s == nil {
		return nil
	}
	if s.namespaced == nil {
		// Upgrade to namespaced state on demand
		s.namespaced = newExecutionState(s.vars, nil, nil)
	}
	return s.namespaced
}

func (s *flowState) setNextIndexFromPlan(plan contracts.ExecutionPlan) {
	if s == nil {
		return
	}
	maxIdx := -1
	for _, instr := range plan.Instructions {
		if instr.Index > maxIdx {
			maxIdx = instr.Index
		}
	}
	maxIdx = maxInt(maxIdx, maxGraphIndex(plan.Graph))
	if maxIdx >= s.nextIndex {
		s.nextIndex = maxIdx + 1
	}
}

func (s *flowState) allocateIndexRange(count int) int {
	if s == nil {
		return 0
	}
	base := s.nextIndex
	s.nextIndex += count
	return base
}

func extractLoopItems(params map[string]any, state *flowState) []any {
	if params == nil {
		return nil
	}
	if raw, ok := params["loopItems"]; ok {
		if arr := coerceArray(raw); len(arr) > 0 {
			return arr
		}
	}
	if raw, ok := params["items"]; ok {
		if arr := coerceArray(raw); len(arr) > 0 {
			return arr
		}
	}
	source := ""
	if raw, ok := params["arraySource"]; ok {
		source, _ = raw.(string)
	}
	if source == "" {
		if raw, ok := params["loopArraySource"]; ok {
			source, _ = raw.(string)
		}
	}
	if source != "" {
		if v, ok := state.get(source); ok {
			if arr := coerceArray(v); len(arr) > 0 {
				return arr
			}
		}
	}
	return nil
}

func evaluateLoopCondition(params map[string]any, state *flowState) bool {
	if params == nil {
		return false
	}
	condType := strings.ToLower(strings.TrimSpace(stringValue(params, "conditionType")))
	if condType == "" {
		condType = strings.ToLower(strings.TrimSpace(stringValue(params, "loopConditionType")))
	}
	if condType == "" {
		condType = "variable"
	}

	switch condType {
	case "variable":
		name := stringValue(params, "conditionVariable")
		if name == "" {
			name = stringValue(params, "loopConditionVariable")
		}
		if name == "" {
			return false
		}
		op := strings.ToLower(strings.TrimSpace(stringValue(params, "conditionOperator")))
		if op == "" {
			op = strings.ToLower(strings.TrimSpace(stringValue(params, "loopConditionOperator")))
		}
		expected := firstPresent(params, "conditionValue", "loopConditionValue")
		current, ok := state.get(name)
		if !ok {
			return false
		}
		return compareValues(current, expected, op)
	case "expression":
		expr := strings.TrimSpace(stringValue(params, "conditionExpression"))
		if expr == "" {
			expr = strings.TrimSpace(stringValue(params, "loopConditionExpression"))
		}
		if expr == "" {
			return false
		}
		result, ok := evaluateExpression(expr, state)
		if !ok {
			return false
		}
		return result
	default:
		return false
	}
}

func compareValues(current any, expected any, op string) bool {
	switch op {
	case "", "eq", "==":
		return fmt.Sprint(current) == fmt.Sprint(expected)
	case "ne", "!=":
		return fmt.Sprint(current) != fmt.Sprint(expected)
	}

	// Numeric comparisons
	curFloat, curOK := toFloat(current)
	expFloat, expOK := toFloat(expected)
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

func toFloat(v any) (float64, bool) {
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

// interpolateInstruction performs variable substitution on instruction
// params/strings using ${var} tokens. Strings, maps, and slices are traversed
// recursively so nested params get interpolated. Supports dot/index paths
// (e.g., ${user.name}, ${items.0}).
func (e *SimpleExecutor) interpolateInstruction(instr contracts.CompiledInstruction, state *flowState) contracts.CompiledInstruction {
	if state == nil || len(state.vars) == 0 || instr.Params == nil {
		return instr
	}
	if params, ok := interpolateValue(instr.Params, state).(map[string]any); ok {
		instr.Params = params
	}
	return instr
}

func (e *SimpleExecutor) interpolatePlanStep(step contracts.PlanStep, state *flowState) contracts.PlanStep {
	if state == nil || len(state.vars) == 0 || step.Params == nil {
		return step
	}
	if params, ok := interpolateValue(step.Params, state).(map[string]any); ok {
		step.Params = params
	}
	return step
}

// interpolateValue performs variable substitution recursively on any value.
// For strings, it supports type preservation when the entire value is a single interpolation.
func interpolateValue(v any, state *flowState) any {
	switch typed := v.(type) {
	case string:
		return interpolateStringTyped(typed, state)
	case map[string]any:
		clone := make(map[string]any, len(typed))
		for key, val := range typed {
			clone[key] = interpolateValue(val, state)
		}
		return clone
	case []any:
		out := make([]any, len(typed))
		for i, val := range typed {
			out[i] = interpolateValue(val, state)
		}
		return out
	default:
		return v
	}
}

// interpolateStringTyped performs interpolation with type preservation.
// If the string is a single interpolation (e.g., "${@store/count}"), the original
// type is preserved. If it contains mixed content, the result is always a string.
func interpolateStringTyped(s string, state *flowState) any {
	// Check if this is a single interpolation that should preserve type
	if isSingleInterpolation(s) {
		token := extractSingleToken(s)
		if resolved, ok := resolveTokenWithFallback(token, state); ok {
			return resolved
		}
		return "" // Return empty string for unresolved single interpolation
	}

	// For mixed content or no interpolation, return string
	return interpolateString(s, state)
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

// resolveTokenWithFallback resolves a token that may contain fallback chains.
// Syntax: @namespace/path|@namespace/path2|"default"
// Returns the first defined value or (nil, false) if all are undefined.
func resolveTokenWithFallback(token string, state *flowState) (any, bool) {
	alternatives := parseFallbackChain(token)
	for _, alt := range alternatives {
		if val, ok := resolveAlternative(alt, state); ok {
			return val, true
		}
	}
	return nil, false
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

// resolveAlternative resolves a single alternative (namespace reference or literal).
func resolveAlternative(alt string, state *flowState) (any, bool) {
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
	if i, err := strconv.Atoi(alt); err == nil {
		return i, true
	}
	if f, err := strconv.ParseFloat(alt, 64); err == nil {
		return f, true
	}

	// Check for namespace reference (@namespace/path)
	if strings.HasPrefix(alt, "@") {
		return resolveNamespacedToken(alt, state)
	}

	// Legacy: resolve against flat state (backward compatibility)
	return state.resolve(alt)
}

// resolveNamespacedToken resolves a @namespace/path reference.
func resolveNamespacedToken(token string, state *flowState) (any, bool) {
	if state == nil {
		return nil, false
	}

	// Strip leading @
	token = strings.TrimPrefix(token, "@")

	// Split into namespace and path
	slashIdx := strings.Index(token, "/")
	if slashIdx == -1 {
		// No path, just namespace - return entire namespace map
		if ns := state.getNamespace(token); ns != nil {
			return copyMap(ns), true
		}
		return nil, false
	}

	namespace := token[:slashIdx]
	path := token[slashIdx+1:]

	// If namespaced state exists, use it
	if state.namespaced != nil {
		return state.namespaced.resolveNamespaced(namespace, path)
	}

	// Fallback to legacy state for @store/ only
	if namespace == "store" {
		return state.resolve(path)
	}

	return nil, false
}

// getNamespace returns the namespace map for the given name.
// Used when resolving @namespace without a path.
func (s *flowState) getNamespace(namespace string) map[string]any {
	if s == nil {
		return nil
	}
	if s.namespaced != nil {
		return s.namespaced.getNamespace(namespace)
	}
	// Legacy fallback: only store is available
	if namespace == "store" {
		return s.vars
	}
	return nil
}

// interpolateString performs string interpolation, always returning a string.
// Supports both ${...} and {{...}} syntax with namespace and fallback support.
func interpolateString(s string, state *flowState) string {
	// Support both ${var} and {{var}} template syntax
	if !strings.Contains(s, "${") && !strings.Contains(s, "{{") {
		return s
	}

	out := s
	maxIterations := 100 // Prevent infinite loops
	for i := 0; i < maxIterations; i++ {
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

		if resolved, ok := resolveTokenWithFallback(token, state); ok {
			out = out[:start] + stringify(resolved) + out[end+len(suffix):]
		} else {
			// Drop unresolved token to avoid infinite loops
			out = out[:start] + out[end+len(suffix):]
		}
	}
	return out
}

// findMatchingClose finds the closing bracket, handling nested brackets.
func findMatchingClose(s, suffix string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		if strings.HasPrefix(s[i:], "${") || strings.HasPrefix(s[i:], "{{") {
			depth++
		}
		if strings.HasPrefix(s[i:], suffix) {
			if depth == 0 {
				return i
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

// evaluateExpression supports simple boolean expressions such as:
// - true/false literals
// - ${var} == 3
// - ${var} != "ok"
// - ${count} > 1
// Whitespace around tokens is allowed. Only a single binary comparison is supported today.
func evaluateExpression(expr string, state *flowState) (bool, bool) {
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

	left := resolveOperand(tokens[0], state)
	right := resolveOperand(tokens[2], state)
	return compareValues(left.val, right.val, tokens[1]), true
}

type operand struct {
	val any
}

func resolveOperand(token string, state *flowState) operand {
	token = strings.TrimSpace(token)
	if strings.HasPrefix(token, "${") && strings.HasSuffix(token, "}") {
		path := strings.TrimSuffix(strings.TrimPrefix(token, "${"), "}")
		if v, ok := state.resolve(path); ok {
			return operand{val: v}
		}
		return operand{}
	}
	if unquoted, ok := unquote(token); ok {
		return operand{val: unquoted}
	}
	if i, err := strconv.Atoi(token); err == nil {
		return operand{val: i}
	}
	if f, err := strconv.ParseFloat(token, 64); err == nil {
		return operand{val: f}
	}
	if b, err := strconv.ParseBool(token); err == nil {
		return operand{val: b}
	}
	return operand{val: token}
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
		for i := 0; i < len(tokens); i++ {
			switch tokens[i] {
			case "=", "!", ">", "<":
				if i+1 < len(tokens) && tokens[i+1] == "=" {
					reconstructed = append(reconstructed, tokens[i]+"=")
					i++
					continue
				}
			}
			reconstructed = append(reconstructed, tokens[i])
		}
		tokens = reconstructed
	}

	return tokens
}

func indexGraph(graph *contracts.PlanGraph) map[string]*contracts.PlanStep {
	index := make(map[string]*contracts.PlanStep)
	if graph == nil {
		return index
	}
	for i := range graph.Steps {
		step := &graph.Steps[i]
		index[step.NodeID] = step
	}
	return index
}

func firstStep(graph *contracts.PlanGraph) *contracts.PlanStep {
	if graph == nil || len(graph.Steps) == 0 {
		return nil
	}
	first := &graph.Steps[0]
	for i := range graph.Steps {
		if graph.Steps[i].Index < first.Index {
			first = &graph.Steps[i]
		}
	}
	return first
}

func maxGraphIndex(graph *contracts.PlanGraph) int {
	if graph == nil {
		return -1
	}
	maxIdx := -1
	for _, step := range graph.Steps {
		if step.Index > maxIdx {
			maxIdx = step.Index
		}
		if step.Loop != nil {
			if nested := maxGraphIndex(step.Loop); nested > maxIdx {
				maxIdx = nested
			}
		}
	}
	return maxIdx
}

func planStepToInstruction(step contracts.PlanStep) contracts.CompiledInstruction {
	return contracts.CompiledInstruction{
		Index:       step.Index,
		NodeID:      step.NodeID,
		Type:        step.Type,
		Params:      step.Params,
		PreloadHTML: step.Preload,
		Context:     step.Context,
		Metadata:    step.Metadata,
		Action:      step.Action, // NEW: Preserve typed Action field
	}
}

func planStepToInstructionStep(instr contracts.CompiledInstruction) contracts.PlanStep {
	return contracts.PlanStep{
		Index:    instr.Index,
		NodeID:   instr.NodeID,
		Type:     instr.Type,
		Params:   instr.Params,
		Metadata: instr.Metadata,
		Context:  instr.Context,
		Preload:  instr.PreloadHTML,
		Action:   instr.Action, // NEW: Preserve typed Action field
	}
}

func (e *SimpleExecutor) nextNodeID(step contracts.PlanStep, outcome contracts.StepOutcome) string {
	if len(step.Outgoing) == 0 {
		return ""
	}

	// Precedence: conditional outcome -> explicit failure path -> explicit success path -> first edge fallback.
	if next := conditionalBranchTarget(step, outcome); next != "" {
		return next
	}
	if next := failureBranchTarget(step, outcome); next != "" {
		return next
	}
	if next := successBranchTarget(step, outcome); next != "" {
		return next
	}

	return step.Outgoing[0].Target
}

func conditionalBranchTarget(step contracts.PlanStep, outcome contracts.StepOutcome) string {
	if !strings.EqualFold(step.Type, "conditional") || outcome.Condition == nil {
		return ""
	}

	if outcome.Condition.Outcome {
		return findEdgeTarget(step.Outgoing, []string{"true", "yes", "success", "ok", "pass"})
	}
	return findEdgeTarget(step.Outgoing, []string{"false", "no", "fail", "failure"})
}

func failureBranchTarget(step contracts.PlanStep, outcome contracts.StepOutcome) string {
	if outcome.Failure == nil {
		return ""
	}
	return findEdgeTarget(step.Outgoing, []string{"failure", "error", "fail"})
}

func successBranchTarget(step contracts.PlanStep, outcome contracts.StepOutcome) string {
	if !outcome.Success {
		return ""
	}
	return findEdgeTarget(step.Outgoing, []string{"success", "ok", "pass", ""})
}

func findEdgeTarget(edges []contracts.PlanEdge, conditions []string) string {
	for _, edge := range edges {
		if matchesCondition(edge.Condition, conditions) {
			return edge.Target
		}
	}
	return ""
}

func matchesCondition(condition string, candidates []string) bool {
	normalized := normalizeEdgeCondition(condition)
	for _, candidate := range candidates {
		if normalized == candidate {
			return true
		}
	}
	return false
}

func normalizeEdgeCondition(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func stringValue(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// intValue safely extracts an integer from params, supporting int, int64, float64, float32, and string types.
// Returns 0 if the key is missing or the value cannot be coerced.
func intValue(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	return coerceToInt(v)
}

// coerceToInt converts various types to int, handling JSON's float64 numbers and strings.
// This is the central type coercion for integer params, addressing assumption A8 in ASSUMPTION_MAPPING.md.
func coerceToInt(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int8:
		return int(t)
	case int16:
		return int(t)
	case int32:
		return int(t)
	case int64:
		return int(t)
	case uint:
		return int(t)
	case uint8:
		return int(t)
	case uint16:
		return int(t)
	case uint32:
		return int(t)
	case uint64:
		return int(t)
	case float32:
		return int(t)
	case float64:
		return int(t)
	case string:
		// Support string-encoded numbers (e.g., "1000" from form data)
		if i, err := strconv.Atoi(t); err == nil {
			return i
		}
		// Try parsing as float then truncating (e.g., "1.5" -> 1)
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return int(f)
		}
	case json.Number:
		if i, err := t.Int64(); err == nil {
			return int(i)
		}
	}
	return 0
}

// floatValue safely extracts a float64 from params.
func floatValue(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok {
		return 0
	}
	f, _ := coerceToFloat(v)
	return f
}

// coerceToFloat converts various types to float64.
func coerceToFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case int32:
		return float64(t), true
	case string:
		if f, err := strconv.ParseFloat(t, 64); err == nil {
			return f, true
		}
	case json.Number:
		if f, err := t.Float64(); err == nil {
			return f, true
		}
	}
	return 0, false
}

// boolValue safely extracts a boolean from params.
func boolValue(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key]
	if !ok {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, _ := strconv.ParseBool(t)
		return b
	case int:
		return t != 0
	case float64:
		return t != 0
	}
	return false
}

func intPtr(v int) *int {
	return &v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func uuidOrDefault(value, fallback uuid.UUID) uuid.UUID {
	if value != uuid.Nil {
		return value
	}
	return fallback
}

func coerceArray(raw any) []any {
	switch v := raw.(type) {
	case []any:
		return v
	case []string:
		out := make([]any, len(v))
		for i := range v {
			out[i] = v[i]
		}
		return out
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil
		}
		var decoded []any
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			return decoded
		}
	}
	return nil
}

func firstPresent(params map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := params[k]; ok {
			return v
		}
	}
	return nil
}

func normalizeVariableValue(value any, valueType string) any {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "boolean", "bool":
		if b, err := strconv.ParseBool(fmt.Sprint(value)); err == nil {
			return b
		}
	case "number", "float", "int":
		if f, ok := toFloat(value); ok {
			if strings.Contains(strings.ToLower(valueType), "int") {
				return int(f)
			}
			return f
		}
	case "json":
		if s, ok := value.(string); ok {
			var decoded any
			if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &decoded); err == nil {
				return decoded
			}
		}
	}
	return value
}
