// Parameter builder functions for converting map-based data to typed proto parameter messages.
// Consolidates parameter building logic that was previously in internal/params.
//
// Field Aliases (both names are actively used and supported):
// - InputParams: "text" and "value" are equivalent (docs/UI use "text", proto uses "value")
// - WaitParams: "duration" and "durationMs" are equivalent
// - AssertParams: "assertMode" and "mode" are equivalent
package typeconv

import (
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
	switch s {
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
// Supports "duration" as alias for "durationMs".
func BuildWaitParams(data map[string]any) *basactions.WaitParams {
	p := &basactions.WaitParams{}
	if dm, ok := ToInt32(data["durationMs"]); ok {
		p.WaitFor = &basactions.WaitParams_DurationMs{DurationMs: dm}
	} else if dur, ok := ToInt32(data["duration"]); ok {
		p.WaitFor = &basactions.WaitParams_DurationMs{DurationMs: dur} // "duration" alias
	} else if selector, ok := data["selector"].(string); ok && selector != "" {
		p.WaitFor = &basactions.WaitParams_Selector{Selector: selector}
	}
	if state, ok := data["state"].(string); ok {
		ws := StringToWaitState(state)
		p.State = &ws
	}
	if tm, ok := ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

// BuildAssertParams converts a data map to AssertParams proto.
// Supports "assertMode" as alias for "mode".
func BuildAssertParams(data map[string]any) *basactions.AssertParams {
	p := &basactions.AssertParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if mode, ok := data["mode"].(string); ok {
		p.Mode = enums.StringToAssertionMode(mode)
	} else if mode, ok := data["assertMode"].(string); ok {
		p.Mode = enums.StringToAssertionMode(mode) // "assertMode" alias
	}
	if exp := data["expected"]; exp != nil {
		p.Expected = AnyToJsonValue(exp)
	}
	if negated, ok := data["negated"].(bool); ok {
		p.Negated = &negated
	}
	if caseSensitive, ok := data["caseSensitive"].(bool); ok {
		p.CaseSensitive = &caseSensitive
	}
	if attrName, ok := data["attributeName"].(string); ok {
		p.AttributeName = &attrName
	}
	if failureMsg, ok := data["failureMessage"].(string); ok {
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
