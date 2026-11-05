package incidents

import (
	"context"

	"github.com/vrooli/api-core/eventbus"
)

type EventBusPublisher struct {
	Client eventbus.Client
}

func NewEventBusPublisher(baseURL string) EventPublisher {
	return EventBusPublisher{Client: eventbus.Client{BaseURL: baseURL}}
}

// Enabled reports whether Events is configured. Pending lifecycle publications
// must remain queued when the shared event client is intentionally disabled.
func (p EventBusPublisher) Enabled() bool { return p.Client.Enabled() }

func (p EventBusPublisher) Publish(ctx context.Context, eventType string, payload map[string]any) error {
	return p.Client.PublishDomainEvent(ctx, eventbus.DomainEvent{Source: "vrooli-autoheal", EventType: eventType, Payload: payload})
}

// PublishDomainEvent satisfies incidents.DecisionPublisher (and
// operatordecision.Publisher) so a producer that builds a full decision
// DomainEvent — Source/EventType/Occurred/Payload set by operatordecision.
// BuildEvent — publishes it verbatim rather than through the legacy
// eventType+payload path. The deterministic Occurred time is preserved so the
// event id is stable across retries.
func (p EventBusPublisher) PublishDomainEvent(ctx context.Context, event eventbus.DomainEvent) error {
	return p.Client.PublishDomainEvent(ctx, event)
}
