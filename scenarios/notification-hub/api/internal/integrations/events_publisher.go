package integrations

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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
	if err := (eventbus.Client{BaseURL: base}).PublishDomainEvent(ctx, event); err != nil {
		return err
	}
	// Decision producers with a durable answer route keep the resolution in
	// the outbox until Events confirms the matching delivery queue row. Other
	// producers retain their own routing.
	source, _ := event.Payload["source"].(string)
	if event.EventType != "notification_hub.ask.resolved.v1" {
		return nil
	}
	switch source {
	case "agent-manager":
		return reconcileDecisionResolution(ctx, base, source, "agent-manager-decision-answers", event)
	case "treasury":
		return reconcileDecisionResolution(ctx, base, source, "treasury-approval-decision-answers", event)
	default:
		return nil
	}
}

func reconcileDecisionResolution(ctx context.Context, base, owner, subscriptionName string, event eventbus.DomainEvent) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	client, err := eventbus.NewSubscriptionClient(base, nil)
	if err != nil {
		return err
	}
	subscriptions, err := client.ListSubscriptions(ctx, owner)
	if err != nil {
		return err
	}
	var id int64
	for _, sub := range subscriptions {
		if sub.Name == subscriptionName && sub.Enabled && sub.EventPattern == "notification_hub.ask.resolved.v1" {
			if id != 0 {
				return fmt.Errorf("multiple %s answer subscriptions", owner)
			}
			id = sub.ID
		}
	}
	if id == 0 {
		return fmt.Errorf("%s answer subscription unavailable", owner)
	}
	// This is the same deterministic identity used by eventbus.Client.
	seed, err := json.Marshal(event)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(seed)
	return client.ReconcileDelivery(ctx, id, hex.EncodeToString(hash[:]))
}
