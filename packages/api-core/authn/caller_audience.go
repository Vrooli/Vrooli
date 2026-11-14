package authn

import (
	"context"
	"net/http"

	"github.com/vrooli/api-core/identity"
)

// WithCreateRunCallerAudience admits the existing immediate human adapter hop
// from one explicitly selected relying scenario. Its original issuer, expiry,
// subject and scopes remain mandatory. Other routes keep the destination's
// canonical audience, including grant approval and global management.
func (c Config) WithCreateRunCallerAudience(resource string) Config {
	providers := append([]Provider(nil), c.Providers...)
	for i, provider := range providers {
		native, ok := provider.(*JWTVerifier)
		if !ok || native.Source() != identity.SourceScenarioAuthenticator {
			continue
		}
		cfg := native.cfg
		cfg.AcceptedAudiences = append(append([]string(nil), cfg.AcceptedAudiences...), "scenario-authenticator:"+resource)
		providers[i] = createRunAudienceProvider{ordinary: native, creation: NewJWTVerifier(cfg)}
	}
	c.Providers = providers
	return c
}

type createRunAudienceProvider struct{ ordinary, creation *JWTVerifier }

func (p createRunAudienceProvider) Source() identity.AuthSource { return p.ordinary.Source() }

func (p createRunAudienceProvider) VerifyRequest(ctx context.Context, r *http.Request) (identity.Principal, error) {
	if r != nil && r.Method == http.MethodPost && r.URL.Path == "/api/v1/runs" {
		return p.creation.VerifyRequest(ctx, r)
	}
	return p.ordinary.VerifyRequest(ctx, r)
}
