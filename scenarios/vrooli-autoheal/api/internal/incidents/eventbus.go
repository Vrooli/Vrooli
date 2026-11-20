package incidents

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/eventbus"
)

type EventBusPublisher struct {
	Client  eventbus.Client
	Resolve func(context.Context) (string, error)
}

const eventPublicationTimeout = 10 * time.Second

func NewEventBusPublisher(baseURL string) EventPublisher {
	return EventBusPublisher{Client: eventbus.Client{BaseURL: baseURL}}
}

// NewEventBusPublisherResolver creates a publisher that discovers Vrooli
// Events for each publication. The resolver form survives a peer restart
// without changing the fixed-base constructor used by pinned deployments.
func NewEventBusPublisherResolver(resolve func(context.Context) (string, error)) EventPublisher {
	return EventBusPublisher{Client: eventbus.Client{HTTPClient: &http.Client{Timeout: eventPublicationTimeout}}, Resolve: resolve}
}

// Enabled reports whether Events is configured. Pending lifecycle publications
// must remain queued when the shared event client is intentionally disabled.
func (p EventBusPublisher) Enabled() bool { return p.Resolve != nil || p.Client.Enabled() }

func (p EventBusPublisher) Publish(ctx context.Context, eventType string, payload map[string]any) error {
	client, err := p.client(ctx)
	if err != nil {
		return err
	}
	return client.PublishDomainEvent(ctx, eventbus.DomainEvent{Source: "vrooli-autoheal", EventType: eventType, Payload: payload})
}

// PublishDomainEvent satisfies incidents.DecisionPublisher (and
// operatordecision.Publisher) so a producer that builds a full decision
// DomainEvent — Source/EventType/Occurred/Payload set by operatordecision.
// BuildEvent — publishes it verbatim rather than through the legacy
// eventType+payload path. The deterministic Occurred time is preserved so the
// event id is stable across retries.
func (p EventBusPublisher) PublishDomainEvent(ctx context.Context, event eventbus.DomainEvent) error {
	client, err := p.client(ctx)
	if err != nil {
		return err
	}
	return client.PublishDomainEvent(ctx, event)
}

func (p EventBusPublisher) client(ctx context.Context) (eventbus.Client, error) {
	if p.Resolve == nil {
		return p.Client, nil
	}
	baseURL, err := p.Resolve(ctx)
	if err != nil {
		return eventbus.Client{}, err
	}
	if strings.TrimSpace(baseURL) == "" {
		return eventbus.Client{}, fmt.Errorf("vrooli-events resolver returned an empty URL")
	}
	return eventbus.Client{BaseURL: strings.TrimRight(baseURL, "/"), HTTPClient: p.Client.HTTPClient}, nil
}
