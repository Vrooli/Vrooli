package setup

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/askreader"
	"github.com/vrooli/api-core/operatordecision"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

const DecisionWebhookPath = "/api/v1/events/notification-hub"

func ResolveWebhookSecret(authority *credentialauthority.Authority) (string, error) {
	identity, err := credentialauthority.ParseIdentity("vrooli/vrooli-events")
	if err != nil {
		return "", err
	}
	secret, err := authority.ResolveOrMint(identity, "agent-manager-webhook-secret", nil, func() (string, error) { return credentialauthority.RandomBase64(32) })
	if err != nil {
		return "", fmt.Errorf("resolve events webhook signing secret: %w", err)
	}
	return strings.TrimSpace(secret), nil
}

func DecisionWebhook(secret string, reader operatordecision.AskReader, repo Repository, additional ...operatordecision.DomainEffect) http.Handler {
	effects := []operatordecision.DomainEffect{ResolutionEffect{Repo: repo}}
	effects = append(effects, additional...)
	return operatordecision.NewWebhook(operatordecision.WebhookConfig{Secret: operatordecision.StaticSecret(secret), Source: "compute-manager", Reader: reader, Effects: effects})
}

func AskReaderCredential(ctx context.Context, resolve func(context.Context, string, string) (string, error)) (string, error) {
	identity, err := credentialauthority.ParseIdentity(askreader.CredentialIdentity("compute-manager"))
	if err != nil {
		return "", err
	}
	return resolve(ctx, string(identity), askreader.CredentialField)
}
