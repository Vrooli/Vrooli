package approval

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
)

const (
	DecisionAnswerSubscriptionName = "treasury-approval-decision-answers"
	DecisionAnswerEventPattern     = "notification_hub.ask.resolved.v1"
)

// ReconcileDecisionEventSubscription ensures Events has one Treasury-owned
// delivery route for Notification Hub answers. Events owns the durable row;
// this owner keeps the exact target and event pattern current across restarts.
func ReconcileDecisionEventSubscription(ctx context.Context, eventsBase, target string) error {
	base := strings.TrimRight(strings.TrimSpace(eventsBase), "/")
	if base == "" {
		resolved, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
		if err != nil {
			return err
		}
		base = strings.TrimRight(resolved, "/")
	}
	if base == "" || strings.TrimSpace(target) == "" {
		return fmt.Errorf("Vrooli Events base URL and Treasury answer webhook target are required")
	}
	client, err := eventbus.NewSubscriptionClient(base, nil)
	if err != nil {
		return err
	}
	payload := eventbus.Subscription{
		Name: DecisionAnswerSubscriptionName, OwnerScenario: "treasury",
		EventPattern: DecisionAnswerEventPattern, DeliveryType: "webhook",
		DeliveryTarget: strings.TrimSpace(target), Enabled: true,
	}
	if err := client.EnsureNamedSubscription(ctx, payload); err != nil {
		return fmt.Errorf("reconcile Treasury answer subscription: %w", err)
	}
	return nil
}
