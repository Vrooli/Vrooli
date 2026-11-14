package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

type onboardingCookieProviderStub struct {
	seenAuthorization string
}

func (p *onboardingCookieProviderStub) Source() identity.AuthSource {
	return identity.SourceScenarioAuthenticator
}

func (p *onboardingCookieProviderStub) VerifyRequest(_ context.Context, req *http.Request) (identity.Principal, error) {
	p.seenAuthorization = req.Header.Get("Authorization")
	return identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Source: identity.SourceScenarioAuthenticator}, nil
}

func TestOnboardingScenarioCookieProviderUsesSameOriginCookie(t *testing.T) {
	stub := &onboardingCookieProviderStub{}
	provider := onboardingScenarioCookieProvider{provider: stub}
	req := httptest.NewRequest(http.MethodPost, "/credentials", nil)
	req.AddCookie(&http.Cookie{Name: onboardingAccessTokenCookieName, Value: "signed-token"})

	principal, err := provider.VerifyRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("VerifyRequest() error = %v", err)
	}
	if !principal.IsHuman() {
		t.Fatalf("principal = %#v, want verified human", principal)
	}
	if got, want := stub.seenAuthorization, "Bearer signed-token"; got != want {
		t.Fatalf("Authorization = %q, want %q", got, want)
	}
}

func TestOnboardingAuthProvidersWrapsOnlyScenarioAuthenticator(t *testing.T) {
	scenario := &onboardingCookieProviderStub{}
	other := authn.Provider(fakeOnboardingProvider{source: identity.SourceCloudflareAccess})
	providers := onboardingAuthProviders([]authn.Provider{scenario, other})

	if _, ok := providers[0].(onboardingScenarioCookieProvider); !ok {
		t.Fatalf("scenario provider type = %T, want onboardingScenarioCookieProvider", providers[0])
	}
	if providers[1] != other {
		t.Fatalf("non-scenario provider was replaced")
	}
}

type fakeOnboardingProvider struct{ source identity.AuthSource }

func (p fakeOnboardingProvider) Source() identity.AuthSource { return p.source }
func (p fakeOnboardingProvider) VerifyRequest(context.Context, *http.Request) (identity.Principal, error) {
	return identity.Principal{}, identity.NewFailure(identity.FailureMissing, p.source)
}
