// Package events provides conversion utilities between contracts.StepOutcome
// and the unified bastimeline.TimelineEntry proto format.
//
// This enables the execution engine to produce TimelineEntry messages that can be
// streamed to the UI via WebSocket, supporting the shared Record/Execute UX.
package events

import (
	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/automation/telemetry"
	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// StepOutcomeToTimelineEntry converts a StepOutcome to the unified TimelineEntry format.
//
// This function now delegates to the telemetry package for unified conversion.
func StepOutcomeToTimelineEntry(outcome contracts.StepOutcome, executionID uuid.UUID) *bastimeline.TimelineEntry {
	tel := telemetry.StepOutcomeToTelemetry(outcome, executionID)
	return telemetry.TelemetryToTimelineEntry(tel)
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
