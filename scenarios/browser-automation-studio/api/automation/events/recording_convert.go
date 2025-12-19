// Package events provides conversion utilities for recording actions to the
// unified bastimeline.TimelineEntry proto format.
//
// This enables recording telemetry to use the same data structure as execution
// telemetry, supporting the shared Record/Execute UX on the timeline.
//
// See "UNIFIED RECORDING/EXECUTION MODEL" in shared.proto for design rationale.
package events

import (
	"time"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
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
			entry.Timestamp = contracts.TimeToTimestamp(ts)
		} else if ts, err := time.Parse(time.RFC3339, action.Timestamp); err == nil {
			entry.Timestamp = contracts.TimeToTimestamp(ts)
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
// Uses typeconv builders to avoid code duplication with the compiler package.
func buildRecordingActionDefinition(action *livecapture.RecordedAction) *basactions.ActionDefinition {
	actionType := enums.StringToActionType(action.ActionType)

	def := &basactions.ActionDefinition{
		Type: actionType,
	}

	// Convert RecordedAction to params map format expected by typeconv builders
	params := buildParamsFromRecordedAction(action)

	// Build typed params using shared typeconv builders
	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		def.Params = &basactions.ActionDefinition_Navigate{
			Navigate: typeconv.BuildNavigateParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_CLICK:
		def.Params = &basactions.ActionDefinition_Click{
			Click: typeconv.BuildClickParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_INPUT:
		def.Params = &basactions.ActionDefinition_Input{
			Input: typeconv.BuildInputParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		def.Params = &basactions.ActionDefinition_Scroll{
			Scroll: typeconv.BuildScrollParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_HOVER:
		def.Params = &basactions.ActionDefinition_Hover{
			Hover: typeconv.BuildHoverParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		def.Params = &basactions.ActionDefinition_Focus{
			Focus: typeconv.BuildFocusParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_BLUR:
		def.Params = &basactions.ActionDefinition_Blur{
			Blur: typeconv.BuildBlurParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SELECT:
		def.Params = &basactions.ActionDefinition_SelectOption{
			SelectOption: typeconv.BuildSelectParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		def.Params = &basactions.ActionDefinition_Keyboard{
			Keyboard: typeconv.BuildKeyboardParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_WAIT:
		def.Params = &basactions.ActionDefinition_Wait{
			Wait: typeconv.BuildWaitParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		def.Params = &basactions.ActionDefinition_Assert{
			Assert: typeconv.BuildAssertParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		def.Params = &basactions.ActionDefinition_Screenshot{
			Screenshot: typeconv.BuildScreenshotParams(params),
		}
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		def.Params = &basactions.ActionDefinition_Evaluate{
			Evaluate: typeconv.BuildEvaluateParams(params),
		}
	default:
		// Default to click for unknown types
		def.Params = &basactions.ActionDefinition_Click{
			Click: typeconv.BuildClickParams(params),
		}
	}

	// Add metadata (recording-specific, not delegated to typeconv)
	def.Metadata = buildRecordingMetadata(action)

	return def
}

// buildParamsFromRecordedAction normalizes a RecordedAction into the params map
// format expected by typeconv builders. This handles recording-specific field
// names and aliases.
func buildParamsFromRecordedAction(action *livecapture.RecordedAction) map[string]any {
	params := make(map[string]any)

	// Copy all payload fields
	for k, v := range action.Payload {
		params[k] = v
	}

	// Add selector if present
	if action.Selector != nil && action.Selector.Primary != "" {
		params["selector"] = action.Selector.Primary
	}

	// Normalize recording-specific field names to typeconv-expected names:

	// Navigate: prefer targetUrl from payload, fall back to action.URL
	if _, hasURL := params["url"]; !hasURL {
		if targetURL, ok := params["targetUrl"].(string); ok {
			params["url"] = targetURL
		} else {
			params["url"] = action.URL
		}
	}

	// Select: normalize "selectedText" → "label", "selectedIndex" → "index"
	if selectedText, ok := params["selectedText"].(string); ok && selectedText != "" {
		if _, hasLabel := params["label"]; !hasLabel {
			params["label"] = selectedText
		}
	}
	if selectedIndex, ok := params["selectedIndex"]; ok {
		if _, hasIndex := params["index"]; !hasIndex {
			params["index"] = selectedIndex
		}
	}

	// Scroll: normalize "scrollX" → "x", "scrollY" → "y"
	if scrollX, ok := params["scrollX"]; ok {
		if _, hasX := params["x"]; !hasX {
			params["x"] = scrollX
		}
	}
	if scrollY, ok := params["scrollY"]; ok {
		if _, hasY := params["y"]; !hasY {
			params["y"] = scrollY
		}
	}

	// Wait: normalize "ms" → "durationMs"
	if ms, ok := params["ms"]; ok {
		if _, hasDuration := params["durationMs"]; !hasDuration {
			params["durationMs"] = ms
		}
	}

	// Input: normalize "delay" → "delayMs"
	if delay, ok := params["delay"]; ok {
		if _, hasDelayMs := params["delayMs"]; !hasDelayMs {
			params["delayMs"] = delay
		}
	}

	// Assert: default mode to "exists" if not specified (recording behavior)
	if _, hasMode := params["mode"]; !hasMode {
		if _, hasAssertMode := params["assertMode"]; !hasAssertMode {
			params["mode"] = "exists"
		}
	}

	return params
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
			meta.CapturedAt = contracts.TimeToTimestamp(ts)
		} else if ts, err := time.Parse(time.RFC3339, action.Timestamp); err == nil {
			meta.CapturedAt = contracts.TimeToTimestamp(ts)
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
				Type:        enums.StringToSelectorType(c.Type),
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
			return Capitalize(actionType) + ": " + Truncate(ariaLabel, 30)
		}
		if text != "" {
			return Capitalize(actionType) + ": \"" + Truncate(text, 30) + "\""
		}
		if id != "" {
			return Capitalize(actionType) + ": #" + id
		}
		return Capitalize(actionType) + ": " + tag
	}

	// Special case for navigate
	if actionType == "navigate" {
		if targetURL, ok := action.Payload["targetUrl"].(string); ok {
			return "Navigate to " + Truncate(targetURL, 40)
		}
		return "Navigate to " + Truncate(action.URL, 40)
	}

	return Capitalize(actionType)
}

// extractInt32FromPayload wraps the shared helper for backward compatibility.
// Deprecated: Use ExtractInt32FromPayload directly.
func extractInt32FromPayload(payload map[string]any, key string) (int32, bool) {
	return ExtractInt32FromPayload(payload, key)
}
