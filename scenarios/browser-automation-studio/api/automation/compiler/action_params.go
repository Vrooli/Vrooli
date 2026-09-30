// Parameter builder functions for converting map-based data to typed proto parameter messages.
// Consolidates parameter building logic that was previously in internal/params.
//
// Field Aliases (both names are actively used and supported):
// - InputParams: "text" and "value" are equivalent (docs/UI use "text", proto uses "value")
// - WaitParams: "duration" and "durationMs" are equivalent
// - AssertParams: "assertMode" and "mode" are equivalent
package compiler

import (
	contractvalues "github.com/vrooli/browser-automation-studio/automation/contracts"
	"strings"

	"github.com/vrooli/browser-automation-studio/internal/enums"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func generatedEnumValue(values map[string]int32, input string) int32 {
	normalized := strings.ToUpper(strings.TrimSpace(input))
	compact := strings.ReplaceAll(strings.ReplaceAll(normalized, "_", ""), "-", "")
	for name, value := range values {
		parts := strings.Split(strings.ToUpper(name), "_")
		for start := range parts {
			if strings.Join(parts[start:], "") == compact {
				return value
			}
		}
	}
	return 0
}

// StringToNavigateWaitEvent converts a string to NavigateWaitEvent enum.
func StringToNavigateWaitEvent(s string) basactions.NavigateWaitEvent {
	return basactions.NavigateWaitEvent(generatedEnumValue(basactions.NavigateWaitEvent_value, s))
}

// Note: StringToMouseButton, StringToKeyboardModifier, StringToAssertionMode, and
// StringToSelectorType are defined in internal/enums package.

// StringToWaitState converts a string to WaitState enum.
func StringToWaitState(s string) basactions.WaitState {
	return basactions.WaitState(generatedEnumValue(basactions.WaitState_value, s))
}

// Note: StringToAssertionMode is defined in primitives.go with input normalization.

// StringToScrollBehavior converts a string to ScrollBehavior enum.
func StringToScrollBehavior(s string) basactions.ScrollBehavior {
	return basactions.ScrollBehavior(generatedEnumValue(basactions.ScrollBehavior_value, s))
}

// StringToKeyAction converts a string to KeyAction enum.
func StringToKeyAction(s string) basactions.KeyAction {
	return basactions.KeyAction(generatedEnumValue(basactions.KeyAction_value, s))
}

// BuildNavigateParams converts a data map to NavigateParams proto.
func BuildNavigateParams(data map[string]any) *basactions.NavigateParams {
	p := &basactions.NavigateParams{}
	if url, ok := data["url"].(string); ok {
		p.Url = url
	}

	// Handle scenario-based navigation
	if scenario, ok := data["scenario"].(string); ok && scenario != "" {
		p.Scenario = &scenario
		destType := basactions.NavigateDestinationType_NAVIGATE_DESTINATION_TYPE_SCENARIO
		p.DestinationType = &destType
	}
	// Support both "path" (CLI) and "scenarioPath" (proto camelCase)
	if path, ok := data["path"].(string); ok {
		p.ScenarioPath = &path
	} else if path, ok := data["scenarioPath"].(string); ok {
		p.ScenarioPath = &path
	}

	if wfs, ok := data["waitForSelector"].(string); ok {
		p.WaitForSelector = &wfs
	}
	if tm, ok := contractvalues.ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	if wu, ok := data["waitUntil"].(string); ok {
		ev := StringToNavigateWaitEvent(wu)
		p.WaitUntil = &ev
	}
	return p
}

// BuildClickParams converts a data map to ClickParams proto.
func BuildClickParams(data map[string]any) *basactions.ClickParams {
	p := &basactions.ClickParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if button, ok := data["button"].(string); ok {
		btn := enums.StringToMouseButton(button)
		p.Button = &btn
	}
	if cc, ok := contractvalues.ToInt32(data["clickCount"]); ok {
		p.ClickCount = &cc
	}
	if dm, ok := contractvalues.ToInt32(data["delayMs"]); ok {
		p.DelayMs = &dm
	}
	for _, modifier := range contractvalues.ToStringSlice(data["modifiers"]) {
		p.Modifiers = append(p.Modifiers, enums.StringToKeyboardModifier(modifier))
	}
	if force, ok := data["force"].(bool); ok {
		p.Force = &force
	}
	return p
}

// BuildInputParams converts a data map to InputParams proto.
// Supports "text" as alias for "value" (both are actively used).
func BuildInputParams(data map[string]any) *basactions.InputParams {
	p := &basactions.InputParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if value, ok := data["value"].(string); ok {
		p.Value = value
	}
	if text, ok := data["text"].(string); ok && p.Value == "" {
		p.Value = text // "text" alias (used by docs/UI)
	}
	if sensitive, ok := data["isSensitive"].(bool); ok {
		p.IsSensitive = &sensitive
	}
	if submit, ok := data["submit"].(bool); ok {
		p.Submit = &submit
	}
	if clear, ok := data["clearFirst"].(bool); ok {
		p.ClearFirst = &clear
	}
	if dm, ok := contractvalues.ToInt32(data["delayMs"]); ok {
		p.DelayMs = &dm
	}
	return p
}

// BuildWaitParams converts a data map to WaitParams proto.
// Supports "duration" as alias for "durationMs", and snake_case spellings
// ("duration_ms", "timeout_ms") from UseProtoNames-marshaled definitions.
func BuildWaitParams(data map[string]any) *basactions.WaitParams {
	p := &basactions.WaitParams{}
	if dm, ok := firstInt32(data, "durationMs", "duration_ms", "duration"); ok {
		p.WaitFor = &basactions.WaitParams_DurationMs{DurationMs: dm}
	} else if selector, ok := data["selector"].(string); ok && selector != "" {
		p.WaitFor = &basactions.WaitParams_Selector{Selector: selector}
	}
	if state, ok := data["state"].(string); ok {
		ws := StringToWaitState(state)
		p.State = &ws
	}
	if tm, ok := firstInt32(data, "timeoutMs", "timeout_ms"); ok {
		p.TimeoutMs = &tm
	}
	return p
}

// BuildAssertParams converts a data map to AssertParams proto.
// Supports both camelCase (CLI/UI) and snake_case (proto/compiler) field names.
// Field name aliases:
// - mode: "mode", "assertMode", "assert_mode"
// - expected: "expected", "expectedText", "expected_text", "expectedValue", "expected_value"
// - attributeName: "attributeName", "attribute_name"
// - caseSensitive: "caseSensitive", "case_sensitive"
// - failureMessage: "failureMessage", "failure_message"
func BuildAssertParams(data map[string]any) *basactions.AssertParams {
	p := &basactions.AssertParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	setAssertMode(p, data)
	setAssertExpected(p, data)
	if negated, ok := data["negated"].(bool); ok {
		p.Negated = &negated
	}
	if value, ok := firstBool(data, "caseSensitive", "case_sensitive"); ok {
		p.CaseSensitive = &value
	}
	if value, ok := firstTypedString(data, "attributeName", "attribute_name"); ok {
		p.AttributeName = &value
	}
	if value, ok := firstTypedString(data, "failureMessage", "failure_message"); ok {
		p.FailureMessage = &value
	}
	return p
}

func setAssertMode(p *basactions.AssertParams, data map[string]any) {
	if mode, ok := firstTypedString(data, "mode", "assertMode", "assert_mode"); ok {
		p.Mode = enums.StringToAssertionMode(mode)
	}
}

func setAssertExpected(p *basactions.AssertParams, data map[string]any) {
	for _, key := range []string{"expected", "expectedText", "expected_text", "expectedValue", "expected_value"} {
		if value := data[key]; value != nil {
			p.Expected = contractvalues.AnyToJsonValue(value)
			return
		}
	}
}

// BuildScrollParams converts a data map to ScrollParams proto.
func BuildScrollParams(data map[string]any) *basactions.ScrollParams {
	p := &basactions.ScrollParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = &selector
	}
	if x, ok := contractvalues.ToInt32(data["x"]); ok {
		p.X = &x
	}
	if y, ok := contractvalues.ToInt32(data["y"]); ok {
		p.Y = &y
	}
	if dx, ok := contractvalues.ToInt32(data["deltaX"]); ok {
		p.DeltaX = &dx
	}
	if dy, ok := contractvalues.ToInt32(data["deltaY"]); ok {
		p.DeltaY = &dy
	}
	if behavior, ok := data["behavior"].(string); ok {
		bh := StringToScrollBehavior(behavior)
		p.Behavior = &bh
	}
	return p
}

// BuildDragDropParams converts the canonical drag parameter vocabulary.
func BuildDragDropParams(data map[string]any) *basactions.DragDropParams {
	p := &basactions.DragDropParams{}
	p.SourceSelector, _ = data["sourceSelector"].(string)
	if target, ok := data["targetSelector"].(string); ok {
		p.TargetSelector = &target
	}
	for key, target := range map[string]**int32{"offsetX": &p.OffsetX, "offsetY": &p.OffsetY, "targetOffsetX": &p.TargetOffsetX, "targetOffsetY": &p.TargetOffsetY, "steps": &p.Steps, "delayMs": &p.DelayMs, "timeoutMs": &p.TimeoutMs} {
		if value, ok := contractvalues.ToInt32(data[key]); ok {
			*target = &value
		}
	}
	return p
}

// BuildSelectParams converts a data map to SelectParams proto.
func BuildSelectParams(data map[string]any) *basactions.SelectParams {
	p := &basactions.SelectParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if value, ok := data["value"].(string); ok {
		p.SelectBy = &basactions.SelectParams_Value{Value: value}
	} else if label, ok := data["label"].(string); ok {
		p.SelectBy = &basactions.SelectParams_Label{Label: label}
	} else if idx, ok := contractvalues.ToInt32(data["index"]); ok {
		p.SelectBy = &basactions.SelectParams_Index{Index: idx}
	}
	if tm, ok := contractvalues.ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

// BuildEvaluateParams converts a data map to EvaluateParams proto.
func BuildEvaluateParams(data map[string]any) *basactions.EvaluateParams {
	p := &basactions.EvaluateParams{}
	if expr, ok := data["expression"].(string); ok {
		p.Expression = expr
	}
	if store, ok := data["storeResult"].(string); ok {
		p.StoreResult = &store
	}
	return p
}

// BuildKeyboardParams converts a data map to KeyboardParams proto.
func BuildKeyboardParams(data map[string]any) *basactions.KeyboardParams {
	p := &basactions.KeyboardParams{}
	if key, ok := data["key"].(string); ok {
		p.Key = &key
	}
	if keys, ok := data["keys"].([]any); ok {
		for _, k := range keys {
			if s, ok := k.(string); ok {
				p.Keys = append(p.Keys, s)
			}
		}
	}
	for _, modifier := range contractvalues.ToStringSlice(data["modifiers"]) {
		p.Modifiers = append(p.Modifiers, enums.StringToKeyboardModifier(modifier))
	}
	if action, ok := data["action"].(string); ok {
		act := StringToKeyAction(action)
		p.Action = &act
	}
	return p
}

// BuildHoverParams converts a data map to HoverParams proto.
func BuildHoverParams(data map[string]any) *basactions.HoverParams {
	p := &basactions.HoverParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if tm, ok := contractvalues.ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

// BuildScreenshotParams converts a data map to ScreenshotParams proto.
func BuildScreenshotParams(data map[string]any) *basactions.ScreenshotParams {
	p := &basactions.ScreenshotParams{}
	if fullPage, ok := data["fullPage"].(bool); ok {
		p.FullPage = &fullPage
	}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = &selector
	}
	if quality, ok := contractvalues.ToInt32(data["quality"]); ok {
		p.Quality = &quality
	}
	return p
}

// BuildFocusParams converts a data map to FocusParams proto.
func BuildFocusParams(data map[string]any) *basactions.FocusParams {
	p := &basactions.FocusParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if scroll, ok := data["scroll"].(bool); ok {
		p.Scroll = &scroll
	}
	if tm, ok := contractvalues.ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

// BuildBlurParams converts a data map to BlurParams proto.
func BuildBlurParams(data map[string]any) *basactions.BlurParams {
	p := &basactions.BlurParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = &selector
	}
	if tm, ok := contractvalues.ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

// BuildSubflowParams converts a data map to SubflowParams proto.
// Supports workflowId/workflow_id, workflowPath/workflow_path, workflowVersion/workflow_version,
// and parameters/args for argument passing.
func BuildSubflowParams(data map[string]any) *basactions.SubflowParams {
	p := &basactions.SubflowParams{}
	setSubflowTarget(p, data)
	if version, ok := firstConvertedInt32(data, "workflowVersion", "workflow_version"); ok {
		p.WorkflowVersion = &version
	}
	if args, ok := firstStringMap(data, "parameters", "args"); ok {
		p.Args = buildSubflowArgs(args)
	}
	return p
}

func setSubflowTarget(p *basactions.SubflowParams, data map[string]any) {
	if id, ok := firstNonEmptyString(data, "workflowId", "workflow_id"); ok {
		p.Target = &basactions.SubflowParams_WorkflowId{WorkflowId: id}
		return
	}
	if path, ok := firstNonEmptyString(data, "workflowPath", "workflow_path"); ok {
		p.Target = &basactions.SubflowParams_WorkflowPath{WorkflowPath: path}
	}
}

func firstNonEmptyString(data map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := data[key].(string); ok && value != "" {
			return value, true
		}
	}
	return "", false
}

func firstStringMap(data map[string]any, keys ...string) (map[string]any, bool) {
	for _, key := range keys {
		if value, ok := data[key].(map[string]any); ok {
			return value, true
		}
	}
	return nil, false
}

func buildSubflowArgs(args map[string]any) map[string]*commonv1.JsonValue {
	if len(args) == 0 {
		return nil
	}
	normalized := make(map[string]*commonv1.JsonValue, len(args))
	for key, value := range args {
		switch typed := value.(type) {
		case *commonv1.JsonValue:
			normalized[key] = typed
		default:
			normalized[key] = contractvalues.AnyToJsonValue(typed)
		}
	}
	return normalized
}

// BuildActionMetadata extracts action metadata from a data map.
// Returns nil if no metadata fields are present.
func BuildActionMetadata(data map[string]any) *basactions.ActionMetadata {
	meta := &basactions.ActionMetadata{}
	hasData := false

	if label, ok := data["label"].(string); ok {
		meta.Label = &label
		hasData = true
	}

	if confidence, ok := contractvalues.ToFloat64(data["confidence"]); ok {
		meta.Confidence = &confidence
		hasData = true
	}

	candidates, hasCandidates := selectorCandidatesFrom(data)
	meta.SelectorCandidates = candidates
	hasData = hasData || hasCandidates

	if !hasData {
		return nil
	}
	return meta
}

func selectorCandidatesFrom(data map[string]any) ([]*basdomain.SelectorCandidate, bool) {
	raw, ok := data["selectorCandidates"].([]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	candidates := make([]*basdomain.SelectorCandidate, 0, len(raw))
	for _, value := range raw {
		fields, ok := value.(map[string]any)
		if !ok {
			continue
		}
		candidate := &basdomain.SelectorCandidate{}
		if value, ok := fields["type"].(string); ok {
			candidate.Type = enums.StringToSelectorType(value)
		}
		if value, ok := fields["value"].(string); ok {
			candidate.Value = value
		}
		if value, ok := contractvalues.ToFloat64(fields["confidence"]); ok {
			candidate.Confidence = value
		}
		if value, ok := contractvalues.ToInt32(fields["specificity"]); ok {
			candidate.Specificity = value
		}
		candidates = append(candidates, candidate)
	}
	return candidates, true
}

// BuildExtractParams converts a data map to ExtractParams proto.
// CLI field mappings:
// - selector (positional) -> Selector
// - attribute/attributeName -> AttributeName + ExtractType=ATTRIBUTE
// - outputKey/storeAs -> StoreAs
// - timeoutMs -> TimeoutMs
func BuildExtractParams(data map[string]any) *basactions.ExtractParams {
	p := &basactions.ExtractParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	setExtractAttribute(p, data)
	setExtractStoreAs(p, data)
	setExtractProperty(p, data)
	setExtractTimeout(p, data)
	return p
}

func setExtractAttribute(p *basactions.ExtractParams, data map[string]any) {
	if name, ok := firstTypedString(data, "attribute", "attributeName", "attribute_name"); ok {
		p.AttributeName = &name
		extractType := basactions.ExtractType_EXTRACT_TYPE_ATTRIBUTE
		p.ExtractType = &extractType
	}
}

func setExtractStoreAs(p *basactions.ExtractParams, data map[string]any) {
	if name, ok := firstTypedString(data, "outputKey", "storeAs", "store_as"); ok {
		p.StoreAs = &name
	}
}

func setExtractProperty(p *basactions.ExtractParams, data map[string]any) {
	if name, ok := firstTypedString(data, "propertyName", "property_name"); ok {
		p.PropertyName = &name
		extractType := basactions.ExtractType_EXTRACT_TYPE_PROPERTY
		p.ExtractType = &extractType
	}
}

func setExtractTimeout(p *basactions.ExtractParams, data map[string]any) {
	if value, ok := contractvalues.ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &value
		return
	}
	if value, ok := contractvalues.ToInt32(data["timeout_ms"]); ok {
		p.TimeoutMs = &value
	}
}

// BuildShortcutParams converts a data map to ShortcutParams proto.
// CLI field mappings:
// - keys (positional) or shortcut -> Shortcut
// - selector (optional) -> Selector
// Example shortcuts: "Control+a", "Meta+Shift+s"
func BuildShortcutParams(data map[string]any) *basactions.ShortcutParams {
	p := &basactions.ShortcutParams{}
	// CLI uses "keys" as positional, proto uses "shortcut"
	if keys, ok := data["keys"].(string); ok {
		p.Shortcut = keys
	} else if shortcut, ok := data["shortcut"].(string); ok {
		p.Shortcut = shortcut
	}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = &selector
	}
	return p
}

// BuildGestureParams converts a data map to GestureParams proto.
func BuildGestureParams(data map[string]any) *basactions.GestureParams {
	p := &basactions.GestureParams{}
	setGestureIdentity(p, data)
	setGestureMotion(p, data)
	setGestureTiming(p, data)
	return p
}

func setGestureIdentity(p *basactions.GestureParams, data map[string]any) {
	if gestureType, ok := firstString(data, "gesture_type", "gestureType", "type"); ok {
		p.GestureType = StringToGestureType(gestureType)
	}
	if selector, ok := firstString(data, "selector"); ok {
		p.Selector = &selector
	}
	if direction, ok := firstString(data, "direction"); ok {
		parsed := StringToSwipeDirection(direction)
		p.Direction = &parsed
	}
}

func setGestureMotion(p *basactions.GestureParams, data map[string]any) {
	if distance, ok := firstInt32(data, "distance"); ok {
		p.Distance = &distance
	}
	if scale, ok := firstFloat64(data, "scale"); ok {
		p.Scale = &scale
	}
	if duration, ok := firstInt32(data, "duration_ms", "durationMs"); ok {
		p.DurationMs = &duration
	}
	if steps, ok := firstInt32(data, "steps"); ok {
		p.Steps = &steps
	}
}

func setGestureTiming(p *basactions.GestureParams, data map[string]any) {
	if delay, ok := firstInt32(data, "step_delay_ms", "stepDelayMs"); ok {
		p.StepDelayMs = &delay
	}
	if label, ok := firstString(data, "trace_label", "traceLabel"); ok {
		p.TraceLabel = &label
	}
	if idle, ok := firstInt32(data, "idle_after_ms", "idleAfterMs"); ok {
		p.IdleAfterMs = &idle
	}
	if delta, ok := firstInt32(data, "wheel_delta_y", "wheelDeltaY"); ok {
		p.WheelDeltaY = &delta
	}
	if ctrlKey, ok := firstBool(data, "ctrl_key", "ctrlKey"); ok {
		p.CtrlKey = &ctrlKey
	}
}

// StringToGestureType converts proto and shorthand gesture labels.
func StringToGestureType(s string) basactions.GestureType {
	return basactions.GestureType(generatedEnumValue(basactions.GestureType_value, s))
}

// StringToSwipeDirection converts proto and shorthand swipe directions.
func StringToSwipeDirection(s string) basactions.SwipeDirection {
	return basactions.SwipeDirection(generatedEnumValue(basactions.SwipeDirection_value, s))
}

func firstString(data map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if str := strings.TrimSpace(contractvalues.ToString(value)); str != "" {
				return str, true
			}
		}
	}
	return "", false
}

func firstTypedString(data map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := data[key].(string); ok {
			return value, true
		}
	}
	return "", false
}

func firstConvertedInt32(data map[string]any, keys ...string) (int32, bool) {
	for _, key := range keys {
		if value, ok := contractvalues.ToInt32(data[key]); ok {
			return value, true
		}
	}
	return 0, false
}

func firstInt32(data map[string]any, keys ...string) (int32, bool) {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			return int32(contractvalues.ToInt(value)), true
		}
	}
	return 0, false
}

func firstFloat64(data map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			return contractvalues.ToFloat(value), true
		}
	}
	return 0, false
}

func firstBool(data map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			return contractvalues.ToBool(value), true
		}
	}
	return false, false
}
