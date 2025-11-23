package events

import (
	"context"

	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

// Sink accepts normalized event envelopes and handles ordering/backpressure.
type Sink interface {
	Publish(ctx context.Context, event contracts.EventEnvelope) error
	Limits() contracts.EventBufferLimits
}
