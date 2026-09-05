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

func (p EventBusPublisher) Publish(ctx context.Context, eventType string, payload map[string]any) error {
	return p.Client.PublishDomainEvent(ctx, eventbus.DomainEvent{Source: "vrooli-autoheal", EventType: eventType, Payload: payload})
}
