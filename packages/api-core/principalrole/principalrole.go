// Package principalrole is the shape every service-principal role shares: a
// scenario-authenticator principal named "<source><suffix>" whose token carries
// one exclusive audience and one scope, and whose source scenario the verifying
// service reads off the name rather than from anything the caller sends.
//
// A role is a value. askreader ("<source>-ask-reader", reads back the asks a
// source raised as events), hubsender ("<source>-notification-hub", sends
// notices and asks) and eventpublisher ("<source>-event-publisher", publishes
// events to vrooli-events) are roles; each package only names its constants and
// keeps the vocabulary its callers use. Adding a role is a Role literal, not a
// copy of the name rule, the credential address, the verifier or the token client.
package principalrole

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/vrooli/api-core/authn"
	apiidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/scopecatalog"
	"github.com/vrooli/api-core/servicetoken"
)

// ErrRefused wraps every reason a verified token does not hold a role. It is
// the Hub's signal to name the mismatch to the caller instead of answering
// with a bare "unauthenticated".
var ErrRefused = errors.New("service principal does not hold the role")

// Role is one service-principal role at the Hub.
type Role struct {
	// Audience is the exclusive audience of the role's token; the token carries
	// only Scope.
	Audience string
	// Scope is the one scope the role's token carries.
	Scope string
	// Resource is the resource name scenario-authenticator maps to Audience in
	// VROOLI_AUTH_RESOURCE_AUDIENCES, and the second segment of the credential's
	// address in the credential authority.
	Resource string
	// Suffix ends the name of every principal of the role.
	Suffix string
	// Label names the role in operator-facing errors.
	Label string

	name *regexp.Regexp
}

// NewRole builds a role. The name rule is the authenticator's principal name
// rule with the role's suffix.
func NewRole(audience, scope, resource, suffix, label string) Role {
	return Role{
		Audience: audience, Scope: scope, Resource: resource, Suffix: suffix, Label: label,
		name: regexp.MustCompile(`^[a-z][a-z0-9-]*` + regexp.QuoteMeta(suffix) + `$`),
	}
}

// Principal is the principal name that holds the role for source.
func (r Role) Principal(source string) string { return strings.TrimSpace(source) + r.Suffix }

// SourceOf is the source scenario a principal name holds the role for, and
// false for a subject that is not a principal name of this role.
func (r Role) SourceOf(subject string) (string, bool) {
	if r.name == nil || !r.name.MatchString(subject) {
		return "", false
	}
	return strings.TrimSuffix(subject, r.Suffix), true
}

// CredentialIdentity is where the credential authority stores the credential
// scenario-authenticator issues for source's principal of the role.
func (r Role) CredentialIdentity(source string) string {
	return "scenario-authenticator/service-principal/" + r.Principal(source) + "/" + r.Resource
}

// NewVerifier verifies the role's tokens at the Hub: the audience must be the
// token's only audience, and the token must be issued by scenario-authenticator
// for a service. It never accepts a human or agent token.
func (r Role) NewVerifier() authn.TokenVerifier {
	return authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{
		Audience:          r.Audience,
		ExclusiveAudience: true,
		Kind:              apiidentity.ActorService,
	})
}

// Source is the source scenario a verified principal holds the role for. The
// error wraps ErrRefused and names what is missing, so the caller can be told.
func (r Role) Source(p apiidentity.Principal) (string, error) {
	switch {
	case !p.Verified || p.Kind != apiidentity.ActorService:
		return "", fmt.Errorf("%w: a verified service token is required", ErrRefused)
	case p.Audience != r.Audience:
		return "", fmt.Errorf("%w: %q holds audience %q, not %q", ErrRefused, p.Subject, p.Audience, r.Audience)
	case !scopecatalog.MatchCapability(p.Scopes, r.Scope):
		return "", fmt.Errorf("%w: %q lacks scope %q", ErrRefused, p.Subject, r.Scope)
	}
	source, ok := r.SourceOf(p.Subject)
	if !ok {
		return "", fmt.Errorf("%w: %q is not named <source>%s", ErrRefused, p.Subject, r.Suffix)
	}
	return source, nil
}

// VerifiedSource verifies the request's bearer token and returns the source
// scenario it holds the role for. A missing or malformed bearer, a token that
// does not verify, and a verified token without the role are all errors; only
// the last wraps ErrRefused, because only then did the caller present a real
// service identity worth explaining the refusal to. The cookie is never
// considered: a service has no browser.
func (r Role) VerifiedSource(ctx context.Context, verifier authn.TokenVerifier, headers http.Header) (string, error) {
	if verifier == nil {
		return "", errors.New("service identity verifier unavailable")
	}
	token, found := strings.CutPrefix(strings.TrimSpace(headers.Get("Authorization")), "Bearer ")
	token = strings.TrimSpace(token)
	if !found || token == "" || strings.ContainsAny(token, " \t\r\n") {
		return "", errors.New("bearer token required")
	}
	principal, err := verifier.Verify(ctx, token)
	if err != nil {
		return "", err
	}
	return r.Source(principal)
}

// NewTokens builds the token client for the scenario acting as source: it
// presents only a token that is exactly the source's principal, the role's
// audience and its single scope. An empty source builds a client that refuses.
func (r Role) NewTokens(source string, credential servicetoken.CredentialSource, options ...servicetoken.Option) *servicetoken.Tokens {
	identity := &servicetoken.Identity{}
	if source != "" {
		identity = servicetoken.ExpectIdentity(r.Principal(source), r.Audience, r.Scope)
	}
	return servicetoken.New(servicetoken.Config{
		Name:         source + " " + r.Label,
		Credential:   credential,
		Expect:       identity,
		MinRemaining: servicetoken.DefaultRefreshMargin,
	}, options...)
}
