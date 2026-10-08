package integrations

import (
	"context"
	"errors"
	"strings"

	"github.com/vrooli/api-core/eventbus"
)

// ErrEventsUnavailable means the vrooli-events base URL is unknown, so an
// event cannot be published yet. Callers keep the event in their outbox.
var ErrEventsUnavailable = errors.New("vrooli-events base URL is not resolved")

// EventsPublisher publishes domain events to the vrooli-events base URL the
// integration config currently holds. Unlike a bare eventbus.Client, an
// unknown base URL is an error rather than a silent no-op, so the hub's
// ask-resolved outbox retries instead of losing the event.
type EventsPublisher struct {
	BaseURL func() string
}

func (p EventsPublisher) PublishDomainEvent(ctx context.Context, event eventbus.DomainEvent) error {
	base := ""
	if p.BaseURL != nil {
		base = strings.TrimSpace(p.BaseURL())
	}
	if base == "" {
		return ErrEventsUnavailable
	}
	return eventbus.Client{BaseURL: base}.PublishDomainEvent(ctx, event)
}
