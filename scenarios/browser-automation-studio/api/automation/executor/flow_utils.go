package executor

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// flow_utils.go holds small helpers for graph traversal and value coercion.

type flowState struct {
	vars map[string]any
}

func newFlowState(seed map[string]any) *flowState {
	state := &flowState{vars: map[string]any{}}
	for k, v := range seed {
		state.vars[k] = v
	}
	return state
}

func (s *flowState) get(key string) (any, bool) {
	if s == nil {
		return nil, false
	}
	v, ok := s.vars[key]
	return v, ok
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

func extractLoopItems(params map[string]any, state *flowState) []any {
	if params == nil {
		return nil
	}
	if raw, ok := params["loopItems"]; ok {
		if arr, ok := raw.([]any); ok {
			return arr
		}
	}
	if raw, ok := params["items"]; ok {
		if arr, ok := raw.([]any); ok {
			return arr
		}
	}
	if raw, ok := params["arraySource"]; ok {
		if name, ok := raw.(string); ok && name != "" {
			if v, ok := state.get(name); ok {
				if arr, ok := v.([]any); ok {
					return arr
				}
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
		condType = "variable"
	}

	switch condType {
	case "variable":
		name := stringValue(params, "conditionVariable")
		if name == "" {
			return false
		}
		op := strings.ToLower(strings.TrimSpace(stringValue(params, "conditionOperator")))
		expected, _ := params["conditionValue"]
		current, ok := state.get(name)
		if !ok {
			return false
		}
		return compareValues(current, expected, op)
	case "expression":
		// Placeholder: expression evaluation not yet implemented; treat non-empty expression as true.
		return strings.TrimSpace(stringValue(params, "conditionExpression")) != ""
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

// interpolateInstruction performs simple variable substitution on instruction
// params/strings using ${var} tokens. Only string values are interpolated.
func (e *SimpleExecutor) interpolateInstruction(instr contracts.CompiledInstruction, state *flowState) contracts.CompiledInstruction {
	if state == nil || len(state.vars) == 0 || instr.Params == nil {
		return instr
	}
	clone := make(map[string]any, len(instr.Params))
	for k, v := range instr.Params {
		clone[k] = interpolateValue(v, state)
	}
	instr.Params = clone
	return instr
}

func interpolateValue(v any, state *flowState) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	if !strings.Contains(s, "${") {
		return s
	}
	out := s
	for key, val := range state.vars {
		placeholder := "${" + key + "}"
		if strings.Contains(out, placeholder) {
			out = strings.ReplaceAll(out, placeholder, fmt.Sprint(val))
		}
	}
	return out
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

func uuidOrDefault(value, fallback uuid.UUID) uuid.UUID {
	if value != uuid.Nil {
		return value
	}
	return fallback
}
