package readinesshandler

import (
	"net/http"

	"github.com/vrooli/api-core/operatordecision"
	"github.com/vrooli/vrooli/packages/credential-authority-go/hubidentity"
)

// DecisionEventWebhookPath is the notification-hub ask.resolved webhook route
// deployment-manager mounts to receive release-decision resolutions. It matches
// agent-manager's route so the operator-decision rail is uniform across
// scenarios.
const DecisionEventWebhookPath = "/api/v1/events/notification-hub"

// DecisionResolvedWebhook receives notification-hub's
// notification_hub.ask.resolved.v1 through the signed vrooli-events webhook and
// applies it via the shared operatordecision.Webhook. The substrate verifies the
// HMAC signature, decodes the envelope, gates on the ask-resolved event type,
// confirms the resolution with the Hub (the release-approval ask must be read back
// as deployment-manager's own ask-reader before a release is approved), and fans
// the resolution to the release-decision effect (the only effect
// deployment-manager registers today). Until the operator issues the ask-reader
// credential with `scenario-authenticator service-principals sync --apply`, the
// body is applied as before and each such resolution is counted.
func DecisionResolvedWebhook(secret string, effects ...operatordecision.DomainEffect) http.Handler {
	return operatordecision.NewWebhook(hubidentity.WebhookConfig(operatordecision.WebhookConfig{
		Secret: operatordecision.StaticSecret(secret), Source: ReleaseDecisionSource,
		Effects: effects,
	}))
}
