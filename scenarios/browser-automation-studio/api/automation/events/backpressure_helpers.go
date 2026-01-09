package events

import "github.com/vrooli/browser-automation-studio/automation/contracts"

// isDroppable returns true when the event kind may be discarded under
// backpressure without violating delivery guarantees.
func isDroppable(kind contracts.EventKind) bool {
	return kind == contracts.EventKindStepTelemetry || kind == contracts.EventKindStepHeartbeat
}
