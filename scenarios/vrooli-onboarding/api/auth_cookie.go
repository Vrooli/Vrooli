package main

import (
	"context"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// onboardingScenarioCookieProvider preserves api-core's provider verification
// while adding the same-origin cookie issued by the onboarding AuthService.
// Browser cookies are only used when the request has no explicit header.
type onboardingScenarioCookieProvider struct {
	provider authn.Provider
}

func (p onboardingScenarioCookieProvider) Source() identity.AuthSource {
	return p.provider.Source()
}

func (p onboardingScenarioCookieProvider) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if p.provider == nil || req == nil {
		return identity.Principal{}, identity.NewFailure(identity.FailureUnavailable, identity.SourceScenarioAuthenticator)
	}
	if strings.TrimSpace(req.Header.Get("Authorization")) != "" {
		return p.provider.VerifyRequest(ctx, req)
	}
	for _, cookieName := range []string{onboardingAccessTokenCookieName, "vrooli_access_token", "gct_access_token"} {
		cookie, err := req.Cookie(cookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			continue
		}
		cloned := req.Clone(ctx)
		cloned.Header = req.Header.Clone()
		cloned.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cookie.Value))
		return p.provider.VerifyRequest(ctx, cloned)
	}
	return p.provider.VerifyRequest(ctx, req)
}

func onboardingAuthProviders(providers []authn.Provider) []authn.Provider {
	result := make([]authn.Provider, 0, len(providers))
	for _, provider := range providers {
		if provider != nil && provider.Source() == identity.SourceScenarioAuthenticator {
			provider = onboardingScenarioCookieProvider{provider: provider}
		}
		result = append(result, provider)
	}
	return result
}
