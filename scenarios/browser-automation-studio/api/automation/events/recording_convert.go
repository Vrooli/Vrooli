// Package events provides conversion utilities for recording actions to the
// unified bastimeline.TimelineEntry proto format.
//
// This enables recording telemetry to use the same data structure as execution
// telemetry, supporting the shared Record/Execute UX on the timeline.
//
// See "UNIFIED RECORDING/EXECUTION MODEL" in shared.proto for design rationale.
package events

import (
	"strings"
	"time"

	"github.com/vrooli/browser-automation-studio/internal/params"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RecordedActionToTimelineEntry converts a RecordedAction from the recording
// controller to the unified TimelineEntry proto format.
func RecordedActionToTimelineEntry(action *livecapture.RecordedAction) *bastimeline.TimelineEntry {
	if action == nil {
		return nil
	}

	// Build action definition
	actionDef := buildRecordingActionDefinition(action)

	// Build telemetry
	telemetry := buildRecordingTelemetry(action)

	// Build event context with recording origin
	context := buildRecordingEventContext(action)

	entry := &bastimeline.TimelineEntry{
		Id:          action.ID,
		SequenceNum: int32(action.SequenceNum),
		Action:      actionDef,
		Telemetry:   telemetry,
		Context:     context,
	}

	// Parse and set timestamp
	if action.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, action.Timestamp); err == nil {
			entry.Timestamp = timestamppb.New(ts)
		} else if ts, err := time.Parse(time.RFC3339, action.Timestamp); err == nil {
			entry.Timestamp = timestamppb.New(ts)
		}
	}

	// Set duration if present
	if action.DurationMs > 0 {
		durationMs := int32(action.DurationMs)
		entry.DurationMs = &durationMs
	}

	return entry
}

// buildRecordingActionDefinition creates an ActionDefinition from a RecordedAction.
func buildRecordingActionDefinition(action *livecapture.RecordedAction) *basactions.ActionDefinition {
	actionType := mapRecordingActionType(action.ActionType)

	def := &basactions.ActionDefinition{
		Type: actionType,
	}

	// Get selector from the action
	selector := ""
	if action.Selector != nil && action.Selector.Primary != "" {
		selector = action.Selector.Primary
	}

	// Build params based on action type
	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		url := action.URL
		if targetURL, ok := action.Payload["targetUrl"].(string); ok {
			url = targetURL
		}
		def.Params = &basactions.ActionDefinition_Navigate{
			Navigate: &basactions.NavigateParams{
				Url: url,
			},
		}

	case basactions.ActionType_ACTION_TYPE_CLICK:
		clickParams := &basactions.ClickParams{
			Selector: selector,
		}
		if button, ok := action.Payload["button"].(string); ok {
			btn := params.StringToMouseButton(button)
			clickParams.Button = &btn
		}
		if clickCount, ok := extractInt32FromPayload(action.Payload, "clickCount"); ok {
			clickParams.ClickCount = &clickCount
		}
		if modifiers, ok := action.Payload["modifiers"].([]any); ok {
			for _, m := range modifiers {
				if s, ok := m.(string); ok {
					clickParams.Modifiers = append(clickParams.Modifiers, params.StringToKeyboardModifier(s))
				}
			}
		}
		def.Params = &basactions.ActionDefinition_Click{Click: clickParams}

	case basactions.ActionType_ACTION_TYPE_INPUT:
		inputParams := &basactions.InputParams{
			Selector: selector,
		}
		if text, ok := action.Payload["text"].(string); ok {
			inputParams.Value = text
		}
		if clearFirst, ok := action.Payload["clearFirst"].(bool); ok {
			inputParams.ClearFirst = &clearFirst
		}
		if delay, ok := extractInt32FromPayload(action.Payload, "delay"); ok {
			inputParams.DelayMs = &delay
		}
		def.Params = &basactions.ActionDefinition_Input{Input: inputParams}

	case basactions.ActionType_ACTION_TYPE_SCROLL:
		scrollParams := &basactions.ScrollParams{}
		if selector != "" {
			scrollParams.Selector = &selector
		}
		if x, ok := extractInt32FromPayload(action.Payload, "scrollX"); ok {
			scrollParams.X = &x
		}
		if y, ok := extractInt32FromPayload(action.Payload, "scrollY"); ok {
			scrollParams.Y = &y
		}
		if deltaX, ok := extractInt32FromPayload(action.Payload, "deltaX"); ok {
			scrollParams.DeltaX = &deltaX
		}
		if deltaY, ok := extractInt32FromPayload(action.Payload, "deltaY"); ok {
			scrollParams.DeltaY = &deltaY
		}
		def.Params = &basactions.ActionDefinition_Scroll{Scroll: scrollParams}

	case basactions.ActionType_ACTION_TYPE_HOVER:
		def.Params = &basactions.ActionDefinition_Hover{
			Hover: &basactions.HoverParams{Selector: selector},
		}

	case basactions.ActionType_ACTION_TYPE_FOCUS:
		def.Params = &basactions.ActionDefinition_Focus{
			Focus: &basactions.FocusParams{Selector: selector},
		}

	case basactions.ActionType_ACTION_TYPE_BLUR:
		blurParams := &basactions.BlurParams{}
		if selector != "" {
			blurParams.Selector = &selector
		}
		def.Params = &basactions.ActionDefinition_Blur{Blur: blurParams}

	case basactions.ActionType_ACTION_TYPE_SELECT:
		selectParams := &basactions.SelectParams{Selector: selector}
		if value, ok := action.Payload["value"].(string); ok {
			selectParams.SelectBy = &basactions.SelectParams_Value{Value: value}
		} else if label, ok := action.Payload["selectedText"].(string); ok {
			selectParams.SelectBy = &basactions.SelectParams_Label{Label: label}
		} else if index, ok := extractInt32FromPayload(action.Payload, "selectedIndex"); ok {
			selectParams.SelectBy = &basactions.SelectParams_Index{Index: index}
		}
		def.Params = &basactions.ActionDefinition_SelectOption{SelectOption: selectParams}

	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		keyboardParams := &basactions.KeyboardParams{}
		if key, ok := action.Payload["key"].(string); ok {
			keyboardParams.Key = &key
		}
		if modifiers, ok := action.Payload["modifiers"].([]any); ok {
			for _, m := range modifiers {
				if s, ok := m.(string); ok {
					keyboardParams.Modifiers = append(keyboardParams.Modifiers, params.StringToKeyboardModifier(s))
				}
			}
		}
		def.Params = &basactions.ActionDefinition_Keyboard{Keyboard: keyboardParams}

	case basactions.ActionType_ACTION_TYPE_WAIT:
		waitParams := &basactions.WaitParams{}
		if ms, ok := extractInt32FromPayload(action.Payload, "ms"); ok {
			waitParams.WaitFor = &basactions.WaitParams_DurationMs{DurationMs: ms}
		} else if waitSelector, ok := action.Payload["selector"].(string); ok {
			waitParams.WaitFor = &basactions.WaitParams_Selector{Selector: waitSelector}
		}
		def.Params = &basactions.ActionDefinition_Wait{Wait: waitParams}

	case basactions.ActionType_ACTION_TYPE_ASSERT:
		assertParams := &basactions.AssertParams{
			Selector: selector,
			Mode:     basbase.AssertionMode_ASSERTION_MODE_EXISTS,
		}
		def.Params = &basactions.ActionDefinition_Assert{Assert: assertParams}

	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		screenshotParams := &basactions.ScreenshotParams{}
		if fullPage, ok := action.Payload["fullPage"].(bool); ok {
			screenshotParams.FullPage = &fullPage
		}
		def.Params = &basactions.ActionDefinition_Screenshot{Screenshot: screenshotParams}

	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		evalParams := &basactions.EvaluateParams{}
		if expr, ok := action.Payload["expression"].(string); ok {
			evalParams.Expression = expr
		}
		def.Params = &basactions.ActionDefinition_Evaluate{Evaluate: evalParams}

	default:
		// Default to click for unknown types
		def.Params = &basactions.ActionDefinition_Click{
			Click: &basactions.ClickParams{Selector: selector},
		}
	}

	// Add metadata
	def.Metadata = buildRecordingMetadata(action)

	return def
}

// buildRecordingMetadata creates ActionMetadata from a RecordedAction.
func buildRecordingMetadata(action *livecapture.RecordedAction) *basactions.ActionMetadata {
	meta := &basactions.ActionMetadata{}

	// Generate label from element info
	label := generateRecordingLabel(action)
	meta.Label = &label

	// Add confidence
	if action.Confidence > 0 {
		meta.Confidence = &action.Confidence
	}

	// Add captured timestamp (renamed from recorded_at)
	if action.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, action.Timestamp); err == nil {
			meta.CapturedAt = timestamppb.New(ts)
		} else if ts, err := time.Parse(time.RFC3339, action.Timestamp); err == nil {
			meta.CapturedAt = timestamppb.New(ts)
		}
	}

	// Add bounding box (renamed from recorded_bounding_box)
	if action.BoundingBox != nil {
		meta.CapturedBoundingBox = &basbase.BoundingBox{
			X:      action.BoundingBox.X,
			Y:      action.BoundingBox.Y,
			Width:  action.BoundingBox.Width,
			Height: action.BoundingBox.Height,
		}
	}

	// Add selector candidates
	if action.Selector != nil && len(action.Selector.Candidates) > 0 {
		meta.SelectorCandidates = make([]*basdomain.SelectorCandidate, 0, len(action.Selector.Candidates))
		for _, c := range action.Selector.Candidates {
			meta.SelectorCandidates = append(meta.SelectorCandidates, &basdomain.SelectorCandidate{
				Type:        typeconv.StringToSelectorType(c.Type),
				Value:       c.Value,
				Confidence:  c.Confidence,
				Specificity: int32(c.Specificity),
			})
		}
	}

	// Add element snapshot (uses ElementMeta from record_mode.proto)
	if action.ElementMeta != nil {
		meta.ElementSnapshot = &basdomain.ElementMeta{
			TagName:   action.ElementMeta.TagName,
			Id:        action.ElementMeta.ID,
			ClassName: action.ElementMeta.ClassName,
			InnerText: action.ElementMeta.InnerText,
			IsVisible: action.ElementMeta.IsVisible,
			IsEnabled: action.ElementMeta.IsEnabled,
			Role:      action.ElementMeta.Role,
			AriaLabel: action.ElementMeta.AriaLabel,
		}
		if len(action.ElementMeta.Attributes) > 0 {
			meta.ElementSnapshot.Attributes = action.ElementMeta.Attributes
		}
	}

	return meta
}

// buildRecordingTelemetry creates ActionTelemetry from a RecordedAction.
func buildRecordingTelemetry(action *livecapture.RecordedAction) *basdomain.ActionTelemetry {
	tel := &basdomain.ActionTelemetry{
		Url: action.URL,
	}

	if action.FrameID != "" {
		tel.FrameId = &action.FrameID
	}

	// Add bounding box
	if action.BoundingBox != nil {
		tel.ElementBoundingBox = &basbase.BoundingBox{
			X:      action.BoundingBox.X,
			Y:      action.BoundingBox.Y,
			Width:  action.BoundingBox.Width,
			Height: action.BoundingBox.Height,
		}
	}

	// Add cursor position
	if action.CursorPos != nil {
		tel.CursorPosition = &basbase.Point{
			X: action.CursorPos.X,
			Y: action.CursorPos.Y,
		}
	}

	return tel
}

// buildRecordingEventContext creates EventContext with recording origin.
// EventContext is the unified context type that replaces RecordingContext/ExecutionContext.
func buildRecordingEventContext(action *livecapture.RecordedAction) *basbase.EventContext {
	// Determine if user confirmation is needed
	needsConfirmation := false
	if action.Selector != nil && len(action.Selector.Candidates) > 0 {
		needsConfirmation = len(action.Selector.Candidates) > 1 || action.Confidence < 0.8
	}

	source := basbase.RecordingSource_RECORDING_SOURCE_AUTO
	ctx := &basbase.EventContext{
		Origin:            &basbase.EventContext_SessionId{SessionId: action.SessionID},
		Source:            &source,
		NeedsConfirmation: &needsConfirmation,
	}

	return ctx
}

// mapRecordingActionType converts an action type string to ActionType enum.
// Delegates to typeconv.StringToActionType for the canonical implementation.
func mapRecordingActionType(actionType string) basactions.ActionType {
	return typeconv.StringToActionType(actionType)
}

// generateRecordingLabel creates a human-readable label for an action.
func generateRecordingLabel(action *livecapture.RecordedAction) string {
	actionType := action.ActionType

	// Try to create a descriptive label from element info
	if action.ElementMeta != nil {
		tag := action.ElementMeta.TagName
		text := action.ElementMeta.InnerText
		id := action.ElementMeta.ID
		ariaLabel := action.ElementMeta.AriaLabel

		if ariaLabel != "" {
			return capitalize(actionType) + ": " + truncate(ariaLabel, 30)
		}
		if text != "" {
			return capitalize(actionType) + ": \"" + truncate(text, 30) + "\""
		}
		if id != "" {
			return capitalize(actionType) + ": #" + id
		}
		return capitalize(actionType) + ": " + tag
	}

	// Special case for navigate
	if actionType == "navigate" {
		if targetURL, ok := action.Payload["targetUrl"].(string); ok {
			return "Navigate to " + truncate(targetURL, 40)
		}
		return "Navigate to " + truncate(action.URL, 40)
	}

	return capitalize(actionType)
}

// capitalize capitalizes the first letter of a string.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// truncate truncates a string to maxLen and adds "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// extractInt32FromPayload extracts an int32 value from a payload map.
func extractInt32FromPayload(payload map[string]any, key string) (int32, bool) {
	raw, ok := payload[key]
	if !ok {
		return 0, false
	}
	switch v := raw.(type) {
	case int:
		return int32(v), true
	case int32:
		return v, true
	case int64:
		return int32(v), true
	case float64:
		return int32(v), true
	case float32:
		return int32(v), true
	default:
		return 0, false
	}
}
