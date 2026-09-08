package access

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

const authenticatorScenario = "scenario-authenticator"

const compatibilityAuthenticatorAudience = "scenario-authenticator:default"

// authenticatorAudience is retained for package tests and compatibility
// documentation; runtime configuration can override it through the lifecycle.
const authenticatorAudience = compatibilityAuthenticatorAudience

type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// JWKSValidator is the token-economy adapter around api-core's provider-
// neutral verifier. Token-economy owns scope decisions; it does not own JWT
// parsing, signature checks, issuer checks, audience checks, or JWKS caching.
type JWKSValidator struct {
	verifier *authn.JWTVerifier
	now      func() time.Time
}

func NewJWKSValidator(resolver URLResolver, doer HTTPDoer) *JWKSValidator {
	if doer == nil {
		doer = &http.Client{Timeout: 10 * time.Second}
	}
	validator := &JWKSValidator{now: time.Now}
	audience := strings.TrimSpace(os.Getenv("VROOLI_AUTH_SCENARIO_AUDIENCE"))
	if audience == "" {
		audience = compatibilityAuthenticatorAudience
	}
	validator.verifier = authn.NewJWTVerifier(authn.JWTConfig{
		Source:   identity.SourceScenarioAuthenticator,
		Issuer:   authenticatorScenario,
		Audience: audience,
		Kind:     identity.ActorHuman,
		Doer:     doer,
		Now:      func() time.Time { return validator.now() },
		ResolveJWKS: func(ctx context.Context) (string, error) {
			if resolver == nil {
				return "", ErrUnavailable
			}
			base, err := resolver.ResolveScenarioURLDefault(ctx, authenticatorScenario)
			if err != nil {
				return "", err
			}
			return strings.TrimRight(base, "/") + "/.well-known/jwks.json", nil
		},
	})
	return validator
}

func (v *JWKSValidator) Validate(ctx context.Context, token string) (Identity, error) {
	if v == nil || v.verifier == nil || strings.TrimSpace(token) == "" {
		return Identity{}, ErrUnauthenticated
	}
	principal, err := v.verifier.Verify(ctx, token)
	if err != nil {
		if failure, ok := identity.FailureFromError(err); ok && failure.Class == identity.FailureUnavailable {
			return Identity{}, ErrUnavailable
		}
		return Identity{}, ErrUnauthenticated
	}
	return Identity{
		Subject: principal.Subject,
		Roles:   append([]string(nil), principal.Roles...),
		Scopes:  append([]string(nil), principal.Scopes...),
	}, nil
}
