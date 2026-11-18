package wiring

import (
	"context"
	"errors"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
)

// decisionEventClient exposes whether a publication destination actually exists.
// eventbus.Client intentionally treats a disabled optional bus as a no-op;
// durable operator decisions must instead remain pending until it is available.
type decisionEventClient interface {
	Enabled() bool
	PublishDomainEvent(context.Context, eventbus.DomainEvent) error
}

type rediscoveringDecisionPublisher struct {
	resolve    func(context.Context) decisionEventClient
	resolveURL func(context.Context) (string, error)
}

func (p rediscoveringDecisionPublisher) PublishDomainEvent(ctx context.Context, event eventbus.DomainEvent) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	client := p.resolve(ctx)
	if client == nil || !client.Enabled() {
		return errors.New("Vrooli Events is unavailable for decision publication")
	}
	return client.PublishDomainEvent(ctx, event)
}

// ReconcileDecisionDelivery requests an idempotent queue repair from the
// Events owner, using only the exact retained event identity. The owner checks
// the subscription and loads the original payload; this caller supplies none.
func (p rediscoveringDecisionPublisher) ReconcileDecisionDelivery(ctx context.Context, eventID string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if p.resolveURL == nil {
		return errors.New("Events delivery reconciliation unavailable")
	}
	base, err := p.resolveURL(ctx)
	if err != nil {
		return err
	}
	client, err := eventbus.NewSubscriptionClient(base, nil)
	if err != nil {
		return err
	}
	subscriptions, err := client.ListSubscriptions(ctx, "notification-hub")
	if err != nil {
		return err
	}
	var id int64
	for _, sub := range subscriptions {
		if sub.Name == "notification-hub-decisions" && sub.Enabled && sub.EventPattern == "**.decision_requested.v1" {
			if id != 0 {
				return errors.New("multiple Hub decision subscriptions")
			}
			id = sub.ID
		}
	}
	if id == 0 {
		return errors.New("Hub decision subscription unavailable")
	}
	return client.ReconcileDelivery(ctx, id, eventID)
}

func newDecisionPublisher() rediscoveringDecisionPublisher {
	return rediscoveringDecisionPublisher{resolveURL: func(ctx context.Context) (string, error) {
		return discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
	}, resolve: func(ctx context.Context) decisionEventClient {
		// Resolve per attempt so startup unavailability or a changed owner port
		// cannot permanently disable the durable decision outbox.
		return eventbus.NewDiscoveredClient(ctx)
	}}
}
