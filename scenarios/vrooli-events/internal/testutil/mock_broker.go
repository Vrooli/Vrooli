// DOC: docs/internal/UNIT_TEST_ARCHITECTURE.md
package testutil

import (
	"context"
	"sync"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/broker"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

// MockBroker implements broker.EventBroker for unit-testing HTTP handlers
// without requiring real goroutines or channels.
type MockBroker struct {
	mu sync.Mutex

	subscriberCount int
	publishedEvents []PublishedEvent

	// Configurable Subscribe behavior
	subscribeCh  chan broker.SSEMessage
	subscribeCtx context.Context
}

// PublishedEvent records a Publish call for assertion.
type PublishedEvent struct {
	Event   store.Event
	SSEData string
}

// Compile-time interface check.
var _ broker.EventBroker = (*MockBroker)(nil)

// NewMockBroker creates a MockBroker with a buffered channel for Subscribe.
func NewMockBroker() *MockBroker {
	return &MockBroker{
		subscribeCh: make(chan broker.SSEMessage, 64),
	}
}

// WithSubscriberCount sets the value returned by SubscriberCount.
func (m *MockBroker) WithSubscriberCount(n int) *MockBroker {
	m.subscriberCount = n
	return m
}

func (m *MockBroker) Subscribe(ctx context.Context, _ broker.SubscribeOpts) (<-chan broker.SSEMessage, context.Context, func()) {
	subCtx, cancel := context.WithCancel(ctx)
	m.subscribeCtx = subCtx
	return m.subscribeCh, subCtx, cancel
}

func (m *MockBroker) Publish(e store.Event, sseData string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedEvents = append(m.publishedEvents, PublishedEvent{Event: e, SSEData: sseData})
}

func (m *MockBroker) SubscriberCount() int {
	return m.subscriberCount
}

func (m *MockBroker) DroppedCount(_ <-chan broker.SSEMessage) int64 {
	return 0
}

func (m *MockBroker) Close() {
}

// PublishedEvents returns a copy of all events passed to Publish.
func (m *MockBroker) PublishedEvents() []PublishedEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]PublishedEvent, len(m.publishedEvents))
	copy(out, m.publishedEvents)
	return out
}

// SendToSubscriber pushes a message into the Subscribe channel (for tests that
// call handleSubscribe and need to simulate incoming events).
func (m *MockBroker) SendToSubscriber(msg broker.SSEMessage) {
	m.subscribeCh <- msg
}
