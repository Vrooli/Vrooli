package deletiongrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
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
	base = strings.TrimRight(base, "/")
	client := &http.Client{Timeout: 10 * time.Second}
	const pattern = operatordecision.AskResolvedEventType
	var existing []struct {
		ID             int64  `json:"id"`
		EventPattern   string `json:"event_pattern"`
		DeliveryTarget string `json:"delivery_target"`
		Enabled        bool   `json:"enabled"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/v1/subscriptions?owner="+subscriptionOwner, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("list event subscriptions returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(&existing); err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{
		"name":            subscriptionName,
		"owner_scenario":  subscriptionOwner,
		"event_pattern":   pattern,
		"delivery_type":   "webhook",
		"delivery_target": target,
		"enabled":         true,
	})
	if err != nil {
		return err
	}
	for _, item := range existing {
		if item.Enabled && item.DeliveryTarget == target && item.EventPattern == pattern {
			return nil
		}
		if item.Enabled && item.DeliveryTarget == target {
			return putSubscription(ctx, client, fmt.Sprintf("%s/api/v1/subscriptions/%d", base, item.ID), payload)
		}
	}
	req, err = http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/subscriptions", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("create event subscription returned %s", resp.Status)
	}
	return nil
}

func putSubscription(ctx context.Context, client *http.Client, url string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("update event subscription returned %s", resp.Status)
	}
	return nil
}
