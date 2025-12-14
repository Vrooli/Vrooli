// Package events provides conversion utilities between legacy contracts.StepOutcome
// and the unified bastimeline.TimelineEntry proto format.
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
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StepOutcomeToTimelineEntry converts a legacy StepOutcome to the unified TimelineEntry format.
// This enables execution telemetry to use the same data structure as recording telemetry.
func StepOutcomeToTimelineEntry(outcome contracts.StepOutcome, executionID uuid.UUID) *bastimeline.TimelineEntry {
	// Generate entry ID from execution and step
	entryID := fmt.Sprintf("%s-step-%d-attempt-%d", executionID.String(), outcome.StepIndex, outcome.Attempt)

	// Build action definition from step outcome
	action := buildActionDefinition(outcome)

	// Build telemetry from step outcome
	telemetry := buildActionTelemetry(outcome)

	// Build event context (unified for recording/execution)
	context := buildEventContext(outcome, executionID)

	stepIndex := int32(outcome.StepIndex)
	entry := &bastimeline.TimelineEntry{
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
func buildActionDefinition(outcome contracts.StepOutcome) *basactions.ActionDefinition {
	actionType := mapStepTypeToActionType(outcome.StepType)

	def := &basactions.ActionDefinition{
		Type: actionType,
	}

	// Set params based on action type
	// Note: For execution, we don't have full params - the instruction params were
	// consumed by the engine. We reconstruct what we can from the outcome.
	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		def.Params = &basactions.ActionDefinition_Navigate{
			Navigate: &basactions.NavigateParams{
				Url: outcome.FinalURL,
			},
		}
	case basactions.ActionType_ACTION_TYPE_CLICK:
		// Click params come from instruction, but we can note position from outcome
		def.Params = &basactions.ActionDefinition_Click{
			Click: &basactions.ClickParams{},
		}
	case basactions.ActionType_ACTION_TYPE_INPUT:
		def.Params = &basactions.ActionDefinition_Input{
			Input: &basactions.InputParams{},
		}
	case basactions.ActionType_ACTION_TYPE_WAIT:
		def.Params = &basactions.ActionDefinition_Wait{
			Wait: &basactions.WaitParams{},
		}
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		if outcome.Assertion != nil {
			def.Params = &basactions.ActionDefinition_Assert{
				Assert: &basactions.AssertParams{
					Selector: outcome.Assertion.Selector,
					Mode:     params.StringToAssertionMode(outcome.Assertion.Mode),
					Negated:  &outcome.Assertion.Negated,
				},
			}
		}
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		def.Params = &basactions.ActionDefinition_Scroll{
			Scroll: &basactions.ScrollParams{},
		}
	case basactions.ActionType_ACTION_TYPE_HOVER:
		def.Params = &basactions.ActionDefinition_Hover{
			Hover: &basactions.HoverParams{},
		}
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		def.Params = &basactions.ActionDefinition_Keyboard{
			Keyboard: &basactions.KeyboardParams{},
		}
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		def.Params = &basactions.ActionDefinition_Screenshot{
			Screenshot: &basactions.ScreenshotParams{},
		}
	case basactions.ActionType_ACTION_TYPE_SELECT:
		def.Params = &basactions.ActionDefinition_SelectOption{
			SelectOption: &basactions.SelectParams{},
		}
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		def.Params = &basactions.ActionDefinition_Evaluate{
			Evaluate: &basactions.EvaluateParams{},
		}
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		def.Params = &basactions.ActionDefinition_Focus{
			Focus: &basactions.FocusParams{},
		}
	case basactions.ActionType_ACTION_TYPE_BLUR:
		def.Params = &basactions.ActionDefinition_Blur{
			Blur: &basactions.BlurParams{},
		}
	}

	// Add metadata with node info
	def.Metadata = &basactions.ActionMetadata{
		Label: &outcome.NodeID,
	}

	return def
}

// buildActionTelemetry creates ActionTelemetry from the step outcome.
func buildActionTelemetry(outcome contracts.StepOutcome) *basdomain.ActionTelemetry {
	tel := &basdomain.ActionTelemetry{
		Url: outcome.FinalURL,
	}

	// Add screenshot if present
	// Note: TimelineScreenshot is the proto type which has ArtifactId, Url, etc.
	// The contracts.Screenshot has raw Data, MediaType, etc.
	// For real-time streaming, we don't have artifact URLs yet - those come from storage.
	// We include what we can (dimensions, content type).
	if outcome.Screenshot != nil {
		tel.Screenshot = &basdomain.TimelineScreenshot{
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
		tel.CursorTrail = make([]*basbase.Point, 0, len(outcome.CursorTrail))
		for _, pos := range outcome.CursorTrail {
			tel.CursorTrail = append(tel.CursorTrail, convertPointToProto(&pos.Point))
		}
	}

	// Add highlight regions
		if len(outcome.HighlightRegions) > 0 {
			tel.HighlightRegions = make([]*basdomain.HighlightRegion, 0, len(outcome.HighlightRegions))
			for _, r := range outcome.HighlightRegions {
				region := &basdomain.HighlightRegion{
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
		tel.MaskRegions = make([]*basdomain.MaskRegion, 0, len(outcome.MaskRegions))
		for _, r := range outcome.MaskRegions {
			tel.MaskRegions = append(tel.MaskRegions, &basdomain.MaskRegion{
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
		tel.ConsoleLogs = make([]*basdomain.ConsoleLogEntry, 0, len(outcome.ConsoleLogs))
		for _, log := range outcome.ConsoleLogs {
			entry := &basdomain.ConsoleLogEntry{
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
		tel.NetworkEvents = make([]*basdomain.NetworkEvent, 0, len(outcome.Network))
		for _, net := range outcome.Network {
			event := &basdomain.NetworkEvent{
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
func buildEventContext(outcome contracts.StepOutcome, executionID uuid.UUID) *basbase.EventContext {
	ctx := &basbase.EventContext{
		// Set execution_id as the origin (vs session_id for recordings)
		Origin:  &basbase.EventContext_ExecutionId{ExecutionId: executionID.String()},
		Success: &outcome.Success,
	}

	// Add retry status with current attempt
	ctx.RetryStatus = &basbase.RetryStatus{
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
		ctx.Assertion = &basbase.AssertionResult{
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
func mapStepTypeToActionType(stepType string) basactions.ActionType {
	return typeconv.StringToActionType(stepType)
}

// convertBoundingBoxToProto converts a contracts.BoundingBox to proto format.
func convertBoundingBoxToProto(b *contracts.BoundingBox) *basbase.BoundingBox {
	if b == nil {
		return nil
	}
	return &basbase.BoundingBox{
		X:      b.X,
		Y:      b.Y,
		Width:  b.Width,
		Height: b.Height,
	}
}

// convertPointToProto converts a contracts.Point to proto format.
func convertPointToProto(p *contracts.Point) *basbase.Point {
	if p == nil {
		return nil
	}
	return &basbase.Point{
		X: p.X,
		Y: p.Y,
	}
}

// CompiledInstructionToActionDefinition converts a CompiledInstruction to ActionDefinition.
// This is used when we have the original instruction params available.
func CompiledInstructionToActionDefinition(instr contracts.CompiledInstruction) *basactions.ActionDefinition {
	actionType := mapStepTypeToActionType(instr.Type)

	def := &basactions.ActionDefinition{
		Type: actionType,
	}

	// Build params based on action type using instruction params
	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		def.Params = &basactions.ActionDefinition_Navigate{
			Navigate: params.BuildNavigateParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_CLICK:
		def.Params = &basactions.ActionDefinition_Click{
			Click: params.BuildClickParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_INPUT:
		def.Params = &basactions.ActionDefinition_Input{
			Input: params.BuildInputParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_WAIT:
		def.Params = &basactions.ActionDefinition_Wait{
			Wait: params.BuildWaitParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		def.Params = &basactions.ActionDefinition_Assert{
			Assert: params.BuildAssertParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		def.Params = &basactions.ActionDefinition_Scroll{
			Scroll: params.BuildScrollParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_SELECT:
		def.Params = &basactions.ActionDefinition_SelectOption{
			SelectOption: params.BuildSelectParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		def.Params = &basactions.ActionDefinition_Evaluate{
			Evaluate: params.BuildEvaluateParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		def.Params = &basactions.ActionDefinition_Keyboard{
			Keyboard: params.BuildKeyboardParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_HOVER:
		def.Params = &basactions.ActionDefinition_Hover{
			Hover: params.BuildHoverParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		def.Params = &basactions.ActionDefinition_Screenshot{
			Screenshot: params.BuildScreenshotParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		def.Params = &basactions.ActionDefinition_Focus{
			Focus: params.BuildFocusParams(instr.Params),
		}
	case basactions.ActionType_ACTION_TYPE_BLUR:
		def.Params = &basactions.ActionDefinition_Blur{
			Blur: params.BuildBlurParams(instr.Params),
		}
	}

	// Add metadata with node info
	def.Metadata = &basactions.ActionMetadata{
		Label: &instr.NodeID,
	}

	return def
}
