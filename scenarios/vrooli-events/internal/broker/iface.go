// DOC: docs/internal/SEAMS.md
// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
package broker

import (
	"context"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

// EventBroker defines the interface for event pub-sub used by HTTP handlers.
// This seam allows handlers to be unit-tested with a mock broker instead of
// requiring the full Broker implementation with goroutines and channels.
type EventBroker interface {
	// Subscribe registers a new subscriber and returns its event channel,
	// a context that's cancelled when the subscriber is removed, and a cleanup function.
	Subscribe(ctx context.Context, opts SubscribeOpts) (<-chan SSEMessage, context.Context, func())

	// Publish sends an event to all matching subscribers (non-blocking).
	Publish(e store.Event, sseData string)

	// SubscriberCount returns the current number of active subscribers.
	SubscriberCount() int

	// DroppedCount returns the number of events dropped for the subscriber owning the given channel.
	DroppedCount(ch <-chan SSEMessage) int64

	// Close shuts down the broker and all subscribers.
	Close()
}

// Compile-time check that Broker implements EventBroker.
var _ EventBroker = (*Broker)(nil)
