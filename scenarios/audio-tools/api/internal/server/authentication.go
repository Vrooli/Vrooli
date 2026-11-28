package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

// AuthenticationConfig composes configured shared identity providers and
// ensures Audio Tools accepts its scenario-authenticator audience. The
// middleware is passive: domain handlers decide which operations require a
// verified actor or capability.
func AuthenticationConfig(getenv func(string) string) (authn.Config, error) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	config, err := authn.FromEnvironment(getenv)
	if err != nil {
		return authn.Config{}, err
	}
	config = config.WithScenarioAuthenticator("audio-tools")
	issuer := strings.TrimSpace(getenv("LPBS_CONSUMER_TOKEN_ISSUER"))
	if issuer == "" {
		issuer = "https://vrooli.com"
	}
	consumer := consumeridentity.NewRequestProvider(consumeridentity.NewScenarioCache(issuer, "landing-page-business-suite"))
	return withConsumerIdentity(config, consumer, issuer), nil
}

// withConsumerIdentity combines LPBS browser access tokens and the native
// scenario-authenticator token as one credential family. The shared authn
// chain treats an invalid presented credential as an error, so these two
// mutually-exclusive token issuers must be routed to their own verifier before
// they enter that chain. The unverified issuer is only a dispatch hint; the
// selected provider still verifies signature, expiry, and issuer.
func withConsumerIdentity(config authn.Config, consumer authn.Provider, consumerIssuer string) authn.Config {
	var scenario authn.Provider
	providers := make([]authn.Provider, 0, len(config.Providers))
	for _, provider := range config.Providers {
		if provider != nil && provider.Source() == identity.SourceScenarioAuthenticator {
			scenario = provider
			continue
		}
		providers = append(providers, provider)
	}
	if scenario == nil {
		scenario = authn.NewScenarioAuthenticatorProvider(authn.JWTConfig{Audience: "scenario-authenticator:audio-tools", CookieName: "vrooli_access_token"})
	}
	providers = append(providers, consumerOrScenarioProvider{consumer: consumer, scenario: scenario, consumerIssuer: consumerIssuer})
	config.Providers = providers
	return config
}

type consumerOrScenarioProvider struct {
	consumer       authn.Provider
	scenario       authn.Provider
	consumerIssuer string
}

func (p consumerOrScenarioProvider) Source() identity.AuthSource {
	return identity.SourceScenarioAuthenticator
}

func (p consumerOrScenarioProvider) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if req != nil {
		if token := consumeridentity.RequestAccessToken(req); token != "" && tokenIssuerHint(token) == p.consumerIssuer {
			return p.consumer.VerifyRequest(ctx, req)
		}
	}
	return p.scenario.VerifyRequest(ctx, req)
}

// IdentityMiddleware mirrors the shared API server stack for Audio Tools'
// custom net/http server, which otherwise bypasses api-core's standard
// authentication and provenance wrappers.
func IdentityMiddleware(config authn.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return provenance.Middleware(provenance.CLIUtilVerifier{})(authn.Middleware(config)(next))
	}
}
