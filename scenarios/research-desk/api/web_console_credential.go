package main

import (
	"context"
	"errors"

	"github.com/vrooli/api-core/servicetoken"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
)

// The desk's own Web Console identity: the `research-desk` service principal
// (resource web-console, scopes web-console:read and web-console:write) that
// .vrooli/service-principals.json declares. The operator configures and issues
// it with serviceprincipal.IssueCommand; an agent run cannot, because issuing
// needs a verified human.
const (
	webConsoleCredentialIdentity = "scenario-authenticator/service-principal/research-desk/web-console"
	webConsoleCredentialField    = "credential"
	webConsoleAudience           = "scenario-authenticator:web-console"
)

// newWebConsoleTokens is the desk's Web Console token client over a credential
// source. It presents only a token that is exactly the desk's principal for
// Web Console: subject research-desk, audience web-console, read and write.
func newWebConsoleTokens(credential servicetoken.CredentialSource, options ...servicetoken.Option) *servicetoken.Tokens {
	return servicetoken.New(servicetoken.Config{
		Name:         "research-desk web-console",
		Credential:   credential,
		Expect:       servicetoken.ExpectIdentity("research-desk", webConsoleAudience, "web-console:read", "web-console:write"),
		MinRemaining: servicetoken.DefaultRefreshMargin,
	}, options...)
}

// authorityWebConsoleCredential reads the desk's credential from the credential
// authority entry scenario-authenticator writes on issuance.
func authorityWebConsoleCredential(context.Context) (string, error) {
	authority, err := credentialauthority.Default()
	if err != nil {
		return "", err
	}
	identity, err := credentialauthority.ParseIdentity(webConsoleCredentialIdentity)
	if err != nil {
		return "", err
	}
	return authority.Require(identity, webConsoleCredentialField)
}

// webConsoleAuthorization is the peer credential for Web Console: the desk's
// bearer once the operator has issued the principal. Until then it sends none,
// and Web Console's documented availability policy decides, which `research-desk
// now` and /health say. A credential that exists but cannot be used (revoked,
// rejected, or not exactly the desk's identity) is an error, so the desk does
// not call: it never falls back to the weaker policy from a credential it was
// given, and never retries with a wider one.
func webConsoleAuthorization(tokens *servicetoken.Tokens) func(context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		authorization, err := tokens.Authorization(ctx)
		if errors.Is(err, servicetoken.ErrNotIssued) {
			return "", nil
		}
		return authorization, err
	}
}
