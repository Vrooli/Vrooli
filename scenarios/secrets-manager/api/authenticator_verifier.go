package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/identity"
)

var errAuthenticatorTokenInvalid = errors.New("authenticator token is invalid")

const compatibilityAuthenticatorAudience = "scenario-authenticator:default"

type authenticatorIdentity struct {
	Subject  string
	IssuedAt time.Time
}

type authenticatorVerifier struct {
	verifier *authn.JWTVerifier
}

func newAuthenticatorVerifierFromEnv() *authenticatorVerifier {
	issuer := strings.TrimSpace(getEnv("SECRETS_MANAGER_AUTHENTICATOR_ISSUER"))
	if issuer == "" {
		issuer = "scenario-authenticator"
	}
	audience := strings.TrimSpace(getEnv("SECRETS_MANAGER_AUTHENTICATOR_AUDIENCE"))
	if audience == "" {
		audience = strings.TrimSpace(getEnv("VROOLI_AUTH_SCENARIO_AUDIENCE"))
	}
	if audience == "" {
		audience = compatibilityAuthenticatorAudience
	}
	jwksURL := strings.TrimSpace(getEnv("SECRETS_MANAGER_AUTHENTICATOR_JWKS_URL"))
	client := &http.Client{Timeout: 5 * time.Second}
	config := authn.JWTConfig{
		Source: identity.SourceScenarioAuthenticator, Issuer: issuer, Audience: audience,
		JWKSURL: jwksURL, Client: client,
	}
	if jwksURL == "" {
		config.ResolveJWKS = func(ctx context.Context) (string, error) {
			base, err := discovery.ResolveScenarioURLDefault(ctx, issuer)
			if err != nil {
				return "", err
			}
			return strings.TrimRight(base, "/") + "/.well-known/jwks.json", nil
		}
	}
	return &authenticatorVerifier{verifier: authn.NewJWTVerifier(config)}
}

func (v *authenticatorVerifier) Verify(ctx context.Context, raw string) (authenticatorIdentity, error) {
	if v == nil || v.verifier == nil || strings.TrimSpace(raw) == "" {
		return authenticatorIdentity{}, errAuthenticatorTokenInvalid
	}
	principal, err := v.verifier.Verify(ctx, raw)
	if err != nil {
		return authenticatorIdentity{}, errAuthenticatorTokenInvalid
	}
	return authenticatorIdentity{Subject: principal.Subject}, nil
}
