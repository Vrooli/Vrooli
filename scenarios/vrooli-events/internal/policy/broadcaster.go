package policy

import "sync"

// PolicyEvent represents a change to a policy rule, broadcast to SSE subscribers.
type PolicyEvent struct {
	Action string `json:"action"`         // "created", "updated", "deleted"
	RuleID int64  `json:"rule_id"`        // ID of the affected rule
	Rule   *Rule  `json:"rule,omitempty"` // Full rule for creates/updates; nil for deletes
}

// PolicyBroadcaster fans out policy change events to in-process subscribers
// via buffered channels. Slow consumers that fall behind are silently skipped
// (non-blocking sends) so one stalled subscriber cannot block the broadcaster.
type PolicyBroadcaster struct {
	mu   sync.RWMutex
	subs map[int]chan PolicyEvent
	next int
}

// NewPolicyBroadcaster returns an initialised broadcaster.
func NewPolicyBroadcaster() *PolicyBroadcaster {
	return &PolicyBroadcaster{
		subs: make(map[int]chan PolicyEvent),
	}
}

// Subscribe registers a new subscriber and returns a unique ID plus a
// receive-only channel. The caller must eventually call Unsubscribe with
// the returned ID to release resources.
func (b *PolicyBroadcaster) Subscribe() (int, <-chan PolicyEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	id := b.next
	b.next++
	ch := make(chan PolicyEvent, 64)
	b.subs[id] = ch
	return id, ch
}

// Unsubscribe removes the subscriber identified by id and closes its channel.
func (b *PolicyBroadcaster) Unsubscribe(id int) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.subs[id]; ok {
		close(ch)
		delete(b.subs, id)
	}
}

// Broadcast sends evt to every subscriber using a non-blocking send.
// Subscribers whose buffer is full will miss the event.
func (b *PolicyBroadcaster) Broadcast(evt PolicyEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subs {
		select {
		case ch <- evt:
		default:
			// Subscriber too slow; drop the event for this subscriber.
		}
	}
}
