package subscription

import (
	"fmt"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const (
	webhookCredentialIdentity = "vrooli/vrooli-events"
	webhookCredentialField    = "agent-manager-webhook-secret"
)

func resolveWebhookSecret() (string, error) {
	identity, err := credentialauthority.ParseIdentity(webhookCredentialIdentity)
	if err != nil {
		return "", fmt.Errorf("parse webhook credential identity: %w", err)
	}
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", fmt.Errorf("credential authority unavailable: %w", err)
	}
	return authority.Require(identity, webhookCredentialField)
}
