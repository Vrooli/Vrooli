package server

import (
	"fmt"
	"os"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// eventsWebhookIdentity and eventsWebhookField name the credential vrooli-events
// signs its ask-resolution webhook deliveries with. deployment-manager MUST
// resolve the SAME value so operatordecision.ValidSignature matches; the field
// name mirrors agent-manager's so every scenario on the rail shares one secret.
const (
	eventsWebhookIdentity = "vrooli/vrooli-events"
	eventsWebhookField    = "agent-manager-webhook-secret"
)

// resolveEventsWebhookSecret resolves the vrooli-events webhook signing secret.
// It prefers the explicit VROOLI_EVENTS_WEBHOOK_SECRET override for local and
// test wiring; when that is empty it resolves (or mints, once) the shared secret
// through the credential authority — the same identity/field vrooli-events signs
// with — so the resolution webhook verifies with zero environment config. An
// empty return (with nil error) means neither source produced a secret; the
// caller must then NOT mount the unsigned webhook.
func resolveEventsWebhookSecret() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("VROOLI_EVENTS_WEBHOOK_SECRET")); explicit != "" {
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

// shouldMountDecisionWebhook decides whether the signed resolution webhook (and
// its subscription reconcile) may be wired. It is a pure predicate so the mount
// gate is unit-tested directly: a resolver error or an empty secret means the
// endpoint would reject every signed delivery, so it must NOT be mounted.
func shouldMountDecisionWebhook(secret string, resolveErr error) bool {
	return resolveErr == nil && strings.TrimSpace(secret) != ""
}
