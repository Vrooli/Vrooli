package executor

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/state"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	"google.golang.org/protobuf/encoding/protojson"
)

// flow_utils.go holds helpers for graph traversal, instruction processing,
// and action type conversion.
//
// NOTE: State management has been moved to the automation/state package.
// See: automation/state/execution_state.go for the canonical implementation.

// interpolateInstruction performs variable substitution on instruction
// params/strings using ${var} tokens.
func (e *SimpleExecutor) interpolateInstruction(instr contracts.CompiledInstruction, execState *state.ExecutionState) contracts.CompiledInstruction {
	if execState == nil {
		return instr
	}
	interp := state.NewInterpolator(execState)
	return interp.InterpolateInstruction(instr)
}

// interpolatePlanStep performs variable substitution on a plan step.
func (e *SimpleExecutor) interpolatePlanStep(step contracts.PlanStep, execState *state.ExecutionState) contracts.PlanStep {
	if execState == nil {
		return step
	}
	interp := state.NewInterpolator(execState)
	return interp.InterpolatePlanStep(step)
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
		PreloadHTML: step.Preload,
		Context:     step.Context,
		Metadata:    step.Metadata,
		Action:      step.Action,
	}
}

func planStepToInstructionStep(instr contracts.CompiledInstruction) contracts.PlanStep {
	return contracts.PlanStep{
		Index:    instr.Index,
		NodeID:   instr.NodeID,
		Metadata: instr.Metadata,
		Context:  instr.Context,
		Preload:  instr.PreloadHTML,
		Action:   instr.Action,
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
	isConditional := step.Action != nil && step.Action.Type == basactions.ActionType_ACTION_TYPE_CONDITIONAL
	if !isConditional || outcome.Condition == nil {
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

// firstPresent returns the first present value for the given keys.
func firstPresent(params map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := params[k]; ok {
			return v
		}
	}
	return nil
}

// =============================================================================
// ACTION-AWARE STEP TYPE HELPERS
// =============================================================================
// These helpers extract step type and params from the typed Action field.

// InstructionStepType returns the step type string from a CompiledInstruction.
func InstructionStepType(instr contracts.CompiledInstruction) string {
	if instr.Action != nil && instr.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return actionTypeToString(instr.Action.Type)
	}
	return ""
}

// PlanStepType returns the step type string from a PlanStep.
func PlanStepType(step contracts.PlanStep) string {
	if step.Action != nil && step.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return actionTypeToString(step.Action.Type)
	}
	return ""
}

// InstructionParams returns the params map from a CompiledInstruction.
func InstructionParams(instr contracts.CompiledInstruction) map[string]any {
	if instr.Action != nil && instr.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return actionToParams(instr.Action)
	}
	return nil
}

// PlanStepParams returns the params map from a PlanStep.
func PlanStepParams(step contracts.PlanStep) map[string]any {
	if step.Action != nil && step.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return actionToParams(step.Action)
	}
	return nil
}

// IsActionType checks if an instruction matches the given action type.
func IsActionType(instr contracts.CompiledInstruction, actionType basactions.ActionType) bool {
	if instr.Action != nil {
		return instr.Action.Type == actionType
	}
	return false
}

// IsPlanStepActionType checks if a plan step matches the given action type.
func IsPlanStepActionType(step contracts.PlanStep, actionType basactions.ActionType) bool {
	if step.Action != nil {
		return step.Action.Type == actionType
	}
	return false
}

// actionTypeToString converts an ActionType enum to its canonical string representation.
// This matches the strings defined in automation/actions (e.g., "type" for ACTION_TYPE_INPUT).
func actionTypeToString(at basactions.ActionType) string {
	switch at {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		return "navigate"
	case basactions.ActionType_ACTION_TYPE_CLICK:
		return "click"
	case basactions.ActionType_ACTION_TYPE_INPUT:
		return "type"
	case basactions.ActionType_ACTION_TYPE_WAIT:
		return "wait"
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		return "assert"
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		return "scroll"
	case basactions.ActionType_ACTION_TYPE_SELECT:
		return "select"
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		return "evaluate"
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		return "keyboard"
	case basactions.ActionType_ACTION_TYPE_HOVER:
		return "hover"
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		return "screenshot"
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		return "focus"
	case basactions.ActionType_ACTION_TYPE_BLUR:
		return "blur"
	case basactions.ActionType_ACTION_TYPE_SUBFLOW:
		return "subflow"
	case basactions.ActionType_ACTION_TYPE_EXTRACT:
		return "extract"
	case basactions.ActionType_ACTION_TYPE_UPLOAD_FILE:
		return "uploadFile"
	case basactions.ActionType_ACTION_TYPE_DOWNLOAD:
		return "download"
	case basactions.ActionType_ACTION_TYPE_FRAME_SWITCH:
		return "frameSwitch"
	case basactions.ActionType_ACTION_TYPE_TAB_SWITCH:
		return "tabSwitch"
	case basactions.ActionType_ACTION_TYPE_COOKIE_STORAGE:
		return "setCookie"
	case basactions.ActionType_ACTION_TYPE_SHORTCUT:
		return "shortcut"
	case basactions.ActionType_ACTION_TYPE_DRAG_DROP:
		return "dragDrop"
	case basactions.ActionType_ACTION_TYPE_GESTURE:
		return "gesture"
	case basactions.ActionType_ACTION_TYPE_NETWORK_MOCK:
		return "networkMock"
	case basactions.ActionType_ACTION_TYPE_ROTATE:
		return "rotate"
	case basactions.ActionType_ACTION_TYPE_SET_VARIABLE:
		return "setVariable"
	case basactions.ActionType_ACTION_TYPE_LOOP:
		return "loop"
	case basactions.ActionType_ACTION_TYPE_CONDITIONAL:
		return "conditional"
	default:
		return "custom"
	}
}

// actionToParams extracts params from a typed ActionDefinition into a map.
// This provides compatibility with code that expects map[string]any params.
func actionToParams(action *basactions.ActionDefinition) map[string]any {
	if action == nil {
		return nil
	}

	// Use protojson to convert to JSON, then unmarshal to map for generic access
	data, err := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}.Marshal(action)
	if err != nil {
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}

	// Flatten action-specific params (e.g., action.click.selector → selector)
	return flattenActionParams(result)
}

// flattenActionParams flattens nested action params to top level.
// e.g., {"click": {"selector": "x"}} → {"selector": "x"}
// Also normalizes proto field names (snake_case) to executor-expected names (camelCase).
func flattenActionParams(m map[string]any) map[string]any {
	result := make(map[string]any)

	// Copy top-level fields
	for k, v := range m {
		if k == "type" || k == "metadata" {
			continue // Skip type enum and metadata
		}
		// If this is an action-specific nested object, flatten it
		if nested, ok := v.(map[string]any); ok {
			// Check if this is an action params object (e.g., click, input, navigate)
			if isActionParamsKey(k) {
				for nk, nv := range nested {
					// Normalize proto field names to executor-expected names
					normalizedKey, normalizedValue := normalizeProtoParam(k, nk, nv)
					result[normalizedKey] = normalizedValue
				}
				continue
			}
		}
		result[k] = v
	}

	return result
}

// normalizeProtoParam converts proto field names to executor-expected names.
// This handles the mismatch between proto snake_case and executor camelCase conventions.
func normalizeProtoParam(actionKey, paramKey string, value any) (string, any) {
	// Loop action normalizations
	if actionKey == "loop" {
		return normalizeLoopParam(paramKey, value)
	}
	// Subflow action normalizations
	if actionKey == "subflow" {
		return normalizeSubflowParam(paramKey, value)
	}
	// Default: convert snake_case to camelCase
	return snakeToCamel(paramKey), value
}

// normalizeLoopParam normalizes loop-specific proto params to executor-expected format.
func normalizeLoopParam(key string, value any) (string, any) {
	switch key {
	case "loop_type":
		// Convert "LOOP_TYPE_FOREACH" → "foreach"
		if s, ok := value.(string); ok {
			normalized := strings.ToLower(strings.TrimPrefix(s, "LOOP_TYPE_"))
			return "loopType", normalized
		}
		return "loopType", value
	case "count":
		return "loopCount", value
	case "max_iterations":
		return "loopMaxIterations", value
	case "array_source":
		return "arraySource", value
	case "item_variable":
		return "itemVariable", value
	case "index_variable":
		return "indexVariable", value
	case "iteration_timeout_ms":
		return "iterationTimeoutMs", value
	case "total_timeout_ms":
		return "totalTimeoutMs", value
	case "condition":
		// Normalize nested condition object
		if cond, ok := value.(map[string]any); ok {
			return "condition", normalizeLoopCondition(cond)
		}
		return "condition", value
	default:
		return snakeToCamel(key), value
	}
}

// normalizeLoopCondition normalizes loop condition params.
func normalizeLoopCondition(cond map[string]any) map[string]any {
	result := make(map[string]any, len(cond))
	for k, v := range cond {
		switch k {
		case "type":
			// Convert "LOOP_CONDITION_TYPE_VARIABLE" → "variable"
			if s, ok := v.(string); ok {
				result["conditionType"] = strings.ToLower(strings.TrimPrefix(s, "LOOP_CONDITION_TYPE_"))
			} else {
				result["conditionType"] = v
			}
		case "operator":
			// Convert "LOOP_CONDITION_OPERATOR_EQUALS" → "equals"
			if s, ok := v.(string); ok {
				result["conditionOperator"] = strings.ToLower(strings.TrimPrefix(s, "LOOP_CONDITION_OPERATOR_"))
			} else {
				result["conditionOperator"] = v
			}
		case "variable":
			result["conditionVariable"] = v
		case "value":
			result["conditionValue"] = v
		case "expression":
			result["conditionExpression"] = v
		default:
			result[snakeToCamel(k)] = v
		}
	}
	return result
}

// normalizeSubflowParam normalizes subflow-specific proto params.
func normalizeSubflowParam(key string, value any) (string, any) {
	switch key {
	case "workflow_id":
		return "workflowId", value
	case "workflow_path":
		return "workflowPath", value
	case "workflow_version":
		return "workflowVersion", value
	case "workflow_definition":
		return "workflowDefinition", value
	case "args":
		// Convert args to parameters for executor compatibility
		return "parameters", value
	default:
		return snakeToCamel(key), value
	}
}

// snakeToCamel converts snake_case to camelCase.
func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	if len(parts) == 1 {
		return s
	}
	result := parts[0]
	for _, part := range parts[1:] {
		if len(part) > 0 {
			result += strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return result
}

// isActionParamsKey returns true if the key is an action-specific params field.
func isActionParamsKey(key string) bool {
	actionKeys := []string{
		"navigate", "click", "input", "wait", "assert", "scroll",
		"select_option", "evaluate", "keyboard", "hover", "screenshot",
		"focus", "blur", "subflow", "extract", "upload_file", "download",
		"frame_switch", "tab_switch", "cookie_storage", "shortcut",
		"drag_drop", "gesture", "network_mock", "rotate",
		"set_variable", "loop", "conditional",
	}
	for _, k := range actionKeys {
		if key == k {
			return true
		}
	}
	return false
}
