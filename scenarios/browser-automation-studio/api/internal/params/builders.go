// Package params provides parameter builder functions for converting
// map-based data to typed proto parameter messages.
package params

import (
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
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

// StringToMouseButton converts a string to MouseButton enum.
func StringToMouseButton(s string) basactions.MouseButton {
	switch s {
	case "left":
		return basactions.MouseButton_MOUSE_BUTTON_LEFT
	case "right":
		return basactions.MouseButton_MOUSE_BUTTON_RIGHT
	case "middle":
		return basactions.MouseButton_MOUSE_BUTTON_MIDDLE
	default:
		return basactions.MouseButton_MOUSE_BUTTON_UNSPECIFIED
	}
}

// StringToKeyboardModifier converts a string to KeyboardModifier enum.
func StringToKeyboardModifier(s string) basactions.KeyboardModifier {
	switch s {
	case "Alt":
		return basactions.KeyboardModifier_KEYBOARD_MODIFIER_ALT
	case "Control", "Ctrl":
		return basactions.KeyboardModifier_KEYBOARD_MODIFIER_CTRL
	case "Meta":
		return basactions.KeyboardModifier_KEYBOARD_MODIFIER_META
	case "Shift":
		return basactions.KeyboardModifier_KEYBOARD_MODIFIER_SHIFT
	default:
		return basactions.KeyboardModifier_KEYBOARD_MODIFIER_UNSPECIFIED
	}
}

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

// StringToAssertionMode converts a string to AssertionMode enum.
func StringToAssertionMode(s string) basbase.AssertionMode {
	switch s {
	case "exists":
		return basbase.AssertionMode_ASSERTION_MODE_EXISTS
	case "not_exists":
		return basbase.AssertionMode_ASSERTION_MODE_NOT_EXISTS
	case "visible":
		return basbase.AssertionMode_ASSERTION_MODE_VISIBLE
	case "hidden":
		return basbase.AssertionMode_ASSERTION_MODE_HIDDEN
	case "text_equals":
		return basbase.AssertionMode_ASSERTION_MODE_TEXT_EQUALS
	case "text_contains":
		return basbase.AssertionMode_ASSERTION_MODE_TEXT_CONTAINS
	case "attribute_equals":
		return basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_EQUALS
	case "attribute_contains":
		return basbase.AssertionMode_ASSERTION_MODE_ATTRIBUTE_CONTAINS
	default:
		return basbase.AssertionMode_ASSERTION_MODE_UNSPECIFIED
	}
}

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
	if tm, ok := typeconv.ToInt32(data["timeoutMs"]); ok {
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
		btn := StringToMouseButton(button)
		p.Button = &btn
	}
	if cc, ok := typeconv.ToInt32(data["clickCount"]); ok {
		p.ClickCount = &cc
	}
	if dm, ok := typeconv.ToInt32(data["delayMs"]); ok {
		p.DelayMs = &dm
	}
	if mods, ok := data["modifiers"].([]any); ok {
		for _, m := range mods {
			if s, ok := m.(string); ok {
				p.Modifiers = append(p.Modifiers, StringToKeyboardModifier(s))
			}
		}
	}
	if force, ok := data["force"].(bool); ok {
		p.Force = &force
	}
	return p
}

// BuildInputParams converts a data map to InputParams proto.
// Supports legacy field "text" as alias for "value".
func BuildInputParams(data map[string]any) *basactions.InputParams {
	p := &basactions.InputParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if value, ok := data["value"].(string); ok {
		p.Value = value
	}
	if text, ok := data["text"].(string); ok && p.Value == "" {
		p.Value = text // Legacy field name
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
	if dm, ok := typeconv.ToInt32(data["delayMs"]); ok {
		p.DelayMs = &dm
	}
	return p
}

// BuildWaitParams converts a data map to WaitParams proto.
// Supports legacy field "duration" as alias for "durationMs".
func BuildWaitParams(data map[string]any) *basactions.WaitParams {
	p := &basactions.WaitParams{}
	if dm, ok := typeconv.ToInt32(data["durationMs"]); ok {
		p.WaitFor = &basactions.WaitParams_DurationMs{DurationMs: dm}
	} else if dur, ok := typeconv.ToInt32(data["duration"]); ok {
		p.WaitFor = &basactions.WaitParams_DurationMs{DurationMs: dur} // Legacy field name
	} else if selector, ok := data["selector"].(string); ok && selector != "" {
		p.WaitFor = &basactions.WaitParams_Selector{Selector: selector}
	}
	if state, ok := data["state"].(string); ok {
		ws := StringToWaitState(state)
		p.State = &ws
	}
	if tm, ok := typeconv.ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
}

// BuildAssertParams converts a data map to AssertParams proto.
// Supports legacy field "assertMode" as alias for "mode".
func BuildAssertParams(data map[string]any) *basactions.AssertParams {
	p := &basactions.AssertParams{}
	if selector, ok := data["selector"].(string); ok {
		p.Selector = selector
	}
	if mode, ok := data["mode"].(string); ok {
		p.Mode = StringToAssertionMode(mode)
	} else if mode, ok := data["assertMode"].(string); ok {
		p.Mode = StringToAssertionMode(mode) // Legacy field name
	}
	if exp := data["expected"]; exp != nil {
		p.Expected = typeconv.AnyToJsonValue(exp)
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
	if x, ok := typeconv.ToInt32(data["x"]); ok {
		p.X = &x
	}
	if y, ok := typeconv.ToInt32(data["y"]); ok {
		p.Y = &y
	}
	if dx, ok := typeconv.ToInt32(data["deltaX"]); ok {
		p.DeltaX = &dx
	}
	if dy, ok := typeconv.ToInt32(data["deltaY"]); ok {
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
	} else if idx, ok := typeconv.ToInt32(data["index"]); ok {
		p.SelectBy = &basactions.SelectParams_Index{Index: idx}
	}
	if tm, ok := typeconv.ToInt32(data["timeoutMs"]); ok {
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
				p.Modifiers = append(p.Modifiers, StringToKeyboardModifier(s))
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
	if tm, ok := typeconv.ToInt32(data["timeoutMs"]); ok {
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
	if quality, ok := typeconv.ToInt32(data["quality"]); ok {
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
	if tm, ok := typeconv.ToInt32(data["timeoutMs"]); ok {
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
	if tm, ok := typeconv.ToInt32(data["timeoutMs"]); ok {
		p.TimeoutMs = &tm
	}
	return p
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

	if confidence, ok := typeconv.ToFloat64(data["confidence"]); ok {
		meta.Confidence = &confidence
		hasData = true
	}

	// Extract selector candidates if present
	if candidates, ok := data["selectorCandidates"].([]any); ok && len(candidates) > 0 {
		for _, c := range candidates {
			if cm, ok := c.(map[string]any); ok {
				candidate := &basdomain.SelectorCandidate{}
				if t, ok := cm["type"].(string); ok {
					candidate.Type = typeconv.StringToSelectorType(t)
				}
				if v, ok := cm["value"].(string); ok {
					candidate.Value = v
				}
				if conf, ok := typeconv.ToFloat64(cm["confidence"]); ok {
					candidate.Confidence = conf
				}
				if spec, ok := typeconv.ToInt32(cm["specificity"]); ok {
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
