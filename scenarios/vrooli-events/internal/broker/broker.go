package broker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
)

const (
	subscriberBufSize = 64
	heartbeatInterval = 30 * time.Second
)

// SSEMessage represents a message to send over an SSE connection.
type SSEMessage struct {
	ID    int64  // maps to SSE "id:" field (store autoincrement ID)
	Event string // maps to SSE "event:" field (event type)
	Data  string // maps to SSE "data:" field (JSON payload)
}

// subscriber tracks a single SSE connection.
type subscriber struct {
	ch           chan SSEMessage
	typePat      string
	sourcePat    string
	targetPat    string
	droppedCount atomic.Int64
	cancel       context.CancelFunc
}

// Broker manages SSE subscribers and distributes events.
type Broker struct {
	mu          sync.RWMutex
	subscribers map[*subscriber]struct{}
	store       store.Store
	done        chan struct{}
}

// NewBroker creates a new SSE broker.
func NewBroker(s store.Store) *Broker {
	return &Broker{
		subscribers: make(map[*subscriber]struct{}),
		store:       s,
		done:        make(chan struct{}),
	}
}

// SubscribeOpts holds subscription parameters.
type SubscribeOpts struct {
	EventTypePattern      string
	SourceScenarioPattern string
	TargetScenarioPattern string
}

// Subscribe registers a new subscriber and returns its event channel and a cleanup function.
// The caller should read from the channel until it's closed.
// The returned context is cancelled when the subscriber is removed.
func (b *Broker) Subscribe(ctx context.Context, opts SubscribeOpts) (<-chan SSEMessage, context.Context, func()) {
	subCtx, cancel := context.WithCancel(ctx)

	sub := &subscriber{
		ch:        make(chan SSEMessage, subscriberBufSize),
		typePat:   opts.EventTypePattern,
		sourcePat: opts.SourceScenarioPattern,
		targetPat: opts.TargetScenarioPattern,
		cancel:    cancel,
	}

	b.mu.Lock()
	b.subscribers[sub] = struct{}{}
	b.mu.Unlock()

	cleanup := func() {
		b.mu.Lock()
		delete(b.subscribers, sub)
		b.mu.Unlock()
		cancel() // stops heartbeat goroutine
	}

	// Start heartbeat for this subscriber
	go b.heartbeat(subCtx, sub)

	return sub.ch, subCtx, cleanup
}

// DroppedCount returns the number of events dropped for the subscriber owning the given channel.
// This is used by the heartbeat to report backpressure.
func (b *Broker) DroppedCount(ch <-chan SSEMessage) int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for sub := range b.subscribers {
		if sub.ch == ch {
			return sub.droppedCount.Swap(0)
		}
	}
	return 0
}

// Publish sends an event to all matching subscribers (non-blocking).
func (b *Broker) Publish(e store.Event, sseData string) {
	msg := SSEMessage{
		ID:    e.ID,
		Event: e.EventType,
		Data:  sseData,
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	for sub := range b.subscribers {
		if !b.matches(sub, e) {
			continue
		}
		select {
		case sub.ch <- msg:
		default:
			// Channel full — drop and track
			sub.droppedCount.Add(1)
		}
	}
}

// Close shuts down the broker and all subscribers.
func (b *Broker) Close() {
	close(b.done)
	b.mu.Lock()
	for sub := range b.subscribers {
		sub.cancel()
		close(sub.ch)
	}
	b.subscribers = make(map[*subscriber]struct{})
	b.mu.Unlock()
}

// SubscriberCount returns the current number of active subscribers.
func (b *Broker) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

func (b *Broker) matches(sub *subscriber, e store.Event) bool {
	if !Match(sub.typePat, e.EventType) {
		return false
	}
	if !Match(sub.sourcePat, e.SourceScenario) {
		return false
	}
	if !Match(sub.targetPat, e.TargetScenario) {
		return false
	}
	return true
}

func (b *Broker) heartbeat(ctx context.Context, sub *subscriber) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-b.done:
			return
		case <-ticker.C:
			dropped := sub.droppedCount.Swap(0)
			msg := SSEMessage{Event: "heartbeat"}
			if dropped > 0 {
				msg.Data = fmt.Sprintf("dropped_count=%d", dropped)
			}
			select {
			case sub.ch <- msg:
			default:
				// If we can't even send a heartbeat, subscriber is hopelessly behind
			}
		}
	}
}
