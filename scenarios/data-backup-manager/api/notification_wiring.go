package main

import (
	"fmt"
	"os"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	dbmEventsWebhookIdentity = "vrooli/vrooli-events"
	dbmEventsWebhookField    = "agent-manager-webhook-secret"
)

// resolveEventsWebhookSecret reads the same signing secret vrooli-events uses
// for ask-resolution deliveries. An empty result means the decision webhook
// must remain unmounted; it is never acceptable to receive unsigned approvals.
func resolveEventsWebhookSecret() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("VROOLI_EVENTS_WEBHOOK_SECRET")); explicit != "" {
		return explicit, nil
	}
	identity, err := credentialauthority.ParseIdentity(dbmEventsWebhookIdentity)
	if err != nil {
		return "", fmt.Errorf("parse events webhook credential identity: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("credential authority unavailable: %w", err)
	}
	return authority.ResolveOrMint(identity, dbmEventsWebhookField, nil, func() (string, error) {
		return credentialauthority.RandomBase64(32)
	})
}
