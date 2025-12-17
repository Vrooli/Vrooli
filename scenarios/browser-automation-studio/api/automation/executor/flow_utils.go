package executor

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/state"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// flow_utils.go holds small helpers for graph traversal and value coercion.
//
// MIGRATION NOTE: State management has been extracted to the automation/state package.
// The flowState type below is now a thin wrapper around state.ExecutionState for
// backward compatibility. New code should use state.ExecutionState directly.
//
// See: automation/state/execution_state.go for the canonical implementation.

// executionState is an alias to the canonical state.ExecutionState type.
// Deprecated: Use state.ExecutionState directly for new code.
type executionState = state.ExecutionState

// newExecutionState creates a new ExecutionState with the given initial values.
// Deprecated: Use state.New() directly for new code.
func newExecutionState(initialStore, initialParams, env map[string]any) *executionState {
	return state.New(initialStore, initialParams, env)
}

// resolvePath resolves a dot-notation path against a map.
// Deprecated: This is now available in the state package.
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
// Deprecated: Use state.ExecutionState directly for new code.
//
// MIGRATION STRATEGY:
// flowState now wraps state.ExecutionState internally. The legacy 'vars' map
// is no longer used directly; all operations delegate to the underlying
// ExecutionState. This provides backward compatibility while the codebase
// migrates to using state.ExecutionState directly.
type flowState struct {
	// state is the underlying namespace-aware state management.
	// All operations delegate to this instance.
	state *state.ExecutionState
}

// newFlowState creates a flowState from seed values.
// Deprecated: Use state.NewFromStore() directly for new code.
func newFlowState(seed map[string]any) *flowState {
	return &flowState{
		state: state.NewFromStore(seed),
	}
}

// newFlowStateWithNamespaces creates a flowState with namespace-aware state management.
// Deprecated: Use state.New() directly for new code.
func newFlowStateWithNamespaces(initialStore, initialParams, env map[string]any) *flowState {
	return &flowState{
		state: state.New(initialStore, initialParams, env),
	}
}

// hasNamespaces returns true if this flowState has namespace-aware state.
// Always returns true for the refactored implementation.
func (s *flowState) hasNamespaces() bool {
	return s != nil && s.state != nil
}

// ExecutionState returns the underlying state.ExecutionState.
// Use this when migrating code to use the new state package directly.
func (s *flowState) ExecutionState() *state.ExecutionState {
	if s == nil {
		return nil
	}
	return s.state
}

func (s *flowState) markEntryChecked() {
	if s == nil || s.state == nil {
		return
	}
	s.state.MarkEntryChecked()
}

func (s *flowState) hasCheckedEntry() bool {
	if s == nil || s.state == nil {
		return false
	}
	return s.state.HasCheckedEntry()
}

func (s *flowState) get(key string) (any, bool) {
	if s == nil || s.state == nil {
		return nil, false
	}
	return s.state.Get(key)
}

// resolve supports dot/index path resolution (e.g., user.name, items.0) and
// falls back to a direct lookup for raw keys.
func (s *flowState) resolve(path string) (any, bool) {
	if s == nil || s.state == nil {
		return nil, false
	}
	return s.state.Resolve(path)
}

func (s *flowState) set(key string, value any) {
	if s == nil || s.state == nil {
		return
	}
	s.state.Set(key, value)
}

// setNamespaced sets a value in the specified namespace.
// Only @store/ is writable; attempts to write to @params/ or @env/ are silently ignored.
func (s *flowState) setNamespaced(namespace, key string, value any) {
	if s == nil || s.state == nil {
		return
	}
	if namespace != "store" {
		return // Only store is writable
	}
	s.state.Set(key, value)
}

func (s *flowState) merge(vars map[string]any) {
	if s == nil || s.state == nil || vars == nil {
		return
	}
	s.state.Merge(vars)
}

// mergeNamespacedStore merges vars into the @store/ namespace.
func (s *flowState) mergeNamespacedStore(vars map[string]any) {
	if s == nil || s.state == nil || vars == nil {
		return
	}
	s.state.Merge(vars)
}

// getNamespacedState returns the underlying executionState.
// Deprecated: Use ExecutionState() instead.
func (s *flowState) getNamespacedState() *executionState {
	if s == nil {
		return nil
	}
	return s.state
}

func (s *flowState) setNextIndexFromPlan(plan contracts.ExecutionPlan) {
	if s == nil || s.state == nil {
		return
	}
	s.state.SetNextIndexFromPlan(plan)
}

func (s *flowState) allocateIndexRange(count int) int {
	if s == nil || s.state == nil {
		return 0
	}
	return s.state.AllocateIndexRange(count)
}

// getNamespace returns the namespace map for the given name.
// Used when resolving @namespace without a path.
func (s *flowState) getNamespace(namespace string) map[string]any {
	if s == nil || s.state == nil {
		return nil
	}
	return s.state.GetNamespace(namespace)
}

// extractLoopItems extracts loop items from params.
// Deprecated: Use state.ExtractLoopItems() for new code.
func extractLoopItems(params map[string]any, fs *flowState) []any {
	if fs == nil || fs.state == nil {
		return nil
	}
	return state.ExtractLoopItems(params, fs.state)
}

// evaluateLoopCondition evaluates a loop condition.
// Deprecated: Use state.EvaluateLoopCondition() for new code.
func evaluateLoopCondition(params map[string]any, fs *flowState) bool {
	if fs == nil || fs.state == nil {
		return false
	}
	return state.EvaluateLoopCondition(params, fs.state)
}

// compareValues compares two values using the specified operator.
// Deprecated: Use state.CompareValues() for new code.
func compareValues(current any, expected any, op string) bool {
	return state.CompareValues(current, expected, op)
}

// toFloat converts a value to float64 if possible.
// Deprecated: Use state.ToFloat() for new code.
func toFloat(v any) (float64, bool) {
	return state.ToFloat(v)
}

// interpolateInstruction performs variable substitution on instruction
// params/strings using ${var} tokens.
// Deprecated: Use state.NewInterpolator(fs.state).InterpolateInstruction() for new code.
func (e *SimpleExecutor) interpolateInstruction(instr contracts.CompiledInstruction, fs *flowState) contracts.CompiledInstruction {
	if fs == nil || fs.state == nil {
		return instr
	}
	interp := state.NewInterpolator(fs.state)
	return interp.InterpolateInstruction(instr)
}

// interpolatePlanStep performs variable substitution on a plan step.
// Deprecated: Use state.NewInterpolator(fs.state).InterpolatePlanStep() for new code.
func (e *SimpleExecutor) interpolatePlanStep(step contracts.PlanStep, fs *flowState) contracts.PlanStep {
	if fs == nil || fs.state == nil {
		return step
	}
	interp := state.NewInterpolator(fs.state)
	return interp.InterpolatePlanStep(step)
}

// interpolateActionDefinition is kept for backward compatibility.
// Deprecated: Use state.NewInterpolator().
func interpolateActionDefinition(action *basactions.ActionDefinition, fs *flowState) *basactions.ActionDefinition {
	if action == nil || fs == nil || fs.state == nil {
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

	interp := state.NewInterpolator(fs.state)
	updated := interp.InterpolateValue(doc)
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

// interpolateValue performs variable substitution recursively on any value.
// Deprecated: Use state.NewInterpolator(fs.state).InterpolateValue() for new code.
func interpolateValue(v any, fs *flowState) any {
	if fs == nil || fs.state == nil {
		return v
	}
	interp := state.NewInterpolator(fs.state)
	return interp.InterpolateValue(v)
}

// interpolateStringTyped performs interpolation with type preservation.
// Deprecated: Use state.NewInterpolator(fs.state).InterpolateValue() for new code.
func interpolateStringTyped(s string, fs *flowState) any {
	if fs == nil || fs.state == nil {
		return s
	}
	interp := state.NewInterpolator(fs.state)
	return interp.InterpolateValue(s)
}

// isSingleInterpolation returns true if the string contains exactly one interpolation
// that spans the entire value (after trimming).
// Deprecated: This is now internal to the state package.
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
// Deprecated: This is now internal to the state package.
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

// interpolateString performs string interpolation, always returning a string.
// Deprecated: Use state.NewInterpolator(fs.state).InterpolateString() for new code.
func interpolateString(s string, fs *flowState) string {
	if fs == nil || fs.state == nil {
		return s
	}
	interp := state.NewInterpolator(fs.state)
	return interp.InterpolateString(s)
}

// evaluateExpression supports simple boolean expressions.
// Deprecated: Use state.NewInterpolator(fs.state).EvaluateExpression() for new code.
func evaluateExpression(expr string, fs *flowState) (bool, bool) {
	if fs == nil || fs.state == nil {
		return false, false
	}
	interp := state.NewInterpolator(fs.state)
	return interp.EvaluateExpression(expr)
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
	instrType := step.Type
	instrParams := step.Params
	if step.Action != nil && step.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		instrType = ""
		instrParams = nil
	}
	return contracts.CompiledInstruction{
		Index:       step.Index,
		NodeID:      step.NodeID,
		Type:        instrType,
		Params:      instrParams,
		PreloadHTML: step.Preload,
		Context:     step.Context,
		Metadata:    step.Metadata,
		Action:      step.Action, // NEW: Preserve typed Action field
	}
}

func planStepToInstructionStep(instr contracts.CompiledInstruction) contracts.PlanStep {
	stepType := instr.Type
	stepParams := instr.Params
	if instr.Action != nil && instr.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		stepType = ""
		stepParams = nil
	}
	return contracts.PlanStep{
		Index:    instr.Index,
		NodeID:   instr.NodeID,
		Type:     stepType,
		Params:   stepParams,
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
	isConditional := strings.EqualFold(step.Type, "conditional") || (step.Action != nil && step.Action.Type == basactions.ActionType_ACTION_TYPE_CONDITIONAL)
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

// stringValue safely extracts a string from params.
// Deprecated: Use state.StringValue() for new code.
func stringValue(m map[string]any, key string) string {
	return state.StringValue(m, key)
}

// intValue safely extracts an integer from params.
// Deprecated: Use state.IntValue() for new code.
func intValue(m map[string]any, key string) int {
	return state.IntValue(m, key)
}

// coerceToInt converts various types to int.
// Deprecated: Use state.CoerceToInt() for new code.
func coerceToInt(v any) int {
	return state.CoerceToInt(v)
}

// floatValue safely extracts a float64 from params.
// Deprecated: Use state.FloatValue() for new code.
func floatValue(m map[string]any, key string) float64 {
	return state.FloatValue(m, key)
}

// coerceToFloat converts various types to float64.
// Deprecated: Use state.CoerceToFloat() for new code.
func coerceToFloat(v any) (float64, bool) {
	return state.CoerceToFloat(v)
}

// boolValue safely extracts a boolean from params.
// Deprecated: Use state.BoolValue() for new code.
func boolValue(m map[string]any, key string) bool {
	return state.BoolValue(m, key)
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

// coerceArray converts various types to []any.
// Deprecated: Use state.CoerceArray() for new code.
func coerceArray(raw any) []any {
	return state.CoerceArray(raw)
}

// normalizeVariableValue normalizes a variable value based on its declared type.
// Deprecated: Use state.NormalizeVariableValue() for new code.
func normalizeVariableValue(value any, valueType string) any {
	return state.NormalizeVariableValue(value, valueType)
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
// These helpers support the Type/Params → Action migration by preferring
// the typed Action field when available, falling back to deprecated Type/Params.

// InstructionStepType returns the step type string from a CompiledInstruction.
// Prefers Action.Type over the deprecated Type field.
func InstructionStepType(instr contracts.CompiledInstruction) string {
	if instr.Action != nil && instr.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return actionTypeToString(instr.Action.Type)
	}
	// Deprecated: Using legacy Type field. Migrate to Action field.
	if instr.Type != "" {
		DeprecationTracker.RecordTypeUsage("InstructionStepType:" + instr.NodeID)
	}
	return instr.Type
}

// PlanStepType returns the step type string from a PlanStep.
// Prefers Action.Type over the deprecated Type field.
func PlanStepType(step contracts.PlanStep) string {
	if step.Action != nil && step.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return actionTypeToString(step.Action.Type)
	}
	// Deprecated: Using legacy Type field. Migrate to Action field.
	if step.Type != "" {
		DeprecationTracker.RecordTypeUsage("PlanStepType:" + step.NodeID)
	}
	return step.Type
}

// InstructionParams returns the params map from a CompiledInstruction.
// If Action is set, extracts params from the typed action; otherwise uses deprecated Params.
func InstructionParams(instr contracts.CompiledInstruction) map[string]any {
	if instr.Action != nil && instr.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return actionToParams(instr.Action)
	}
	// Deprecated: Using legacy Params field. Migrate to Action field.
	if len(instr.Params) > 0 {
		DeprecationTracker.RecordParamsUsage("InstructionParams:" + instr.NodeID)
	}
	return instr.Params
}

// PlanStepParams returns the params map from a PlanStep.
// If Action is set, extracts params from the typed action; otherwise uses deprecated Params.
func PlanStepParams(step contracts.PlanStep) map[string]any {
	if step.Action != nil && step.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return actionToParams(step.Action)
	}
	// Deprecated: Using legacy Params field. Migrate to Action field.
	if len(step.Params) > 0 {
		DeprecationTracker.RecordParamsUsage("PlanStepParams:" + step.NodeID)
	}
	return step.Params
}

// IsActionType checks if an instruction matches the given action type.
// Supports both enum and string comparisons.
func IsActionType(instr contracts.CompiledInstruction, actionType basactions.ActionType) bool {
	if instr.Action != nil && instr.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return instr.Action.Type == actionType
	}
	// Deprecated: Falling back to string comparison on legacy Type field.
	if instr.Type != "" {
		DeprecationTracker.RecordTypeUsage("IsActionType:" + instr.NodeID)
	}
	return strings.EqualFold(strings.TrimSpace(instr.Type), actionTypeToString(actionType))
}

// IsPlanStepActionType checks if a plan step matches the given action type.
func IsPlanStepActionType(step contracts.PlanStep, actionType basactions.ActionType) bool {
	if step.Action != nil && step.Action.Type != basactions.ActionType_ACTION_TYPE_UNSPECIFIED {
		return step.Action.Type == actionType
	}
	// Deprecated: Falling back to string comparison on legacy Type field.
	if step.Type != "" {
		DeprecationTracker.RecordTypeUsage("IsPlanStepActionType:" + step.NodeID)
	}
	return strings.EqualFold(strings.TrimSpace(step.Type), actionTypeToString(actionType))
}

// actionTypeToString converts an ActionType enum to its string representation.
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
					result[nk] = nv
				}
				continue
			}
		}
		result[k] = v
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
