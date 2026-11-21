package deletiongrant

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// WebhookPath is where storage-manager receives notification-hub ask
// resolutions. RequestPath is where an agent asks for a deletion grant.
const (
	WebhookPath = "/api/v1/events/notification-hub"
	RequestPath = "/api/v1/deletion-grants/request"

	// The shared vrooli-events webhook credential. Every scenario on the rail
	// resolves the SAME identity/field so operatordecision.ValidSignature matches
	// what vrooli-events signs with — this is why zero env config is needed.
	eventsWebhookIdentity = "vrooli/vrooli-events"
	eventsWebhookField    = "agent-manager-webhook-secret"

	subscriptionOwner = "storage-manager"
	subscriptionName  = "storage-manager-deletion-answers"
)

// ResolveWebhookSecret resolves the vrooli-events webhook signing secret. It
// prefers the VROOLI_EVENTS_WEBHOOK_SECRET override for local/test wiring;
// otherwise it resolves (or mints, once) the shared secret through the
// credential authority — the SAME identity/field vrooli-events signs with, so
// the resolution webhook verifies with zero environment config, exactly like
// deployment-manager and agent-manager. An empty return (nil error) means
// neither source produced a secret, so the caller must NOT mount the webhook.
func ResolveWebhookSecret(getenv func(string) string) (string, error) {
	if explicit := strings.TrimSpace(getenv("VROOLI_EVENTS_WEBHOOK_SECRET")); explicit != "" {
		return explicit, nil
	}
	identity, err := credentialauthority.ParseIdentity(eventsWebhookIdentity)
	if err != nil {
		return "", fmt.Errorf("parse events webhook credential identity: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("credential authority unavailable: %w", err)
	}
	return authority.ResolveOrMint(identity, eventsWebhookField, nil, func() (string, error) {
		return credentialauthority.RandomBase64(32)
	})
}

// ShouldMountWebhook is a pure predicate: an unsigned (empty-secret) or errored
// webhook would reject every delivery, so it must not be mounted.
func ShouldMountWebhook(secret string, resolveErr error) bool {
	return resolveErr == nil && strings.TrimSpace(secret) != ""
}

// ReconcileSubscription ensures exactly one enabled vrooli-events subscription
// delivers notification-hub ask resolutions to target (storage-manager's
// webhook). Without it an answered deletion decision is never delivered back, so
// the approve→project round-trip is dead on arrival. vrooli-events owns the
// durable row; this only ensures it exists and targets the live webhook.
func ReconcileSubscription(ctx context.Context, target string) error {
	base, err := discovery.ResolveScenarioURLDefault(ctx, "vrooli-events")
	if err != nil {
		return err
	}
	client, err := eventbus.NewSubscriptionClient(base, &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		return err
	}
	return reconcileSubscription(ctx, client, target)
}

func reconcileSubscription(ctx context.Context, client *eventbus.SubscriptionClient, target string) error {
	return client.EnsureNamedSubscription(ctx, eventbus.Subscription{
		Name: subscriptionName, OwnerScenario: subscriptionOwner,
		EventPattern: operatordecision.AskResolvedEventType, DeliveryType: "webhook",
		DeliveryTarget: target, Enabled: true,
	})
}
