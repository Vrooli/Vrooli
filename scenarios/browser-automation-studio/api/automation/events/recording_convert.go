// Package events provides conversion utilities for recording actions to the
// unified bastimeline.TimelineEntry proto format.
//
// This enables recording telemetry to use the same data structure as execution
// telemetry, supporting the shared Record/Execute UX on the timeline.
//
// See "UNIFIED RECORDING/EXECUTION MODEL" in shared.proto for design rationale.
package events

import (
	"github.com/vrooli/browser-automation-studio/automation/telemetry"
	livecapture "github.com/vrooli/browser-automation-studio/services/live-capture"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
)

// RecordedActionToTimelineEntry converts a RecordedAction from the recording
// controller to the unified TimelineEntry proto format.
//
// This function now delegates to the telemetry package for unified conversion.
func RecordedActionToTimelineEntry(action *livecapture.RecordedAction) *bastimeline.TimelineEntry {
	tel := telemetry.RecordedActionToTelemetry(action)
	return telemetry.TelemetryToTimelineEntry(tel)
}
