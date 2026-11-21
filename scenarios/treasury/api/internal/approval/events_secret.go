package approval

import (
	"fmt"
	"os"
	"strings"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	eventsWebhookIdentity = "vrooli/vrooli-events"
	eventsWebhookField    = "agent-manager-webhook-secret"
)

// ResolveEventsWebhookSecret reads the signing secret used by Vrooli Events.
// It is not an operator approval credential; the approval endpoint still
// requires the source's own ask-reader principal and Hub readback.
func ResolveEventsWebhookSecret() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("VROOLI_EVENTS_WEBHOOK_SECRET")); explicit != "" {
		return explicit, nil
	}
	identity, err := credentialauthority.ParseIdentity(eventsWebhookIdentity)
	if err != nil {
		return "", fmt.Errorf("parse Vrooli Events webhook credential identity: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("credential authority unavailable: %w", err)
	}
	return authority.ResolveOrMint(identity, eventsWebhookField, nil, func() (string, error) {
		return credentialauthority.RandomBase64(32)
	})
}
