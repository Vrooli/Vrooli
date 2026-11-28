package askreader

import "github.com/vrooli/api-core/servicetoken"

// The ask-reader token client is the shared service-principal client
// (servicetoken) configured with the ask-reader identity: the scenario's
// "<source>-ask-reader" principal, the Hub's exclusive audience, and the single
// ask-read scope.
type (
	// Tokens supplies the bearer token a scenario reads its asks with. It
	// refuses a token whose identity is not exactly the scenario's ask-reader
	// principal.
	Tokens = servicetoken.Tokens
	// CredentialSource reads the scenario's ask-reader credential from the
	// credential authority: credentialauthority.Default().Require of
	// ParseIdentity(CredentialIdentity(source)), CredentialField.
	CredentialSource = servicetoken.CredentialSource
	// Exchanged is what scenario-authenticator returns for the credential.
	Exchanged = servicetoken.Exchanged
	// Exchanger trades a credential for an access token.
	Exchanger = servicetoken.Exchanger
	// Option customizes Tokens.
	Option = servicetoken.Option
)

var (
	// ErrNotIssued means no ask-reader credential could be read: the credential
	// authority holds none (the operator issues it; the scenario cannot). The
	// error names serviceprincipal.IssueCommand.
	ErrNotIssued = servicetoken.ErrNotIssued
	// ErrExchangeFailed means scenario-authenticator did not exchange it.
	ErrExchangeFailed = servicetoken.ErrExchangeFailed
	// ErrNoUsableToken means the exchange returned no token or an expiring one.
	ErrNoUsableToken = servicetoken.ErrNoUsableToken
	// ErrIdentityMismatch means the token is not exactly the scenario's
	// ask-reader identity, so it is never presented to the Hub.
	ErrIdentityMismatch = servicetoken.ErrIdentityMismatch
)

// WithExchanger replaces the exchange at the discovered scenario-authenticator.
var WithExchanger = servicetoken.WithExchanger

// WithClock replaces the clock.
var WithClock = servicetoken.WithClock

// NewTokens builds the token client for the scenario that raised asks as source.
// An empty source builds a client that refuses.
func NewTokens(source string, credential CredentialSource, options ...Option) *Tokens {
	return Role.NewTokens(source, credential, options...)
}
