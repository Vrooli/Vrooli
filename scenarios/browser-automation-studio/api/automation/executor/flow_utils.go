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

type flowState struct {
	vars         map[string]any
	nextIndex    int
	entryChecked bool
}

func newFlowState(seed map[string]any) *flowState {
	state := &flowState{vars: map[string]any{}}
	for k, v := range seed {
		state.vars[k] = v
	}
	return state
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
}

func (s *flowState) merge(vars map[string]any) {
	if s == nil || vars == nil {
		return
	}
	for k, v := range vars {
		s.set(k, v)
	}
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

func interpolateValue(v any, state *flowState) any {
	switch typed := v.(type) {
	case string:
		return interpolateString(typed, state)
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

func interpolateString(s string, state *flowState) string {
	// Support both ${var} and {{var}} template syntax
	if !strings.Contains(s, "${") && !strings.Contains(s, "{{") {
		return s
	}

	out := s
	for {
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

		end := strings.Index(out[start+len(prefix):], suffix)
		if end == -1 {
			break
		}
		end = start + len(prefix) + end
		token := out[start+len(prefix) : end]

		if resolved, ok := state.resolve(token); ok {
			out = out[:start] + stringify(resolved) + out[end+len(suffix):]
		} else {
			// Drop unresolved token to avoid infinite loops.
			out = out[:start] + out[end+len(suffix):]
		}
	}
	return out
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
	}
}

func (e *SimpleExecutor) nextNodeID(step contracts.PlanStep, outcome contracts.StepOutcome) string {
	if len(step.Outgoing) == 0 {
		return ""
	}

	normalized := func(value string) string {
		return strings.ToLower(strings.TrimSpace(value))
	}

	if strings.EqualFold(step.Type, "conditional") && outcome.Condition != nil {
		targetValue := "false"
		if outcome.Condition.Outcome {
			targetValue = "true"
		}
		for _, edge := range step.Outgoing {
			cond := normalized(edge.Condition)
			switch cond {
			case targetValue:
				return edge.Target
			case "yes", "success", "ok", "pass":
				if targetValue == "true" {
					return edge.Target
				}
			case "no", "fail", "failure":
				if targetValue == "false" {
					return edge.Target
				}
			}
		}
	}

	if outcome.Failure != nil {
		for _, edge := range step.Outgoing {
			cond := normalized(edge.Condition)
			if cond == "failure" || cond == "error" || cond == "fail" {
				return edge.Target
			}
		}
	}

	if outcome.Success {
		for _, edge := range step.Outgoing {
			cond := normalized(edge.Condition)
			if cond == "success" || cond == "ok" || cond == "pass" || cond == "" {
				return edge.Target
			}
		}
	}

	// Default: first edge wins.
	return step.Outgoing[0].Target
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

func intValue(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case int:
			return t
		case int64:
			return int(t)
		case float64:
			return int(t)
		case float32:
			return int(t)
		}
	}
	return 0
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
