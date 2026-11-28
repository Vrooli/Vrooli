// Package hubsender is the contract between Notification Hub and a scenario
// that sends the operator a notice or a decision ask directly (NotificationsService
// Send and Relay, ConversationsService Ask), as that scenario and not as an
// anonymous host process.
//
// A scenario is a sender through an operator-issued scenario-authenticator
// service principal named "<source>-notification-hub": the Hub reads the source
// off the name, so everything the principal sends is attributed to its own
// scenario and to no other. No request field names a source, so there is nothing
// to spoof; a token that is not a sender principal is refused with the reason,
// never downgraded to an anonymous send. See principalrole for the shared shape;
// the credential and token client work as for askreader.
package hubsender

import (
	"context"
	"net/http"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/principalrole"
	"github.com/vrooli/api-core/servicetoken"
)

const (
	// Audience is the exclusive audience of a sender token. The token carries
	// only Scope.
	Audience = "scenario-authenticator:notification-hub"
	// Scope is the one scope a sender token carries: send notices and asks, and
	// read back the asks it sent.
	Scope = "notification-hub:send"
	// Resource is the resource name scenario-authenticator maps to Audience and
	// the second segment of the credential's address.
	Resource = "notification-hub"
	// Suffix ends the name of every sender principal. treasury-notification-hub
	// sends as treasury.
	Suffix = "-notification-hub"
)

// Role is the sender role.
var Role = principalrole.NewRole(Audience, Scope, Resource, Suffix, "notification sender")

// Principal is the service-principal name that sends as source.
func Principal(source string) string { return Role.Principal(source) }

// SourceOf is the source scenario a sender principal sends as, and false for a
// subject that is not a sender principal name.
func SourceOf(subject string) (string, bool) { return Role.SourceOf(subject) }

// CredentialIdentity is where the credential authority stores the credential
// scenario-authenticator issues for source's sender principal.
func CredentialIdentity(source string) string { return Role.CredentialIdentity(source) }

// CredentialField is the field of that credential entry.
const CredentialField = "credential"

// NewVerifier verifies sender tokens at the Hub.
func NewVerifier() authn.TokenVerifier { return Role.NewVerifier() }

// VerifiedSource is the source a request's bearer token sends as.
func VerifiedSource(ctx context.Context, verifier authn.TokenVerifier, headers http.Header) (string, error) {
	return Role.VerifiedSource(ctx, verifier, headers)
}

// NewTokens builds the token client for the scenario sending as source.
func NewTokens(source string, credential servicetoken.CredentialSource, options ...servicetoken.Option) *servicetoken.Tokens {
	return Role.NewTokens(source, credential, options...)
}
