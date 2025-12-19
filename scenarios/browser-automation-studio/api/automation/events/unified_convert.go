// Package events provides conversion utilities between contracts.StepOutcome
// and the unified bastimeline.TimelineEntry proto format.
//
// This enables the execution engine to produce TimelineEntry messages that can be
// streamed to the UI via WebSocket, supporting the shared Record/Execute UX.
package events

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	"github.com/vrooli/browser-automation-studio/internal/typeconv"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// StepOutcomeToTimelineEntry converts a StepOutcome to the unified TimelineEntry format.
func StepOutcomeToTimelineEntry(outcome contracts.StepOutcome, executionID uuid.UUID) *bastimeline.TimelineEntry {
	entryID := fmt.Sprintf("%s-step-%d-attempt-%d", executionID.String(), outcome.StepIndex, outcome.Attempt)
	stepIndex := int32(outcome.StepIndex)

	entry := &bastimeline.TimelineEntry{
		Id:          entryID,
		SequenceNum: int32(outcome.StepIndex),
		StepIndex:   &stepIndex,
		Action:      buildActionDefinition(outcome),
		Telemetry:   buildActionTelemetry(outcome),
		Context:     buildEventContext(outcome, executionID),
	}

	if outcome.NodeID != "" {
		entry.NodeId = &outcome.NodeID
	}
	if !outcome.StartedAt.IsZero() {
		entry.Timestamp = contracts.TimeToTimestamp(outcome.StartedAt)
	}
	if outcome.DurationMs > 0 {
		durationMs := int32(outcome.DurationMs)
		entry.DurationMs = &durationMs
	}

	return entry
}

// buildActionDefinition creates an ActionDefinition from the step outcome.
// Now that execution captures element context, we can include the used selector
// for element-targeting actions, providing recording-quality timeline entries.
func buildActionDefinition(outcome contracts.StepOutcome) *basactions.ActionDefinition {
	actionType := enums.StringToActionType(outcome.StepType)

	def := &basactions.ActionDefinition{
		Type: actionType,
		Metadata: &basactions.ActionMetadata{
			Label: &outcome.NodeID,
		},
	}

	// Add element context to metadata if available (recording-quality telemetry)
	if outcome.SelectorConfidence > 0 {
		def.Metadata.Confidence = &outcome.SelectorConfidence
	}
	if outcome.ElementSnapshot != nil {
		def.Metadata.ElementSnapshot = outcome.ElementSnapshot
	}
	if outcome.ElementBoundingBox != nil {
		def.Metadata.CapturedBoundingBox = outcome.ElementBoundingBox
	}

	// Populate params based on action type
	// Now that we have UsedSelector from element context, we can populate
	// selector params for element-targeting actions
	switch actionType {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		if outcome.FinalURL != "" {
			def.Params = &basactions.ActionDefinition_Navigate{
				Navigate: &basactions.NavigateParams{Url: outcome.FinalURL},
			}
		}
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		if outcome.Assertion != nil {
			def.Params = &basactions.ActionDefinition_Assert{
				Assert: &basactions.AssertParams{
					Selector: outcome.Assertion.Selector,
					Mode:     enums.StringToAssertionMode(outcome.Assertion.Mode),
					Negated:  &outcome.Assertion.Negated,
				},
			}
		}
	case basactions.ActionType_ACTION_TYPE_CLICK:
		if outcome.UsedSelector != "" {
			def.Params = &basactions.ActionDefinition_Click{
				Click: &basactions.ClickParams{Selector: outcome.UsedSelector},
			}
		}
	case basactions.ActionType_ACTION_TYPE_INPUT:
		if outcome.UsedSelector != "" {
			def.Params = &basactions.ActionDefinition_Input{
				Input: &basactions.InputParams{Selector: outcome.UsedSelector},
			}
		}
	case basactions.ActionType_ACTION_TYPE_HOVER:
		if outcome.UsedSelector != "" {
			def.Params = &basactions.ActionDefinition_Hover{
				Hover: &basactions.HoverParams{Selector: outcome.UsedSelector},
			}
		}
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		if outcome.UsedSelector != "" {
			def.Params = &basactions.ActionDefinition_Focus{
				Focus: &basactions.FocusParams{Selector: outcome.UsedSelector},
			}
		}
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		if outcome.UsedSelector != "" {
			def.Params = &basactions.ActionDefinition_Scroll{
				Scroll: &basactions.ScrollParams{Selector: &outcome.UsedSelector},
			}
		}
	case basactions.ActionType_ACTION_TYPE_SELECT:
		if outcome.UsedSelector != "" {
			def.Params = &basactions.ActionDefinition_SelectOption{
				SelectOption: &basactions.SelectParams{Selector: outcome.UsedSelector},
			}
		}
	}

	return def
}

// buildActionTelemetry creates ActionTelemetry from the step outcome.
func buildActionTelemetry(outcome contracts.StepOutcome) *basdomain.ActionTelemetry {
	tel := &basdomain.ActionTelemetry{
		Url: outcome.FinalURL,
	}

	// Screenshot (contracts.Screenshot has []byte Data not in proto, so partial conversion)
	if outcome.Screenshot != nil {
		tel.Screenshot = &basdomain.TimelineScreenshot{
			Width:       int32(outcome.Screenshot.Width),
			Height:      int32(outcome.Screenshot.Height),
			ContentType: outcome.Screenshot.MediaType,
		}
	}

	// DOM snapshot
	if outcome.DOMSnapshot != nil {
		if outcome.DOMSnapshot.Preview != "" {
			tel.DomSnapshotPreview = &outcome.DOMSnapshot.Preview
		}
		if outcome.DOMSnapshot.HTML != "" {
			tel.DomSnapshotHtml = &outcome.DOMSnapshot.HTML
		}
	}

	// Geometry types are already proto aliases - direct assignment
	tel.ElementBoundingBox = outcome.ElementBoundingBox
	tel.ClickPosition = outcome.ClickPosition

	// Cursor trail (needs Point extraction from CursorPosition)
	if len(outcome.CursorTrail) > 0 {
		tel.CursorTrail = make([]*basbase.Point, 0, len(outcome.CursorTrail))
		for i := range outcome.CursorTrail {
			if outcome.CursorTrail[i].Point != nil {
				tel.CursorTrail = append(tel.CursorTrail, outcome.CursorTrail[i].Point)
			}
		}
	}

	// Region types are already proto aliases - direct assignment
	tel.HighlightRegions = outcome.HighlightRegions
	tel.MaskRegions = outcome.MaskRegions

	if outcome.ZoomFactor != 0 {
		tel.ZoomFactor = &outcome.ZoomFactor
	}

	// Console logs (need time.Time → proto Timestamp conversion)
	tel.ConsoleLogs = ConvertConsoleLogs(outcome.ConsoleLogs)

	// Network events (need time.Time → proto Timestamp conversion)
	tel.NetworkEvents = ConvertNetworkEvents(outcome.Network)

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
			Mode:          enums.StringToAssertionMode(outcome.Assertion.Mode),
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

// CompiledInstructionToActionDefinition converts a CompiledInstruction to ActionDefinition.
// Since instructions now contain the Action field directly, this simply returns it.
func CompiledInstructionToActionDefinition(instr contracts.CompiledInstruction) *basactions.ActionDefinition {
	if instr.Action != nil {
		return instr.Action
	}
	// Return an empty action definition for instructions without an action
	return &basactions.ActionDefinition{
		Type: basactions.ActionType_ACTION_TYPE_UNSPECIFIED,
		Metadata: &basactions.ActionMetadata{
			Label: &instr.NodeID,
		},
	}
}
