// Package askreader is the contract between Notification Hub and a scenario
// that reads back the decisions it raised as events.
//
// The Hub shows an ask only to the owner, to the agent run that sent it, or to
// the service that raised it as an event (see notification-hub SECURITY.md,
// "Who reads an ask"). A scenario is the third kind of reader through an
// operator-issued scenario-authenticator service principal named
// "<source>-ask-reader": the Hub reads the source off the name, so the
// principal can read only the asks whose source is its own scenario. This
// package holds the names both sides agree on (the Hub's verifier and each
// reader's token client use the same constants), and Tokens, the one client
// every reading scenario uses to present that identity.
package askreader

import (
	"context"
	"net/http"

	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/principalrole"
)

const (
	// Audience is the exclusive audience of an ask-reader token. The token
	// carries only Scope.
	Audience = "scenario-authenticator:notification-hub-ask-reader"
	// Scope is the one scope an ask-reader token carries.
	Scope = "notification-hub:ask-read"
	// Resource is the resource name scenario-authenticator maps to Audience in
	// VROOLI_AUTH_RESOURCE_AUDIENCES, and the second segment of the credential's
	// address in the credential authority.
	Resource = "notification-hub-ask-reader"
	// Suffix ends the name of every ask-reader principal. A scenario's other
	// service principals never read asks, and the scenario-authenticator rule
	// that principal names are unique does not collide with them:
	// vrooli-autoheal-ask-reader reads the asks whose source is vrooli-autoheal.
	Suffix = "-ask-reader"
)

// Role is the ask-reader role.
var Role = principalrole.NewRole(Audience, Scope, Resource, Suffix, "ask-reader")

// Principal is the service-principal name that reads the asks raised by source.
func Principal(source string) string { return Role.Principal(source) }

// SourceOf is the source scenario an ask-reader principal reads, and false for
// a subject that is not an ask-reader principal name.
func SourceOf(subject string) (string, bool) { return Role.SourceOf(subject) }

// CredentialIdentity is where the credential authority stores the credential
// scenario-authenticator issues for source's ask-reader principal.
func CredentialIdentity(source string) string { return Role.CredentialIdentity(source) }

// CredentialField is the field of that credential entry.
const CredentialField = "credential"

// NewVerifier verifies ask-reader tokens at the Hub.
func NewVerifier() authn.TokenVerifier { return NewVerifierWithConfig(authn.JWTConfig{}) }

// NewVerifierWithConfig verifies the same exclusive ask-reader service role
// while allowing the runtime to supply its trusted authenticator JWKS source.
// Audience, actor kind, exclusivity and issuer remain owned by this role.
func NewVerifierWithConfig(config authn.JWTConfig) authn.TokenVerifier {
	config.Audience = Audience
	config.ExclusiveAudience = true
	config.Kind = apiidentity.ActorService
	config.Issuer = "scenario-authenticator"
	return authn.NewScenarioAuthenticatorProvider(config)
}

// VerifiedSource is the source whose asks a request's bearer token reads.
func VerifiedSource(ctx context.Context, verifier authn.TokenVerifier, headers http.Header) (string, error) {
	return Role.VerifiedSource(ctx, verifier, headers)
}
