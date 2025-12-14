// Package events provides conversion utilities between legacy contracts.StepOutcome
// and the unified browser_automation_studio_v1.TimelineEntry proto format.
//
// This enables the execution engine to produce TimelineEntry messages that can be
// streamed to the UI via WebSocket, supporting the shared Record/Execute UX.
//
// See "UNIFIED RECORDING/EXECUTION MODEL" in shared.proto for design rationale.
package events

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/params"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StepOutcomeToTimelineEntry converts a legacy StepOutcome to the unified TimelineEntry format.
// This enables execution telemetry to use the same data structure as recording telemetry.
func StepOutcomeToTimelineEntry(outcome contracts.StepOutcome, executionID uuid.UUID) *basv1.TimelineEntry {
	// Generate entry ID from execution and step
	entryID := fmt.Sprintf("%s-step-%d-attempt-%d", executionID.String(), outcome.StepIndex, outcome.Attempt)

	// Build action definition from step outcome
	action := buildActionDefinition(outcome)

	// Build telemetry from step outcome
	telemetry := buildActionTelemetry(outcome)

	// Build event context (unified for recording/execution)
	context := buildEventContext(outcome, executionID)

	stepIndex := int32(outcome.StepIndex)
	entry := &basv1.TimelineEntry{
		Id:          entryID,
		SequenceNum: int32(outcome.StepIndex),
		StepIndex:   &stepIndex,
		Action:      action,
		Telemetry:   telemetry,
		Context:     context,
	}

	// Set node_id if available
	if outcome.NodeID != "" {
		entry.NodeId = &outcome.NodeID
	}

	// Set timestamp
	if !outcome.StartedAt.IsZero() {
		entry.Timestamp = timestamppb.New(outcome.StartedAt)
	}

	// Set duration
	if outcome.DurationMs > 0 {
		durationMs := int32(outcome.DurationMs)
		entry.DurationMs = &durationMs
	}

	return entry
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
					Mode:     params.StringToAssertionMode(outcome.Assertion.Mode),
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
				region := &basv1.HighlightRegion{
					Selector:       r.Selector,
					BoundingBox:    convertBoundingBoxToProto(r.BoundingBox),
					Padding:        r.Padding,
					HighlightColor: r.HighlightColor,
					CustomRgba:     r.CustomRgba,
				}
				tel.HighlightRegions = append(tel.HighlightRegions, region)
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
		tel.ConsoleLogs = make([]*basv1.ConsoleLogEntry, 0, len(outcome.ConsoleLogs))
		for _, log := range outcome.ConsoleLogs {
			entry := &basv1.ConsoleLogEntry{
				Level: typeconv.StringToLogLevel(log.Type),
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
		tel.NetworkEvents = make([]*basv1.NetworkEvent, 0, len(outcome.Network))
		for _, net := range outcome.Network {
			event := &basv1.NetworkEvent{
				Type: typeconv.StringToNetworkEventType(net.Type),
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

// buildEventContext creates the unified EventContext for a timeline entry.
// This is the same structure used for both recording and execution events.
// Note: node_id is set on the parent TimelineEntry, not in EventContext.
func buildEventContext(outcome contracts.StepOutcome, executionID uuid.UUID) *basv1.EventContext {
	ctx := &basv1.EventContext{
		// Set execution_id as the origin (vs session_id for recordings)
		Origin:  &basv1.EventContext_ExecutionId{ExecutionId: executionID.String()},
		Success: &outcome.Success,
	}

	// Add retry status with current attempt
	ctx.RetryStatus = &basv1.RetryStatus{
		CurrentAttempt: int32(outcome.Attempt),
		MaxAttempts:    1, // Default, could be enhanced with actual retry config
		Configured:     outcome.Attempt > 0,
	}

	// Add failure information
	if outcome.Failure != nil {
		if outcome.Failure.Message != "" {
			ctx.Error = &outcome.Failure.Message
		}
		if outcome.Failure.Code != "" {
			ctx.ErrorCode = &outcome.Failure.Code
		}
	}

	// Add assertion result
	if outcome.Assertion != nil {
		assertionMsg := outcome.Assertion.Message
		ctx.Assertion = &basv1.AssertionResult{
			Success:       outcome.Assertion.Success,
			Mode:          typeconv.StringToAssertionMode(outcome.Assertion.Mode),
			Selector:      outcome.Assertion.Selector,
			Negated:       outcome.Assertion.Negated,
			CaseSensitive: outcome.Assertion.CaseSensitive,
		}
		if assertionMsg != "" {
			ctx.Assertion.Message = &assertionMsg
		}
		if outcome.Assertion.Expected != nil {
			ctx.Assertion.Expected = typeconv.AnyToJsonValue(outcome.Assertion.Expected)
		}
		if outcome.Assertion.Actual != nil {
			ctx.Assertion.Actual = typeconv.AnyToJsonValue(outcome.Assertion.Actual)
		}
	}

	// Add extracted data
	if len(outcome.ExtractedData) > 0 {
		ctx.ExtractedData = make(map[string]*commonv1.JsonValue, len(outcome.ExtractedData))
		for k, v := range outcome.ExtractedData {
			if jsonVal := typeconv.AnyToJsonValue(v); jsonVal != nil {
				ctx.ExtractedData[k] = jsonVal
			}
		}
	}

	return ctx
}

// mapStepTypeToActionType converts a step type string to ActionType enum.
// Delegates to typeconv.StringToActionType for the canonical implementation.
func mapStepTypeToActionType(stepType string) basv1.ActionType {
	return typeconv.StringToActionType(stepType)
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
			Navigate: params.BuildNavigateParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_CLICK:
		def.Params = &basv1.ActionDefinition_Click{
			Click: params.BuildClickParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_INPUT:
		def.Params = &basv1.ActionDefinition_Input{
			Input: params.BuildInputParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_WAIT:
		def.Params = &basv1.ActionDefinition_Wait{
			Wait: params.BuildWaitParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_ASSERT:
		def.Params = &basv1.ActionDefinition_Assert{
			Assert: params.BuildAssertParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_SCROLL:
		def.Params = &basv1.ActionDefinition_Scroll{
			Scroll: params.BuildScrollParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_SELECT:
		def.Params = &basv1.ActionDefinition_SelectOption{
			SelectOption: params.BuildSelectParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_EVALUATE:
		def.Params = &basv1.ActionDefinition_Evaluate{
			Evaluate: params.BuildEvaluateParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_KEYBOARD:
		def.Params = &basv1.ActionDefinition_Keyboard{
			Keyboard: params.BuildKeyboardParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_HOVER:
		def.Params = &basv1.ActionDefinition_Hover{
			Hover: params.BuildHoverParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_SCREENSHOT:
		def.Params = &basv1.ActionDefinition_Screenshot{
			Screenshot: params.BuildScreenshotParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_FOCUS:
		def.Params = &basv1.ActionDefinition_Focus{
			Focus: params.BuildFocusParams(instr.Params),
		}
	case basv1.ActionType_ACTION_TYPE_BLUR:
		def.Params = &basv1.ActionDefinition_Blur{
			Blur: params.BuildBlurParams(instr.Params),
		}
	}

	// Add metadata with node info
	def.Metadata = &basv1.ActionMetadata{
		Label: &instr.NodeID,
	}

	return def
}
