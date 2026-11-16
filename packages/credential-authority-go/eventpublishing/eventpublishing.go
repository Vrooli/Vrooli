// Package eventpublishing lets a scenario publish events as itself.
//
// vrooli-events takes an event's source from the envelope, so on its own the
// source proves nothing. A scenario proves its source with an operator-issued
// "<source>-event-publisher" service principal (api-core eventpublisher). The
// credential lives in the credential authority, which api-core cannot read, so a
// scenario's main registers this reader once and every eventbus.Client in the
// process presents the identity of the source it publishes as.
//
// Registering is safe before the operator has issued anything: a source with no
// credential publishes exactly as before and vrooli-events counts it as
// unauthenticated. The operator issues every declared principal with
// `scenario-authenticator service-principals sync --apply`.
package eventpublishing

import (
	"context"

	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/eventpublisher"

	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// Register makes every eventbus.Client in this process present its source's
// event-publisher identity.
func Register() { eventbus.UsePublisherCredentials(Credential) }

// Credential reads source's event-publisher credential from the credential
// authority, at the moment it is needed (see credentialauthority.ServiceCredential:
// one issued while the scenario runs is found without a restart).
func Credential(ctx context.Context, source string) (string, error) {
	return credentialauthority.ServiceCredential(eventpublisher.CredentialIdentity(source), eventpublisher.CredentialField)(ctx)
}
