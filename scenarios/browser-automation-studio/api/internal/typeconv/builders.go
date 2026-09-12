// Parameter builder functions for converting map-based data to typed proto parameter messages.
// Consolidates parameter building logic that was previously in internal/params.
//
// Field Aliases (both names are actively used and supported):
// - InputParams: "text" and "value" are equivalent (docs/UI use "text", proto uses "value")
// - WaitParams: "duration" and "durationMs" are equivalent
// - AssertParams: "assertMode" and "mode" are equivalent
package typeconv

import (
	"strings"

	"github.com/vrooli/browser-automation-studio/internal/enums"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// StringToNavigateWaitEvent converts a string to NavigateWaitEvent enum.
func StringToNavigateWaitEvent(s string) basactions.NavigateWaitEvent {
	switch s {
	case "load":
		return basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_LOAD
	case "domcontentloaded":
		return basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED
	case "networkidle":
		return basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_NETWORKIDLE
	default:
		return basactions.NavigateWaitEvent_NAVIGATE_WAIT_EVENT_UNSPECIFIED
	}
}

// Note: StringToMouseButton, StringToKeyboardModifier, StringToAssertionMode, and
// StringToSelectorType are defined in internal/enums package.

// StringToWaitState converts a string to WaitState enum.
func StringToWaitState(s string) basactions.WaitState {
	normalized := strings.TrimSpace(strings.ToLower(s))
	normalized = strings.TrimPrefix(normalized, "wait_state_")
	switch normalized {
	case "attached":
		return basactions.WaitState_WAIT_STATE_ATTACHED
	case "detached":
		return basactions.WaitState_WAIT_STATE_DETACHED
	case "visible":
		return basactions.WaitState_WAIT_STATE_VISIBLE
	case "hidden":
		return basactions.WaitState_WAIT_STATE_HIDDEN
	default:
		return basactions.WaitState_WAIT_STATE_UNSPECIFIED
	}
}

// Note: StringToAssertionMode is defined in primitives.go with input normalization.

// StringToScrollBehavior converts a string to ScrollBehavior enum.
func StringToScrollBehavior(s string) basactions.ScrollBehavior {
	switch s {
	case "auto":
		return basactions.ScrollBehavior_SCROLL_BEHAVIOR_AUTO
	case "smooth":
		return basactions.ScrollBehavior_SCROLL_BEHAVIOR_SMOOTH
	default:
		return basactions.ScrollBehavior_SCROLL_BEHAVIOR_UNSPECIFIED
	}
}

// StringToKeyAction converts a string to KeyAction enum.
func StringToKeyAction(s string) basactions.KeyAction {
	switch s {
	case "press":
		return basactions.KeyAction_KEY_ACTION_PRESS
	case "down":
		return basactions.KeyAction_KEY_ACTION_DOWN
	case "up":
		return basactions.KeyAction_KEY_ACTION_UP
	default:
		return basactions.KeyAction_KEY_ACTION_UNSPECIFIED
	}
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
	if tm, ok := ToInt32(data["timeoutMs"]); ok {
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
	if cc, ok := ToInt32(data["clickCount"]); ok {
		p.ClickCount = &cc
	}
	if dm, ok := ToInt32(data["delayMs"]); ok {
		p.DelayMs = &dm
	}
	if mods, ok := data["modifiers"].([]any); ok {
		for _, m := range mods {
			if s, ok := m.(string); ok {
				p.Modifiers = append(p.Modifiers, enums.StringToKeyboardModifier(s))
			}
		}
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
	if dm, ok := ToInt32(data["delayMs"]); ok {
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
	// Mode: check camelCase first, then snake_case
	if mode, ok := data["mode"].(string); ok {
		p.Mode = enums.StringToAssertionMode(mode)
	} else if mode, ok := data["assertMode"].(string); ok {
		p.Mode = enums.StringToAssertionMode(mode)
	} else if mode, ok := data["assert_mode"].(string); ok {
		p.Mode = enums.StringToAssertionMode(mode)
	}
	// Expected: check proto field first, then CLI aliases (camelCase and snake_case)
	if exp := data["expected"]; exp != nil {
		p.Expected = AnyToJsonValue(exp)
	} else if exp := data["expectedText"]; exp != nil {
		p.Expected = AnyToJsonValue(exp)
	} else if exp := data["expected_text"]; exp != nil {
		p.Expected = AnyToJsonValue(exp)
	} else if exp := data["expectedValue"]; exp != nil {
		p.Expected = AnyToJsonValue(exp)
	} else if exp := data["expected_value"]; exp != nil {
		p.Expected = AnyToJsonValue(exp)
	}
	if negated, ok := data["negated"].(bool); ok {
		p.Negated = &negated
	}
	// CaseSensitive: camelCase and snake_case
	if caseSensitive, ok := data["caseSensitive"].(bool); ok {
		p.CaseSensitive = &caseSensitive
	} else if caseSensitive, ok := data["case_sensitive"].(bool); ok {
		p.CaseSensitive = &caseSensitive
	}
	// AttributeName: camelCase and snake_case (critical for attribute assertions)
	if attrName, ok := data["attributeName"].(string); ok {
		p.AttributeName = &attrName
	} else if attrName, ok := data["attribute_name"].(string); ok {
		p.AttributeName = &attrName
	}
	// FailureMessage: camelCase and snake_case
	if failureMsg, ok := data["failureMessage"].(string); ok {
		p.FailureMessage = &failureMsg
	} else if failureMsg, ok := data["failure_message"].(string); ok {
		p.FailureMessage = &failureMsg
	}
	return p
}

// BuildScrollParams converts a data map to ScrollParams proto.
func BuildScrollParams(data map[string]any) *basactions.ScrollParams {
	p := &basactions.ScrollParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = &selector
	}
	if x, ok := ToInt32(data["x"]); ok {
		p.X = &x
	}
	if y, ok := ToInt32(data["y"]); ok {
		p.Y = &y
	}
	if dx, ok := ToInt32(data["deltaX"]); ok {
		p.DeltaX = &dx
	}
	if dy, ok := ToInt32(data["deltaY"]); ok {
		p.DeltaY = &dy
	}
	if behavior, ok := data["behavior"].(string); ok {
		bh := StringToScrollBehavior(behavior)
		p.Behavior = &bh
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
	} else if idx, ok := ToInt32(data["index"]); ok {
		p.SelectBy = &basactions.SelectParams_Index{Index: idx}
	}
	if tm, ok := ToInt32(data["timeoutMs"]); ok {
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
	if mods, ok := data["modifiers"].([]any); ok {
		for _, m := range mods {
			if s, ok := m.(string); ok {
				p.Modifiers = append(p.Modifiers, enums.StringToKeyboardModifier(s))
			}
		}
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
	if tm, ok := ToInt32(data["timeoutMs"]); ok {
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
	if quality, ok := ToInt32(data["quality"]); ok {
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
	if tm, ok := ToInt32(data["timeoutMs"]); ok {
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
	if tm, ok := ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

// BuildSubflowParams converts a data map to SubflowParams proto.
// Supports workflowId/workflow_id, workflowPath/workflow_path, workflowVersion/workflow_version,
// and parameters/args for argument passing.
func BuildSubflowParams(data map[string]any) *basactions.SubflowParams {
	p := &basactions.SubflowParams{}
	if id, ok := data["workflowId"].(string); ok && id != "" {
		p.Target = &basactions.SubflowParams_WorkflowId{WorkflowId: id}
	} else if id, ok := data["workflow_id"].(string); ok && id != "" {
		p.Target = &basactions.SubflowParams_WorkflowId{WorkflowId: id}
	}
	if p.Target == nil {
		if path, ok := data["workflowPath"].(string); ok && path != "" {
			p.Target = &basactions.SubflowParams_WorkflowPath{WorkflowPath: path}
		} else if path, ok := data["workflow_path"].(string); ok && path != "" {
			p.Target = &basactions.SubflowParams_WorkflowPath{WorkflowPath: path}
		}
	}
	if version, ok := ToInt32(data["workflowVersion"]); ok {
		p.WorkflowVersion = &version
	} else if version, ok := ToInt32(data["workflow_version"]); ok {
		p.WorkflowVersion = &version
	}

	if args, ok := data["parameters"].(map[string]any); ok {
		p.Args = buildSubflowArgs(args)
	} else if args, ok := data["args"].(map[string]any); ok {
		p.Args = buildSubflowArgs(args)
	}
	return p
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
			normalized[key] = AnyToJsonValue(typed)
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

	if confidence, ok := ToFloat64(data["confidence"]); ok {
		meta.Confidence = &confidence
		hasData = true
	}

	// Extract selector candidates if present
	if candidates, ok := data["selectorCandidates"].([]any); ok && len(candidates) > 0 {
		for _, c := range candidates {
			if cm, ok := c.(map[string]any); ok {
				candidate := &basdomain.SelectorCandidate{}
				if t, ok := cm["type"].(string); ok {
					candidate.Type = enums.StringToSelectorType(t)
				}
				if v, ok := cm["value"].(string); ok {
					candidate.Value = v
				}
				if conf, ok := ToFloat64(cm["confidence"]); ok {
					candidate.Confidence = conf
				}
				if spec, ok := ToInt32(cm["specificity"]); ok {
					candidate.Specificity = spec
				}
				meta.SelectorCandidates = append(meta.SelectorCandidates, candidate)
			}
		}
		hasData = true
	}

	if !hasData {
		return nil
	}
	return meta
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
	// If attribute is specified, set extract_type to ATTRIBUTE
	if attr, ok := data["attribute"].(string); ok {
		p.AttributeName = &attr
		extractType := basactions.ExtractType_EXTRACT_TYPE_ATTRIBUTE
		p.ExtractType = &extractType
	} else if attrName, ok := data["attributeName"].(string); ok {
		p.AttributeName = &attrName
		extractType := basactions.ExtractType_EXTRACT_TYPE_ATTRIBUTE
		p.ExtractType = &extractType
	} else if attrName, ok := data["attribute_name"].(string); ok {
		p.AttributeName = &attrName
		extractType := basactions.ExtractType_EXTRACT_TYPE_ATTRIBUTE
		p.ExtractType = &extractType
	}
	// outputKey maps to store_as (proto field)
	if outputKey, ok := data["outputKey"].(string); ok {
		p.StoreAs = &outputKey
	} else if storeAs, ok := data["storeAs"].(string); ok {
		p.StoreAs = &storeAs
	} else if storeAs, ok := data["store_as"].(string); ok {
		p.StoreAs = &storeAs
	}
	// Property name for PROPERTY extraction
	if propName, ok := data["propertyName"].(string); ok {
		p.PropertyName = &propName
		extractType := basactions.ExtractType_EXTRACT_TYPE_PROPERTY
		p.ExtractType = &extractType
	} else if propName, ok := data["property_name"].(string); ok {
		p.PropertyName = &propName
		extractType := basactions.ExtractType_EXTRACT_TYPE_PROPERTY
		p.ExtractType = &extractType
	}
	if tm, ok := ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	} else if tm, ok := ToInt32(data["timeout_ms"]); ok {
		p.TimeoutMs = &tm
	}
	return p
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
	return p
}

// StringToGestureType converts proto and shorthand gesture labels.
func StringToGestureType(s string) basactions.GestureType {
	normalized := strings.TrimSpace(strings.ToLower(s))
	normalized = strings.TrimPrefix(normalized, "gesture_type_")
	switch normalized {
	case "swipe":
		return basactions.GestureType_GESTURE_TYPE_SWIPE
	case "pinch":
		return basactions.GestureType_GESTURE_TYPE_PINCH
	case "zoom":
		return basactions.GestureType_GESTURE_TYPE_ZOOM
	case "long_press", "longpress":
		return basactions.GestureType_GESTURE_TYPE_LONG_PRESS
	case "double_tap", "doubletap":
		return basactions.GestureType_GESTURE_TYPE_DOUBLE_TAP
	default:
		return basactions.GestureType_GESTURE_TYPE_UNSPECIFIED
	}
}

// StringToSwipeDirection converts proto and shorthand swipe directions.
func StringToSwipeDirection(s string) basactions.SwipeDirection {
	normalized := strings.TrimSpace(strings.ToLower(s))
	normalized = strings.TrimPrefix(normalized, "swipe_direction_")
	switch normalized {
	case "up":
		return basactions.SwipeDirection_SWIPE_DIRECTION_UP
	case "down":
		return basactions.SwipeDirection_SWIPE_DIRECTION_DOWN
	case "left":
		return basactions.SwipeDirection_SWIPE_DIRECTION_LEFT
	case "right":
		return basactions.SwipeDirection_SWIPE_DIRECTION_RIGHT
	default:
		return basactions.SwipeDirection_SWIPE_DIRECTION_UNSPECIFIED
	}
}

func firstString(data map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if str := strings.TrimSpace(ToString(value)); str != "" {
				return str, true
			}
		}
	}
	return "", false
}

func firstInt32(data map[string]any, keys ...string) (int32, bool) {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			return int32(ToInt(value)), true
		}
	}
	return 0, false
}

func firstFloat64(data map[string]any, keys ...string) (float64, bool) {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			return ToFloat(value), true
		}
	}
	return 0, false
}

func firstBool(data map[string]any, keys ...string) (bool, bool) {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			return ToBool(value), true
		}
	}
	return false, false
}
