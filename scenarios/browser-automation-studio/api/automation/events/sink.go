package events

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// Sink accepts normalized event envelopes and handles ordering/backpressure.
type Sink interface {
	Publish(ctx context.Context, event contracts.EventEnvelope) error
	Limits() contracts.EventBufferLimits
	// CloseExecution releases execution-owned resources. Decorators must forward
	// it; asynchronous delivery sinks drain accepted events before final cleanup.
	CloseExecution(executionID uuid.UUID)
}
