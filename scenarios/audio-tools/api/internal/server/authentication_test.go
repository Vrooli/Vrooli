package server

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

type verifiedHumanProvider struct{}

func (verifiedHumanProvider) Source() identity.AuthSource { return identity.SourcePersonalLocal }

func (verifiedHumanProvider) VerifyRequest(context.Context, *http.Request) (identity.Principal, error) {
	return identity.Principal{
		Kind: identity.ActorHuman, Subject: "human-123", Verified: true,
		Source: identity.SourcePersonalLocal, Scopes: []string{"audio-tools:read"},
	}, nil
}

func TestAuthenticationConfigAddsAudioToolsScenarioAuthenticator(t *testing.T) {
	config, err := AuthenticationConfig(func(string) string { return "" })
	require.NoError(t, err)
	require.True(t, config.Enabled())
	require.Len(t, config.Providers, 1)
	require.Equal(t, identity.SourceScenarioAuthenticator, config.Providers[0].Source())
	routed, ok := config.Providers[0].(consumerOrScenarioProvider)
	require.True(t, ok)
	require.Equal(t, "https://vrooli.com", routed.consumerIssuer)
	require.Equal(t, identity.SourceConsumerAccessToken, routed.consumer.Source())
}

type routedIdentityProvider struct {
	source  identity.AuthSource
	subject string
	calls   int
}

func (p *routedIdentityProvider) Source() identity.AuthSource { return p.source }

func (p *routedIdentityProvider) VerifyRequest(context.Context, *http.Request) (identity.Principal, error) {
	p.calls++
	return identity.Principal{Kind: identity.ActorHuman, Subject: p.subject, Verified: true, Source: p.source}, nil
}

func TestConsumerOrScenarioProviderRoutesByUntrustedIssuerHint(t *testing.T) {
	consumer := &routedIdentityProvider{source: identity.SourceConsumerAccessToken, subject: "lpbs-user"}
	scenario := &routedIdentityProvider{source: identity.SourceScenarioAuthenticator, subject: "scenario-user"}
	provider := consumerOrScenarioProvider{consumer: consumer, scenario: scenario, consumerIssuer: "https://vrooli.com"}
	makeToken := func(issuer string) string {
		payload := base64.RawURLEncoding.EncodeToString([]byte(`{"iss":"` + issuer + `"}`))
		return "e30." + payload + ".signature"
	}

	for _, tc := range []struct {
		name       string
		issuer     string
		wantSource identity.AuthSource
		wantCalls  string
	}{
		{name: "LPBS", issuer: "https://vrooli.com", wantSource: identity.SourceConsumerAccessToken, wantCalls: "consumer"},
		{name: "scenario", issuer: "scenario-authenticator", wantSource: identity.SourceScenarioAuthenticator, wantCalls: "scenario"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Authorization", "Bearer "+makeToken(tc.issuer))
			principal, err := provider.VerifyRequest(req.Context(), req)
			require.NoError(t, err)
			require.Equal(t, tc.wantSource, principal.Source)
			if tc.wantCalls == "consumer" {
				require.Equal(t, 1, consumer.calls)
				require.Equal(t, 0, scenario.calls)
			} else {
				require.Equal(t, 1, scenario.calls)
				require.Equal(t, 0, consumer.calls)
			}
			consumer.calls, scenario.calls = 0, 0
		})
	}
}

func TestTokenIssuerHintMalformedTokenIsEmpty(t *testing.T) {
	for _, token := range []string{"", "not.a.jwt", "a.!.b", "a." + base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("x", 20))) + ".c"} {
		require.Empty(t, tokenIssuerHint(token))
	}
}

func TestIdentityMiddlewarePropagatesVerifiedHuman(t *testing.T) {
	config := authn.Config{Providers: []authn.Provider{verifiedHumanProvider{}}}
	var got identity.Principal
	handler := IdentityMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		got, err = authn.RequireHuman(r.Context())
		if err != nil {
			http.Error(w, "human identity missing", http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/identity-check", nil))

	require.Equal(t, http.StatusNoContent, response.Code)
	require.Equal(t, "human-123", got.Subject)
	require.True(t, got.IsHuman())
}
