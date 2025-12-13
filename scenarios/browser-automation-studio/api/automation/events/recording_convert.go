// Package events provides conversion utilities for recording actions to the
// unified browser_automation_studio_v1.TimelineEvent proto format.
//
// This enables recording telemetry to use the same data structure as execution
// telemetry, supporting the shared Record/Execute UX on the timeline.
package events

import (
	"strings"
	"time"

	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RecordedAction represents an action captured during recording.
// This mirrors the JSON structure sent from playwright-driver.
type RecordedAction struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"sessionId"`
	SequenceNum int                    `json:"sequenceNum"`
	Timestamp   string                 `json:"timestamp"`
	DurationMs  int                    `json:"durationMs,omitempty"`
	ActionType  string                 `json:"actionType"`
	Confidence  float64                `json:"confidence"`
	Selector    *SelectorSet           `json:"selector,omitempty"`
	ElementMeta *ElementMeta           `json:"elementMeta,omitempty"`
	BoundingBox *RecBoundingBox        `json:"boundingBox,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
	URL         string                 `json:"url"`
	FrameID     string                 `json:"frameId,omitempty"`
	CursorPos   *RecPoint              `json:"cursorPos,omitempty"`
}

// SelectorSet contains multiple selector strategies for resilience.
type SelectorSet struct {
	Primary    string              `json:"primary"`
	Candidates []SelectorCandidate `json:"candidates"`
}

// SelectorCandidate is a single selector with metadata.
type SelectorCandidate struct {
	Type        string  `json:"type"`
	Value       string  `json:"value"`
	Confidence  float64 `json:"confidence"`
	Specificity int     `json:"specificity"`
}

// ElementMeta captures information about the target element.
type ElementMeta struct {
	TagName    string            `json:"tagName"`
	ID         string            `json:"id,omitempty"`
	ClassName  string            `json:"className,omitempty"`
	InnerText  string            `json:"innerText,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	IsVisible  bool              `json:"isVisible"`
	IsEnabled  bool              `json:"isEnabled"`
	Role       string            `json:"role,omitempty"`
	AriaLabel  string            `json:"ariaLabel,omitempty"`
}

// RecBoundingBox for element position on screen.
type RecBoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// RecPoint represents a 2D coordinate.
type RecPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// RecordedActionToTimelineEvent converts a RecordedAction from the recording
// controller to the unified TimelineEvent proto format.
func RecordedActionToTimelineEvent(action *RecordedAction) *basv1.TimelineEvent {
	if action == nil {
		return nil
	}

	// Build action definition
	actionDef := buildRecordingActionDefinition(action)

	// Build telemetry
	telemetry := buildRecordingTelemetry(action)

	// Build recording-specific data
	recording := buildRecordingEventData(action)

	event := &basv1.TimelineEvent{
		Id:          action.ID,
		SequenceNum: int32(action.SequenceNum),
		Action:      actionDef,
		Telemetry:   telemetry,
		ModeData:    &basv1.TimelineEvent_Recording{Recording: recording},
	}

	// Parse and set timestamp
	if action.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, action.Timestamp); err == nil {
			event.Timestamp = timestamppb.New(ts)
		} else if ts, err := time.Parse(time.RFC3339, action.Timestamp); err == nil {
			event.Timestamp = timestamppb.New(ts)
		}
	}

	// Set duration if present
	if action.DurationMs > 0 {
		durationMs := int32(action.DurationMs)
		event.DurationMs = &durationMs
	}

	return event
}

// buildRecordingActionDefinition creates an ActionDefinition from a RecordedAction.
func buildRecordingActionDefinition(action *RecordedAction) *basv1.ActionDefinition {
	actionType := mapRecordingActionType(action.ActionType)

	def := &basv1.ActionDefinition{
		Type: actionType,
	}

	// Get selector from the action
	selector := ""
	if action.Selector != nil && action.Selector.Primary != "" {
		selector = action.Selector.Primary
	}

	// Build params based on action type
	switch actionType {
	case basv1.ActionType_ACTION_TYPE_NAVIGATE:
		url := action.URL
		if targetURL, ok := action.Payload["targetUrl"].(string); ok {
			url = targetURL
		}
		def.Params = &basv1.ActionDefinition_Navigate{
			Navigate: &basv1.NavigateParams{
				Url: url,
			},
		}

	case basv1.ActionType_ACTION_TYPE_CLICK:
		clickParams := &basv1.ClickParams{
			Selector: selector,
		}
		if button, ok := action.Payload["button"].(string); ok {
			clickParams.Button = &button
		}
		if clickCount, ok := extractInt32FromPayload(action.Payload, "clickCount"); ok {
			clickParams.ClickCount = &clickCount
		}
		if modifiers, ok := action.Payload["modifiers"].([]any); ok {
			for _, m := range modifiers {
				if s, ok := m.(string); ok {
					clickParams.Modifiers = append(clickParams.Modifiers, s)
				}
			}
		}
		def.Params = &basv1.ActionDefinition_Click{Click: clickParams}

	case basv1.ActionType_ACTION_TYPE_INPUT:
		inputParams := &basv1.InputParams{
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
		def.Params = &basv1.ActionDefinition_Input{Input: inputParams}

	case basv1.ActionType_ACTION_TYPE_SCROLL:
		scrollParams := &basv1.ScrollParams{}
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
		def.Params = &basv1.ActionDefinition_Scroll{Scroll: scrollParams}

	case basv1.ActionType_ACTION_TYPE_HOVER:
		def.Params = &basv1.ActionDefinition_Hover{
			Hover: &basv1.HoverParams{Selector: selector},
		}

	case basv1.ActionType_ACTION_TYPE_FOCUS:
		def.Params = &basv1.ActionDefinition_Focus{
			Focus: &basv1.FocusParams{Selector: selector},
		}

	case basv1.ActionType_ACTION_TYPE_BLUR:
		blurParams := &basv1.BlurParams{}
		if selector != "" {
			blurParams.Selector = &selector
		}
		def.Params = &basv1.ActionDefinition_Blur{Blur: blurParams}

	case basv1.ActionType_ACTION_TYPE_SELECT:
		selectParams := &basv1.SelectParams{Selector: selector}
		if value, ok := action.Payload["value"].(string); ok {
			selectParams.SelectBy = &basv1.SelectParams_Value{Value: value}
		} else if label, ok := action.Payload["selectedText"].(string); ok {
			selectParams.SelectBy = &basv1.SelectParams_Label{Label: label}
		} else if index, ok := extractInt32FromPayload(action.Payload, "selectedIndex"); ok {
			selectParams.SelectBy = &basv1.SelectParams_Index{Index: index}
		}
		def.Params = &basv1.ActionDefinition_SelectOption{SelectOption: selectParams}

	case basv1.ActionType_ACTION_TYPE_KEYBOARD:
		keyboardParams := &basv1.KeyboardParams{}
		if key, ok := action.Payload["key"].(string); ok {
			keyboardParams.Key = &key
		}
		if modifiers, ok := action.Payload["modifiers"].([]any); ok {
			for _, m := range modifiers {
				if s, ok := m.(string); ok {
					keyboardParams.Modifiers = append(keyboardParams.Modifiers, s)
				}
			}
		}
		def.Params = &basv1.ActionDefinition_Keyboard{Keyboard: keyboardParams}

	case basv1.ActionType_ACTION_TYPE_WAIT:
		waitParams := &basv1.WaitParams{}
		if ms, ok := extractInt32FromPayload(action.Payload, "ms"); ok {
			waitParams.WaitFor = &basv1.WaitParams_DurationMs{DurationMs: ms}
		} else if waitSelector, ok := action.Payload["selector"].(string); ok {
			waitParams.WaitFor = &basv1.WaitParams_Selector{Selector: waitSelector}
		}
		def.Params = &basv1.ActionDefinition_Wait{Wait: waitParams}

	case basv1.ActionType_ACTION_TYPE_ASSERT:
		assertParams := &basv1.AssertParams{
			Selector: selector,
			Mode:     "exists",
		}
		def.Params = &basv1.ActionDefinition_Assert{Assert: assertParams}

	case basv1.ActionType_ACTION_TYPE_SCREENSHOT:
		screenshotParams := &basv1.ScreenshotParams{}
		if fullPage, ok := action.Payload["fullPage"].(bool); ok {
			screenshotParams.FullPage = &fullPage
		}
		def.Params = &basv1.ActionDefinition_Screenshot{Screenshot: screenshotParams}

	case basv1.ActionType_ACTION_TYPE_EVALUATE:
		evalParams := &basv1.EvaluateParams{}
		if expr, ok := action.Payload["expression"].(string); ok {
			evalParams.Expression = expr
		}
		def.Params = &basv1.ActionDefinition_Evaluate{Evaluate: evalParams}

	default:
		// Default to click for unknown types
		def.Params = &basv1.ActionDefinition_Click{
			Click: &basv1.ClickParams{Selector: selector},
		}
	}

	// Add metadata
	def.Metadata = buildRecordingMetadata(action)

	return def
}

// buildRecordingMetadata creates ActionMetadata from a RecordedAction.
func buildRecordingMetadata(action *RecordedAction) *basv1.ActionMetadata {
	meta := &basv1.ActionMetadata{}

	// Generate label from element info
	label := generateRecordingLabel(action)
	meta.Label = &label

	// Add confidence
	if action.Confidence > 0 {
		meta.Confidence = &action.Confidence
	}

	// Add recorded timestamp
	if action.Timestamp != "" {
		if ts, err := time.Parse(time.RFC3339Nano, action.Timestamp); err == nil {
			meta.RecordedAt = timestamppb.New(ts)
		} else if ts, err := time.Parse(time.RFC3339, action.Timestamp); err == nil {
			meta.RecordedAt = timestamppb.New(ts)
		}
	}

	// Add bounding box
	if action.BoundingBox != nil {
		meta.RecordedBoundingBox = &basv1.BoundingBox{
			X:      action.BoundingBox.X,
			Y:      action.BoundingBox.Y,
			Width:  action.BoundingBox.Width,
			Height: action.BoundingBox.Height,
		}
	}

	// Add selector candidates
	if action.Selector != nil && len(action.Selector.Candidates) > 0 {
		meta.SelectorCandidates = make([]*basv1.SelectorCandidate, 0, len(action.Selector.Candidates))
		for _, c := range action.Selector.Candidates {
			meta.SelectorCandidates = append(meta.SelectorCandidates, &basv1.SelectorCandidate{
				Type:        c.Type,
				Value:       c.Value,
				Confidence:  c.Confidence,
				Specificity: int32(c.Specificity),
			})
		}
	}

	// Add element snapshot (uses ElementMeta from record_mode.proto)
	if action.ElementMeta != nil {
		meta.ElementSnapshot = &basv1.ElementMeta{
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
func buildRecordingTelemetry(action *RecordedAction) *basv1.ActionTelemetry {
	tel := &basv1.ActionTelemetry{
		Url: action.URL,
	}

	if action.FrameID != "" {
		tel.FrameId = &action.FrameID
	}

	// Add bounding box
	if action.BoundingBox != nil {
		tel.ElementBoundingBox = &basv1.BoundingBox{
			X:      action.BoundingBox.X,
			Y:      action.BoundingBox.Y,
			Width:  action.BoundingBox.Width,
			Height: action.BoundingBox.Height,
		}
	}

	// Add cursor position
	if action.CursorPos != nil {
		tel.CursorPosition = &basv1.Point{
			X: action.CursorPos.X,
			Y: action.CursorPos.Y,
		}
	}

	return tel
}

// buildRecordingEventData creates recording-specific metadata.
func buildRecordingEventData(action *RecordedAction) *basv1.RecordingEventData {
	recording := &basv1.RecordingEventData{
		SessionId:         action.SessionID,
		Source:            "auto",
		NeedsConfirmation: false,
	}

	// Add selector candidates
	if action.Selector != nil && len(action.Selector.Candidates) > 0 {
		recording.SelectorCandidates = make([]*basv1.SelectorCandidate, 0, len(action.Selector.Candidates))
		for _, c := range action.Selector.Candidates {
			recording.SelectorCandidates = append(recording.SelectorCandidates, &basv1.SelectorCandidate{
				Type:        c.Type,
				Value:       c.Value,
				Confidence:  c.Confidence,
				Specificity: int32(c.Specificity),
			})
		}

		// Needs confirmation if multiple candidates or low confidence
		recording.NeedsConfirmation = len(action.Selector.Candidates) > 1 || action.Confidence < 0.8
	}

	return recording
}

// mapRecordingActionType converts an action type string to ActionType enum.
func mapRecordingActionType(actionType string) basv1.ActionType {
	switch strings.ToLower(strings.TrimSpace(actionType)) {
	case "navigate", "goto":
		return basv1.ActionType_ACTION_TYPE_NAVIGATE
	case "click":
		return basv1.ActionType_ACTION_TYPE_CLICK
	case "input", "type", "fill":
		return basv1.ActionType_ACTION_TYPE_INPUT
	case "wait":
		return basv1.ActionType_ACTION_TYPE_WAIT
	case "assert":
		return basv1.ActionType_ACTION_TYPE_ASSERT
	case "scroll":
		return basv1.ActionType_ACTION_TYPE_SCROLL
	case "select":
		return basv1.ActionType_ACTION_TYPE_SELECT
	case "evaluate", "eval":
		return basv1.ActionType_ACTION_TYPE_EVALUATE
	case "keyboard", "keypress", "press":
		return basv1.ActionType_ACTION_TYPE_KEYBOARD
	case "hover":
		return basv1.ActionType_ACTION_TYPE_HOVER
	case "screenshot":
		return basv1.ActionType_ACTION_TYPE_SCREENSHOT
	case "focus":
		return basv1.ActionType_ACTION_TYPE_FOCUS
	case "blur":
		return basv1.ActionType_ACTION_TYPE_BLUR
	default:
		return basv1.ActionType_ACTION_TYPE_UNSPECIFIED
	}
}

// generateRecordingLabel creates a human-readable label for an action.
func generateRecordingLabel(action *RecordedAction) string {
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
