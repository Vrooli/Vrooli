package telemetry

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/internal/enums"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
)

// StepOutcomeToTelemetry converts an execution step outcome to the
// unified ActionTelemetry format.
func StepOutcomeToTelemetry(outcome contracts.StepOutcome, executionID uuid.UUID) *ActionTelemetry {
	tel := &ActionTelemetry{
		ID:          fmt.Sprintf("%s-step-%d-attempt-%d", executionID.String(), outcome.StepIndex, outcome.Attempt),
		SequenceNum: outcome.StepIndex,
		StepIndex:   outcome.StepIndex,
		NodeID:      outcome.NodeID,
		ActionType:  enums.StringToActionType(outcome.StepType),
		Label:       outcome.NodeID, // Use node ID as label for execution

		// Timing
		Timestamp:  outcome.StartedAt,
		DurationMs: outcome.DurationMs,

		// Element context
		Selector:           outcome.UsedSelector,
		SelectorConfidence: outcome.SelectorConfidence,
		SelectorMatchCount: outcome.SelectorMatchCount,
		ElementSnapshot:    outcome.ElementSnapshot,
		BoundingBox:        outcome.ElementBoundingBox,

		// Page context
		URL:      outcome.FinalURL,
		FinalURL: outcome.FinalURL,

		// Captured artifacts
		Screenshot:  convertContractsScreenshot(outcome.Screenshot),
		DOMSnapshot: convertContractsDOMSnapshot(outcome.DOMSnapshot),
		ConsoleLogs: convertContractsConsoleLogs(outcome.ConsoleLogs),
		Network:     convertContractsNetworkEvents(outcome.Network),

		// Visual context
		ClickPosition:    outcome.ClickPosition,
		CursorTrail:      extractCursorPoints(outcome.CursorTrail),
		HighlightRegions: outcome.HighlightRegions,
		MaskRegions:      outcome.MaskRegions,
		ZoomFactor:       outcome.ZoomFactor,

		// Result
		Success: outcome.Success,
		Failure: convertContractsFailure(outcome.Failure),
	}

	// Execution-specific origin
	tel.Origin = &ExecutionOrigin{
		ExecutionID:   executionID,
		WorkflowID:    uuid.Nil, // Not available from StepOutcome; could be enhanced
		StepIndex:     outcome.StepIndex,
		Attempt:       outcome.Attempt,
		MaxAttempts:   1, // Default; could be enhanced from instruction config
		ExtractedData: outcome.ExtractedData,
		Assertion:     convertContractsAssertion(outcome.Assertion),
		Condition:     convertContractsCondition(outcome.Condition),
		ProbeResult:   outcome.ProbeResult,
	}

	return tel
}

func convertContractsScreenshot(s *contracts.Screenshot) *Screenshot {
	if s == nil {
		return nil
	}
	return &Screenshot{
		Data:      s.Data,
		MediaType: s.MediaType,
		Width:     s.Width,
		Height:    s.Height,
	}
}

func convertContractsDOMSnapshot(d *contracts.DOMSnapshot) *DOMSnapshot {
	if d == nil {
		return nil
	}
	return &DOMSnapshot{
		HTML:    d.HTML,
		Preview: d.Preview,
	}
}

func convertContractsConsoleLogs(logs []contracts.ConsoleLogEntry) []ConsoleLogEntry {
	if len(logs) == 0 {
		return nil
	}
	result := make([]ConsoleLogEntry, len(logs))
	for i, log := range logs {
		result[i] = ConsoleLogEntry{
			Type:      log.Type,
			Text:      log.Text,
			Timestamp: log.Timestamp,
			Stack:     log.Stack,
			Location:  log.Location,
		}
	}
	return result
}

func convertContractsNetworkEvents(events []contracts.NetworkEvent) []NetworkEvent {
	if len(events) == 0 {
		return nil
	}
	result := make([]NetworkEvent, len(events))
	for i, evt := range events {
		result[i] = NetworkEvent{
			Type:         evt.Type,
			URL:          evt.URL,
			Method:       evt.Method,
			ResourceType: evt.ResourceType,
			Status:       evt.Status,
			OK:           evt.OK,
			Failure:      evt.Failure,
			Timestamp:    evt.Timestamp,
		}
	}
	return result
}

func convertContractsFailure(f *contracts.StepFailure) *FailureInfo {
	if f == nil {
		return nil
	}
	return &FailureInfo{
		Kind:      string(f.Kind),
		Code:      f.Code,
		Message:   f.Message,
		Retryable: f.Retryable,
	}
}

func convertContractsAssertion(a *contracts.AssertionOutcome) *AssertionOutcome {
	if a == nil {
		return nil
	}
	return &AssertionOutcome{
		Mode:          a.Mode,
		Selector:      a.Selector,
		Expected:      a.Expected,
		Actual:        a.Actual,
		Success:       a.Success,
		Negated:       a.Negated,
		CaseSensitive: a.CaseSensitive,
		Message:       a.Message,
	}
}

func convertContractsCondition(c *contracts.ConditionOutcome) *ConditionOutcome {
	if c == nil {
		return nil
	}
	return &ConditionOutcome{
		Type:     c.Type,
		Outcome:  c.Outcome,
		Negated:  c.Negated,
		Operator: c.Operator,
		Variable: c.Variable,
		Selector: c.Selector,
		Actual:   c.Actual,
		Expected: c.Expected,
	}
}

func extractCursorPoints(trail []contracts.CursorPosition) []*basbase.Point {
	if len(trail) == 0 {
		return nil
	}
	points := make([]*basbase.Point, 0, len(trail))
	for _, pos := range trail {
		if pos.Point != nil {
			points = append(points, pos.Point)
		}
	}
	return points
}
