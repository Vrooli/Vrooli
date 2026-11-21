package telemetry

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	basdomain "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/domain"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// BuildExecutionTimelineEntry constructs the streamed execution entry at its source
// boundary, without routing it through the recording telemetry intermediate.
func BuildExecutionTimelineEntry(outcome contracts.StepOutcome, executionID uuid.UUID) *bastimeline.TimelineEntry {
	stepIndex := int32(outcome.StepIndex)
	success := outcome.Success
	entry := &bastimeline.TimelineEntry{
		Id:          fmt.Sprintf("%s-step-%d-attempt-%d", executionID, outcome.StepIndex, outcome.Attempt),
		SequenceNum: stepIndex,
		StepIndex:   &stepIndex,
		Action:      &basactions.ActionDefinition{Type: enums.StringToActionType(outcome.StepType)},
		Telemetry:   &basdomain.ActionTelemetry{Url: outcome.FinalURL, ElementBoundingBox: outcome.ElementBoundingBox, ClickPosition: outcome.ClickPosition, HighlightRegions: outcome.HighlightRegions, MaskRegions: outcome.MaskRegions},
		Context: &basbase.EventContext{
			Origin:      &basbase.EventContext_ExecutionId{ExecutionId: executionID.String()},
			Success:     &success,
			Condition:   contracts.ConditionOutcomeToProto(outcome.Condition),
			RetryStatus: &basbase.RetryStatus{CurrentAttempt: int32(outcome.Attempt), MaxAttempts: 1, Configured: outcome.Attempt > 0},
		},
	}
	if outcome.NodeID != "" {
		nodeID := outcome.NodeID
		entry.NodeId = &nodeID
	}
	if !outcome.StartedAt.IsZero() {
		entry.Timestamp = contracts.TimeToTimestamp(outcome.StartedAt)
	}
	if outcome.DurationMs > 0 {
		duration := int32(outcome.DurationMs)
		entry.DurationMs = &duration
	}
	if outcome.SelectorConfidence > 0 || outcome.ElementSnapshot != nil {
		entry.Action.Metadata = &basactions.ActionMetadata{ElementSnapshot: outcome.ElementSnapshot}
		if outcome.SelectorConfidence > 0 {
			entry.Action.Metadata.Confidence = &outcome.SelectorConfidence
		}
		if !outcome.StartedAt.IsZero() {
			entry.Action.Metadata.CapturedAt = contracts.TimeToTimestamp(outcome.StartedAt)
		}
	}
	if outcome.Screenshot != nil {
		entry.Telemetry.Screenshot = &basdomain.TimelineScreenshot{Width: int32(outcome.Screenshot.Width), Height: int32(outcome.Screenshot.Height), ContentType: outcome.Screenshot.MediaType}
	}
	if len(outcome.CursorTrail) > 0 {
		for _, point := range outcome.CursorTrail {
			if point.Point != nil {
				entry.Telemetry.CursorTrail = append(entry.Telemetry.CursorTrail, point.Point)
			}
		}
	}
	if outcome.Assertion != nil {
		assertion := outcome.Assertion
		entry.Context.Assertion = &basbase.AssertionResult{Mode: enums.StringToAssertionMode(assertion.Mode), Selector: assertion.Selector, Success: assertion.Success, Negated: assertion.Negated, CaseSensitive: assertion.CaseSensitive}
		if assertion.Message != "" {
			entry.Context.Assertion.Message = &assertion.Message
		}
		if assertion.Expected != nil {
			entry.Context.Assertion.Expected = contracts.AnyToJsonValue(assertion.Expected)
		}
		if assertion.Actual != nil {
			entry.Context.Assertion.Actual = contracts.AnyToJsonValue(assertion.Actual)
		}
	}
	if len(outcome.ExtractedData) > 0 {
		entry.Context.ExtractedData = make(map[string]*commonv1.JsonValue, len(outcome.ExtractedData))
		for key, value := range outcome.ExtractedData {
			if converted := contracts.AnyToJsonValue(value); converted != nil {
				entry.Context.ExtractedData[key] = converted
			}
		}
	}
	if outcome.Failure != nil {
		if outcome.Failure.Message != "" {
			entry.Context.Error = &outcome.Failure.Message
		}
		if outcome.Failure.Code != "" {
			entry.Context.ErrorCode = &outcome.Failure.Code
		}
	}
	return entry
}
