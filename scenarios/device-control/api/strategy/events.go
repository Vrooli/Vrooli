package strategy

import (
	"sync"
	"time"
)

// StateChangeEvent is the rule-engine substrate. It is intentionally local to
// device-control; state telemetry is not sent through vrooli-events.
type StateChangeEvent struct {
	DeviceID    string    `json:"device_id"`
	Transport   string    `json:"transport"`
	Attribute   string    `json:"attribute"`
	OldValue    any       `json:"old_value"`
	NewValue    any       `json:"new_value"`
	ObservedAt  time.Time `json:"observed_at"`
	CausationID string    `json:"causation_id"`
	StateClass  string    `json:"state_class"`
}

// StateChangeSink is the narrow adapter-to-control-plane seam. Implementations
// must not block a device transport on a slow future consumer.
type StateChangeSink interface {
	Publish(StateChangeEvent)
}

type StateSubscription struct {
	Events <-chan StateChangeEvent
	Cancel func()
}

// EventBus is a bounded local fast path. A slow subscriber receives the
// latest useful events only after it catches up; publishing never blocks an
// actuation or an inventory probe.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[chan StateChangeEvent]struct{}
}

func NewEventBus() *EventBus { return &EventBus{subscribers: map[chan StateChangeEvent]struct{}{}} }

func (b *EventBus) Publish(event StateChangeEvent) {
	if b == nil {
		return
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (b *EventBus) Subscribe(buffer int) StateSubscription {
	if b == nil {
		return StateSubscription{Events: make(chan StateChangeEvent), Cancel: func() {}}
	}
	if buffer < 1 {
		buffer = 16
	}
	ch := make(chan StateChangeEvent, buffer)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	var once sync.Once
	return StateSubscription{Events: ch, Cancel: func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subscribers, ch)
			close(ch)
			b.mu.Unlock()
		})
	}}
}
