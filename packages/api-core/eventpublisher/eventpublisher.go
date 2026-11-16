// Package eventpublisher is the contract between vrooli-events and a scenario
// that publishes events as itself.
//
// vrooli-events accepts an event from any caller and takes its source scenario
// from the envelope, so on its own the source proves nothing: a consumer cannot
// tell a decision request or an operator's answer from a forgery. A scenario
// proves its source with an operator-issued scenario-authenticator service
// principal named "<source>-event-publisher": vrooli-events reads the source off
// the name and compares it with the envelope. This package names what both sides
// agree on, in the shape the other Hub-facing roles share (see principalrole).
package eventpublisher

import (
	"context"
	"net/http"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/principalrole"
	"github.com/vrooli/api-core/servicetoken"
)

const (
	// Audience is the exclusive audience of an event-publisher token. The token
	// carries only Scope.
	Audience = "scenario-authenticator:vrooli-events"
	// Scope is the one scope an event-publisher token carries.
	Scope = "vrooli-events:publish"
	// Resource is the resource name scenario-authenticator maps to Audience and the
	// second segment of the credential's address.
	Resource = "vrooli-events"
	// Suffix ends the name of every event-publisher principal.
	// storage-manager-event-publisher publishes as storage-manager.
	Suffix = "-event-publisher"
)

// Role is the event-publisher role.
var Role = principalrole.NewRole(Audience, Scope, Resource, Suffix, "event publisher")

// Principal is the service-principal name that publishes as source.
func Principal(source string) string { return Role.Principal(source) }

// SourceOf is the source scenario an event-publisher principal publishes as, and
// false for a subject that is not an event-publisher principal name.
func SourceOf(subject string) (string, bool) { return Role.SourceOf(subject) }

// CredentialIdentity is where the credential authority stores the credential
// scenario-authenticator issues for source's event-publisher principal.
func CredentialIdentity(source string) string { return Role.CredentialIdentity(source) }

// CredentialField is the field of that credential entry.
const CredentialField = "credential"

// NewVerifier verifies event-publisher tokens at vrooli-events.
func NewVerifier() authn.TokenVerifier { return Role.NewVerifier() }

// VerifiedSource is the source a request's bearer token publishes as.
func VerifiedSource(ctx context.Context, verifier authn.TokenVerifier, headers http.Header) (string, error) {
	return Role.VerifiedSource(ctx, verifier, headers)
}

// NewTokens builds the token client for the scenario publishing as source.
func NewTokens(source string, credential servicetoken.CredentialSource, options ...servicetoken.Option) *servicetoken.Tokens {
	return Role.NewTokens(source, credential, options...)
}
