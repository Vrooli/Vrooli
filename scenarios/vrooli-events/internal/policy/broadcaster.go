// DOC: docs/internal/TEMPORAL-FLOWS.md
// DOC: docs/guides/managing-policies.md
package policy

import "sync"

// PolicyEvent is the versioned, atomically applicable policy cache payload.
// Snapshot consumers must replace both Rules and ReceiptProjections together;
// incremental Action fields are retained only for decoding historical streams.
type PolicyEvent struct {
	Type               string                  `json:"type"` // "snapshot"
	Version            int64                   `json:"version,omitempty"`
	Rules              []Rule                  `json:"rules,omitempty"`
	ReceiptProjections []ReceiptProjectionRule `json:"receipt_projections,omitempty"`
	Action             string                  `json:"action,omitempty"`  // legacy
	RuleID             int64                   `json:"rule_id,omitempty"` // legacy
	Rule               *Rule                   `json:"rule,omitempty"`    // legacy
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

// BroadcastSnapshot broadcasts a complete, versioned policy generation to all
// subscribers. A partial update is intentionally not representable here.
func (b *PolicyBroadcaster) BroadcastSnapshot(version int64, rules []Rule, projections []ReceiptProjectionRule) {
	b.Broadcast(PolicyEvent{
		Type:               "snapshot",
		Version:            version,
		Rules:              rules,
		ReceiptProjections: projections,
	})
}
