package wiring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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
	base = strings.TrimRight(base, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/subscriptions?owner=notification-hub", nil)
	if err != nil {
		return err
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("list Hub subscriptions: %s", response.Status)
	}
	var subscriptions []struct {
		ID      int64  `json:"id"`
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
		Pattern string `json:"event_pattern"`
	}
	if err := json.NewDecoder(response.Body).Decode(&subscriptions); err != nil {
		return err
	}
	var id int64
	for _, sub := range subscriptions {
		if sub.Name == "notification-hub-decisions" && sub.Enabled && sub.Pattern == "**.decision_requested.v1" {
			if id != 0 {
				return errors.New("multiple Hub decision subscriptions")
			}
			id = sub.ID
		}
	}
	if id == 0 {
		return errors.New("Hub decision subscription unavailable")
	}
	body, err := json.Marshal(map[string]string{"event_id": eventID})
	if err != nil {
		return err
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/v1/subscriptions/%d/deliveries/reconcile", base, id), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("reconcile Hub decision delivery: %s", response.Status)
	}
	return nil
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
