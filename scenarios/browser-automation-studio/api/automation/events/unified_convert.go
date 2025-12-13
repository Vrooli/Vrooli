// Package events provides conversion utilities between legacy contracts.StepOutcome
// and the unified browser_automation_studio_v1.TimelineEvent proto format.
//
// This enables the execution engine to produce TimelineEvent messages that can be
// streamed to the UI via WebSocket, supporting the shared Record/Execute UX.
package events

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StepOutcomeToTimelineEvent converts a legacy StepOutcome to the unified TimelineEvent format.
// This enables execution telemetry to use the same data structure as recording telemetry.
func StepOutcomeToTimelineEvent(outcome contracts.StepOutcome, executionID uuid.UUID) *basv1.TimelineEvent {
	// Generate event ID from execution and step
	eventID := fmt.Sprintf("%s-step-%d-attempt-%d", executionID.String(), outcome.StepIndex, outcome.Attempt)

	// Build action definition from step outcome
	action := buildActionDefinition(outcome)

	// Build telemetry from step outcome
	telemetry := buildActionTelemetry(outcome)

	// Build execution-specific data
	execution := buildExecutionEventData(outcome, executionID)

	event := &basv1.TimelineEvent{
		Id:          eventID,
		SequenceNum: int32(outcome.StepIndex),
		Action:      action,
		Telemetry:   telemetry,
		ModeData:    &basv1.TimelineEvent_Execution{Execution: execution},
	}

	// Set timestamp
	if !outcome.StartedAt.IsZero() {
		event.Timestamp = timestamppb.New(outcome.StartedAt)
	}

	// Set duration
	if outcome.DurationMs > 0 {
		durationMs := int32(outcome.DurationMs)
		event.DurationMs = &durationMs
	}

	return event
}

// buildActionDefinition creates an ActionDefinition from the step outcome's type and params.
func buildActionDefinition(outcome contracts.StepOutcome) *basv1.ActionDefinition {
	actionType := mapStepTypeToActionType(outcome.StepType)

	def := &basv1.ActionDefinition{
		Type: actionType,
	}

	// Set params based on action type
	// Note: For execution, we don't have full params - the instruction params were
	// consumed by the engine. We reconstruct what we can from the outcome.
	switch actionType {
	case basv1.ActionType_ACTION_TYPE_NAVIGATE:
		def.Params = &basv1.ActionDefinition_Navigate{
			Navigate: &basv1.NavigateParams{
				Url: outcome.FinalURL,
			},
		}
	case basv1.ActionType_ACTION_TYPE_CLICK:
		// Click params come from instruction, but we can note position from outcome
		def.Params = &basv1.ActionDefinition_Click{
			Click: &basv1.ClickParams{},
		}
	case basv1.ActionType_ACTION_TYPE_INPUT:
		def.Params = &basv1.ActionDefinition_Input{
			Input: &basv1.InputParams{},
		}
	case basv1.ActionType_ACTION_TYPE_WAIT:
		def.Params = &basv1.ActionDefinition_Wait{
			Wait: &basv1.WaitParams{},
		}
	case basv1.ActionType_ACTION_TYPE_ASSERT:
		if outcome.Assertion != nil {
			def.Params = &basv1.ActionDefinition_Assert{
				Assert: &basv1.AssertParams{
					Selector: outcome.Assertion.Selector,
					Mode:     outcome.Assertion.Mode,
					Negated:  &outcome.Assertion.Negated,
				},
			}
		}
	case basv1.ActionType_ACTION_TYPE_SCROLL:
		def.Params = &basv1.ActionDefinition_Scroll{
			Scroll: &basv1.ScrollParams{},
		}
	case basv1.ActionType_ACTION_TYPE_HOVER:
		def.Params = &basv1.ActionDefinition_Hover{
			Hover: &basv1.HoverParams{},
		}
	case basv1.ActionType_ACTION_TYPE_KEYBOARD:
		def.Params = &basv1.ActionDefinition_Keyboard{
			Keyboard: &basv1.KeyboardParams{},
		}
	case basv1.ActionType_ACTION_TYPE_SCREENSHOT:
		def.Params = &basv1.ActionDefinition_Screenshot{
			Screenshot: &basv1.ScreenshotParams{},
		}
	case basv1.ActionType_ACTION_TYPE_SELECT:
		def.Params = &basv1.ActionDefinition_SelectOption{
			SelectOption: &basv1.SelectParams{},
		}
	case basv1.ActionType_ACTION_TYPE_EVALUATE:
		def.Params = &basv1.ActionDefinition_Evaluate{
			Evaluate: &basv1.EvaluateParams{},
		}
	case basv1.ActionType_ACTION_TYPE_FOCUS:
		def.Params = &basv1.ActionDefinition_Focus{
			Focus: &basv1.FocusParams{},
		}
	case basv1.ActionType_ACTION_TYPE_BLUR:
		def.Params = &basv1.ActionDefinition_Blur{
			Blur: &basv1.BlurParams{},
		}
	}

	// Add metadata with node info
	def.Metadata = &basv1.ActionMetadata{
		Label: &outcome.NodeID,
	}

	return def
}

// buildActionTelemetry creates ActionTelemetry from the step outcome.
func buildActionTelemetry(outcome contracts.StepOutcome) *basv1.ActionTelemetry {
	tel := &basv1.ActionTelemetry{
		Url: outcome.FinalURL,
	}

	// Add screenshot if present
	// Note: TimelineScreenshot is the proto type which has ArtifactId, Url, etc.
	// The contracts.Screenshot has raw Data, MediaType, etc.
	// For real-time streaming, we don't have artifact URLs yet - those come from storage.
	// We include what we can (dimensions, content type).
	if outcome.Screenshot != nil {
		tel.Screenshot = &basv1.TimelineScreenshot{
			Width:       int32(outcome.Screenshot.Width),
			Height:      int32(outcome.Screenshot.Height),
			ContentType: outcome.Screenshot.MediaType,
		}
	}

	// Add DOM snapshot preview
	if outcome.DOMSnapshot != nil {
		if outcome.DOMSnapshot.Preview != "" {
			tel.DomSnapshotPreview = &outcome.DOMSnapshot.Preview
		}
		if outcome.DOMSnapshot.HTML != "" {
			tel.DomSnapshotHtml = &outcome.DOMSnapshot.HTML
		}
	}

	// Add bounding box
	if outcome.ElementBoundingBox != nil {
		tel.ElementBoundingBox = convertBoundingBoxToProto(outcome.ElementBoundingBox)
	}

	// Add click position
	if outcome.ClickPosition != nil {
		tel.ClickPosition = convertPointToProto(outcome.ClickPosition)
	}

	// Add cursor trail
	if len(outcome.CursorTrail) > 0 {
		tel.CursorTrail = make([]*basv1.Point, 0, len(outcome.CursorTrail))
		for _, pos := range outcome.CursorTrail {
			tel.CursorTrail = append(tel.CursorTrail, convertPointToProto(&pos.Point))
		}
	}

	// Add highlight regions
	if len(outcome.HighlightRegions) > 0 {
		tel.HighlightRegions = make([]*basv1.HighlightRegion, 0, len(outcome.HighlightRegions))
		for _, r := range outcome.HighlightRegions {
			tel.HighlightRegions = append(tel.HighlightRegions, &basv1.HighlightRegion{
				Selector:    r.Selector,
				BoundingBox: convertBoundingBoxToProto(r.BoundingBox),
				Padding:     int32(r.Padding),
				Color:       r.Color,
			})
		}
	}

	// Add mask regions
	if len(outcome.MaskRegions) > 0 {
		tel.MaskRegions = make([]*basv1.MaskRegion, 0, len(outcome.MaskRegions))
		for _, r := range outcome.MaskRegions {
			tel.MaskRegions = append(tel.MaskRegions, &basv1.MaskRegion{
				Selector:    r.Selector,
				BoundingBox: convertBoundingBoxToProto(r.BoundingBox),
				Opacity:     r.Opacity,
			})
		}
	}

	// Add zoom factor
	if outcome.ZoomFactor != 0 {
		tel.ZoomFactor = &outcome.ZoomFactor
	}

	// Add console logs
	if len(outcome.ConsoleLogs) > 0 {
		tel.ConsoleLogs = make([]*basv1.ConsoleLogEntryUnified, 0, len(outcome.ConsoleLogs))
		for _, log := range outcome.ConsoleLogs {
			entry := &basv1.ConsoleLogEntryUnified{
				Level: log.Type, // Type maps to Level (log, warn, error, etc.)
				Text:  log.Text,
			}
			if log.Stack != "" {
				entry.Stack = &log.Stack
			}
			if log.Location != "" {
				entry.Location = &log.Location
			}
			if !log.Timestamp.IsZero() {
				entry.Timestamp = timestamppb.New(log.Timestamp)
			}
			tel.ConsoleLogs = append(tel.ConsoleLogs, entry)
		}
	}

	// Add network events
	if len(outcome.Network) > 0 {
		tel.NetworkEvents = make([]*basv1.NetworkEventUnified, 0, len(outcome.Network))
		for _, net := range outcome.Network {
			event := &basv1.NetworkEventUnified{
				Type: net.Type,
				Url:  net.URL,
			}
			if net.Method != "" {
				event.Method = &net.Method
			}
			if net.ResourceType != "" {
				event.ResourceType = &net.ResourceType
			}
			if net.Status != 0 {
				status := int32(net.Status)
				event.Status = &status
			}
			if net.OK {
				event.Ok = &net.OK
			}
			if net.Failure != "" {
				event.Failure = &net.Failure
			}
			if !net.Timestamp.IsZero() {
				event.Timestamp = timestamppb.New(net.Timestamp)
			}
			tel.NetworkEvents = append(tel.NetworkEvents, event)
		}
	}

	return tel
}

// buildExecutionEventData creates execution-specific metadata for the timeline event.
func buildExecutionEventData(outcome contracts.StepOutcome, executionID uuid.UUID) *basv1.ExecutionEventData {
	exec := &basv1.ExecutionEventData{
		ExecutionId: executionID.String(),
		NodeId:      outcome.NodeID,
		Success:     outcome.Success,
		Attempt:     int32(outcome.Attempt),
	}

	// Add failure information
	if outcome.Failure != nil {
		if outcome.Failure.Message != "" {
			exec.Error = &outcome.Failure.Message
		}
		if outcome.Failure.Code != "" {
			exec.ErrorCode = &outcome.Failure.Code
		}
	}

	// Add assertion result
	if outcome.Assertion != nil {
		assertionMsg := outcome.Assertion.Message
		exec.Assertion = &basv1.AssertionResultProto{
			Success:       outcome.Assertion.Success,
			Mode:          outcome.Assertion.Mode,
			Selector:      outcome.Assertion.Selector,
			Negated:       outcome.Assertion.Negated,
			CaseSensitive: outcome.Assertion.CaseSensitive,
		}
		if assertionMsg != "" {
			exec.Assertion.Message = &assertionMsg
		}
		if outcome.Assertion.Expected != nil {
			exec.Assertion.Expected = anyToJsonValue(outcome.Assertion.Expected)
		}
		if outcome.Assertion.Actual != nil {
			exec.Assertion.Actual = anyToJsonValue(outcome.Assertion.Actual)
		}
	}

	// Add extracted data
	if len(outcome.ExtractedData) > 0 {
		exec.ExtractedData = make(map[string]*commonv1.JsonValue, len(outcome.ExtractedData))
		for k, v := range outcome.ExtractedData {
			if jsonVal := anyToJsonValue(v); jsonVal != nil {
				exec.ExtractedData[k] = jsonVal
			}
		}
	}

	return exec
}

// mapStepTypeToActionType converts a step type string to ActionType enum.
func mapStepTypeToActionType(stepType string) basv1.ActionType {
	switch strings.ToLower(strings.TrimSpace(stepType)) {
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
	case "keyboard", "press":
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

// convertBoundingBoxToProto converts a contracts.BoundingBox to proto format.
func convertBoundingBoxToProto(b *contracts.BoundingBox) *basv1.BoundingBox {
	if b == nil {
		return nil
	}
	return &basv1.BoundingBox{
		X:      b.X,
		Y:      b.Y,
		Width:  b.Width,
		Height: b.Height,
	}
}

// convertPointToProto converts a contracts.Point to proto format.
func convertPointToProto(p *contracts.Point) *basv1.Point {
	if p == nil {
		return nil
	}
	return &basv1.Point{
		X: p.X,
		Y: p.Y,
	}
}

// anyToJsonValue converts any Go value to a commonv1.JsonValue proto message.
func anyToJsonValue(v any) *commonv1.JsonValue {
	if v == nil {
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_NullValue{}}
	}
	switch val := v.(type) {
	case bool:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BoolValue{BoolValue: val}}
	case int:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case int64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: val}}
	case uint:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case uint32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case uint64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_IntValue{IntValue: int64(val)}}
	case float32:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: float64(val)}}
	case float64:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_DoubleValue{DoubleValue: val}}
	case string:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: val}}
	case []byte:
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_BytesValue{BytesValue: val}}
	case map[string]any:
		obj := make(map[string]*commonv1.JsonValue, len(val))
		for k, v := range val {
			if nested := anyToJsonValue(v); nested != nil {
				obj[k] = nested
			}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ObjectValue{
			ObjectValue: &commonv1.JsonObject{Fields: obj},
		}}
	case []any:
		items := make([]*commonv1.JsonValue, 0, len(val))
		for _, item := range val {
			if nested := anyToJsonValue(item); nested != nil {
				items = append(items, nested)
			}
		}
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_ListValue{
			ListValue: &commonv1.JsonList{Values: items},
		}}
	default:
		// Fallback: try to convert to string
		return &commonv1.JsonValue{Kind: &commonv1.JsonValue_StringValue{StringValue: fmt.Sprintf("%v", val)}}
	}
}

// CompiledInstructionToActionDefinition converts a CompiledInstruction to ActionDefinition.
// This is used when we have the original instruction params available.
func CompiledInstructionToActionDefinition(instr contracts.CompiledInstruction) *basv1.ActionDefinition {
	actionType := mapStepTypeToActionType(instr.Type)

	def := &basv1.ActionDefinition{
		Type: actionType,
	}

	// Build params based on action type using instruction params
	switch actionType {
	case basv1.ActionType_ACTION_TYPE_NAVIGATE:
		def.Params = &basv1.ActionDefinition_Navigate{
			Navigate: buildNavigateParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_CLICK:
		def.Params = &basv1.ActionDefinition_Click{
			Click: buildClickParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_INPUT:
		def.Params = &basv1.ActionDefinition_Input{
			Input: buildInputParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_WAIT:
		def.Params = &basv1.ActionDefinition_Wait{
			Wait: buildWaitParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_ASSERT:
		def.Params = &basv1.ActionDefinition_Assert{
			Assert: buildAssertParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_SCROLL:
		def.Params = &basv1.ActionDefinition_Scroll{
			Scroll: buildScrollParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_SELECT:
		def.Params = &basv1.ActionDefinition_SelectOption{
			SelectOption: buildSelectParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_EVALUATE:
		def.Params = &basv1.ActionDefinition_Evaluate{
			Evaluate: buildEvaluateParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_KEYBOARD:
		def.Params = &basv1.ActionDefinition_Keyboard{
			Keyboard: buildKeyboardParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_HOVER:
		def.Params = &basv1.ActionDefinition_Hover{
			Hover: buildHoverParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_SCREENSHOT:
		def.Params = &basv1.ActionDefinition_Screenshot{
			Screenshot: buildScreenshotParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_FOCUS:
		def.Params = &basv1.ActionDefinition_Focus{
			Focus: buildFocusParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_BLUR:
		def.Params = &basv1.ActionDefinition_Blur{
			Blur: buildBlurParams(instr.Params),
		}
	}

	// Add metadata with node info
	def.Metadata = &basv1.ActionMetadata{
		Label: &instr.NodeID,
	}

	return def
}

// Helper functions to build params from instruction maps

func buildNavigateParams(params map[string]any) *basv1.NavigateParams {
	p := &basv1.NavigateParams{}
	if url, ok := params["url"].(string); ok {
		p.Url = url
	}
	if waitFor, ok := params["waitForSelector"].(string); ok {
		p.WaitForSelector = &waitFor
	}
	if timeout, ok := extractInt32(params, "timeoutMs"); ok {
		p.TimeoutMs = &timeout
	}
	if waitUntil, ok := params["waitUntil"].(string); ok {
		p.WaitUntil = &waitUntil
	}
	return p
}

func buildClickParams(params map[string]any) *basv1.ClickParams {
	p := &basv1.ClickParams{}
	if selector, ok := params["selector"].(string); ok {
		p.Selector = selector
	}
	if button, ok := params["button"].(string); ok {
		p.Button = &button
	}
	if clickCount, ok := extractInt32(params, "clickCount"); ok {
		p.ClickCount = &clickCount
	}
	if delay, ok := extractInt32(params, "delayMs"); ok {
		p.DelayMs = &delay
	}
	if modifiers, ok := params["modifiers"].([]any); ok {
		for _, m := range modifiers {
			if s, ok := m.(string); ok {
				p.Modifiers = append(p.Modifiers, s)
			}
		}
	}
	if force, ok := params["force"].(bool); ok {
		p.Force = &force
	}
	return p
}

func buildInputParams(params map[string]any) *basv1.InputParams {
	p := &basv1.InputParams{}
	if selector, ok := params["selector"].(string); ok {
		p.Selector = selector
	}
	if value, ok := params["value"].(string); ok {
		p.Value = value
	}
	if sensitive, ok := params["isSensitive"].(bool); ok {
		p.IsSensitive = &sensitive
	}
	if submit, ok := params["submit"].(bool); ok {
		p.Submit = &submit
	}
	if clear, ok := params["clearFirst"].(bool); ok {
		p.ClearFirst = &clear
	}
	if delay, ok := extractInt32(params, "delayMs"); ok {
		p.DelayMs = &delay
	}
	return p
}

func buildWaitParams(params map[string]any) *basv1.WaitParams {
	p := &basv1.WaitParams{}
	if duration, ok := extractInt32(params, "durationMs"); ok {
		p.WaitFor = &basv1.WaitParams_DurationMs{DurationMs: duration}
	} else if selector, ok := params["selector"].(string); ok {
		p.WaitFor = &basv1.WaitParams_Selector{Selector: selector}
	}
	if state, ok := params["state"].(string); ok {
		p.State = &state
	}
	if timeout, ok := extractInt32(params, "timeoutMs"); ok {
		p.TimeoutMs = &timeout
	}
	return p
}

func buildAssertParams(params map[string]any) *basv1.AssertParams {
	p := &basv1.AssertParams{}
	if selector, ok := params["selector"].(string); ok {
		p.Selector = selector
	}
	if mode, ok := params["mode"].(string); ok {
		p.Mode = mode
	}
	if negated, ok := params["negated"].(bool); ok {
		p.Negated = &negated
	}
	if caseSensitive, ok := params["caseSensitive"].(bool); ok {
		p.CaseSensitive = &caseSensitive
	}
	if attrName, ok := params["attributeName"].(string); ok {
		p.AttributeName = &attrName
	}
	if failureMsg, ok := params["failureMessage"].(string); ok {
		p.FailureMessage = &failureMsg
	}
	return p
}

func buildScrollParams(params map[string]any) *basv1.ScrollParams {
	p := &basv1.ScrollParams{}
	if selector, ok := params["selector"].(string); ok {
		p.Selector = &selector
	}
	if x, ok := extractInt32(params, "x"); ok {
		p.X = &x
	}
	if y, ok := extractInt32(params, "y"); ok {
		p.Y = &y
	}
	if behavior, ok := params["behavior"].(string); ok {
		p.Behavior = &behavior
	}
	return p
}

func buildSelectParams(params map[string]any) *basv1.SelectParams {
	p := &basv1.SelectParams{}
	if selector, ok := params["selector"].(string); ok {
		p.Selector = selector
	}
	if value, ok := params["value"].(string); ok {
		p.SelectBy = &basv1.SelectParams_Value{Value: value}
	} else if label, ok := params["label"].(string); ok {
		p.SelectBy = &basv1.SelectParams_Label{Label: label}
	} else if index, ok := extractInt32(params, "index"); ok {
		p.SelectBy = &basv1.SelectParams_Index{Index: index}
	}
	if timeout, ok := extractInt32(params, "timeoutMs"); ok {
		p.TimeoutMs = &timeout
	}
	return p
}

func buildEvaluateParams(params map[string]any) *basv1.EvaluateParams {
	p := &basv1.EvaluateParams{}
	if expr, ok := params["expression"].(string); ok {
		p.Expression = expr
	}
	if storeResult, ok := params["storeResult"].(string); ok {
		p.StoreResult = &storeResult
	}
	return p
}

func buildKeyboardParams(params map[string]any) *basv1.KeyboardParams {
	p := &basv1.KeyboardParams{}
	if key, ok := params["key"].(string); ok {
		p.Key = &key
	}
	if keys, ok := params["keys"].([]any); ok {
		for _, k := range keys {
			if s, ok := k.(string); ok {
				p.Keys = append(p.Keys, s)
			}
		}
	}
	if modifiers, ok := params["modifiers"].([]any); ok {
		for _, m := range modifiers {
			if s, ok := m.(string); ok {
				p.Modifiers = append(p.Modifiers, s)
			}
		}
	}
	if action, ok := params["action"].(string); ok {
		p.Action = &action
	}
	return p
}

func buildHoverParams(params map[string]any) *basv1.HoverParams {
	p := &basv1.HoverParams{}
	if selector, ok := params["selector"].(string); ok {
		p.Selector = selector
	}
	if timeout, ok := extractInt32(params, "timeoutMs"); ok {
		p.TimeoutMs = &timeout
	}
	return p
}

func buildScreenshotParams(params map[string]any) *basv1.ScreenshotParams {
	p := &basv1.ScreenshotParams{}
	if fullPage, ok := params["fullPage"].(bool); ok {
		p.FullPage = &fullPage
	}
	if selector, ok := params["selector"].(string); ok {
		p.Selector = &selector
	}
	if quality, ok := extractInt32(params, "quality"); ok {
		p.Quality = &quality
	}
	return p
}

func buildFocusParams(params map[string]any) *basv1.FocusParams {
	p := &basv1.FocusParams{}
	if selector, ok := params["selector"].(string); ok {
		p.Selector = selector
	}
	if timeout, ok := extractInt32(params, "timeoutMs"); ok {
		p.TimeoutMs = &timeout
	}
	return p
}

func buildBlurParams(params map[string]any) *basv1.BlurParams {
	p := &basv1.BlurParams{}
	if selector, ok := params["selector"].(string); ok {
		p.Selector = &selector
	}
	if timeout, ok := extractInt32(params, "timeoutMs"); ok {
		p.TimeoutMs = &timeout
	}
	return p
}

// extractInt32 extracts an int32 value from a map with various numeric types.
func extractInt32(params map[string]any, key string) (int32, bool) {
	raw, ok := params[key]
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
