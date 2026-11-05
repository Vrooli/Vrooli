package routing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProviderDecisionIdentityFailsClosedBeforeHubUse(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name       string
		credential func() (string, error)
		exchange   serviceCredentialExchanger
	}{
		{name: "missing credential", credential: func() (string, error) { return "", nil }},
		{name: "authority unavailable", credential: func() (string, error) { return "", context.DeadlineExceeded }},
		{name: "exchange failure", credential: func() (string, error) { return "opaque-test-credential", nil }, exchange: func(context.Context, string) (ExchangedServiceIdentity, error) {
			return ExchangedServiceIdentity{}, context.DeadlineExceeded
		}},
		{name: "wrong subject", credential: goodTestCredential, exchange: goodTestExchanger(now, func(i *ExchangedServiceIdentity) { i.Subject = "another-service" })},
		{name: "wrong audience", credential: goodTestCredential, exchange: goodTestExchanger(now, func(i *ExchangedServiceIdentity) { i.Audience = "scenario-authenticator:web-console" })},
		{name: "wrong capability", credential: goodTestCredential, exchange: goodTestExchanger(now, func(i *ExchangedServiceIdentity) {
			i.Scopes = []string{"notification-hub:provider-decision", "web-console:write"}
		})},
		{name: "expired token", credential: goodTestCredential, exchange: goodTestExchanger(now, func(i *ExchangedServiceIdentity) { i.ExpiresAt = now.Add(time.Second) })},
	} {
		t.Run(test.name, func(t *testing.T) {
			identity := NewProviderDecisionIdentity(test.credential, test.exchange, func() time.Time { return now })
			if _, err := identity.BearerToken(context.Background()); err == nil {
				t.Fatal("invalid identity unexpectedly produced a bearer token")
			}
		})
	}
}

func TestProviderDecisionHubClientUsesOnlyVerifiedServiceToken(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	identity := NewProviderDecisionIdentity(goodTestCredential, goodTestExchanger(now, nil), func() time.Time { return now })
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer short-lived-test-token" {
			t.Errorf("Hub request did not carry the exchanged service bearer")
		}
		if r.URL.Path == providerDecisionLookupPath {
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("unexpected Hub request content type")
			}
			_ = json.NewEncoder(w).Encode(ProviderDecisionLookup{AskID: "ask-test", Source: "ai-gateway", EventType: "ai_gateway.provider_decision.decision_requested.v1"})
			return
		}
		if r.URL.Path != providerDecisionPublishPath {
			t.Errorf("unexpected Hub route %q", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()
	client := NewProviderDecisionHubClient(identity, server.Client())
	client.baseURL = func(context.Context) (string, error) { return server.URL, nil }
	lookup, err := client.Lookup(context.Background(), "event-test", strings.Repeat("a", 64), "provider_rule")
	if err != nil || lookup.AskID != "ask-test" {
		t.Fatal("Hub lookup did not return its exact response")
	}
	if err := client.Publish(context.Background(), "event-test", json.RawMessage(`{"provider_decision_context":{"schema_version":1}}`)); err != nil {
		t.Fatal("Hub publish request failed")
	}
	if calls != 2 {
		t.Fatalf("expected two authorized Hub calls, got %d", calls)
	}
}

func TestProviderDecisionHubClientDoesNotCallHubOnIdentityMismatch(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	identity := NewProviderDecisionIdentity(goodTestCredential, goodTestExchanger(now, func(i *ExchangedServiceIdentity) { i.Subject = "not-ai-gateway" }), func() time.Time { return now })
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	client := NewProviderDecisionHubClient(identity, server.Client())
	client.baseURL = func(context.Context) (string, error) { return server.URL, nil }
	if _, err := client.Lookup(context.Background(), "event-test", strings.Repeat("a", 64), "provider_rule"); err == nil {
		t.Fatal("mismatched service identity unexpectedly accessed Hub")
	}
	if calls != 0 {
		t.Fatal("Hub was called with an unverified service identity")
	}
}

func TestProviderDecisionHubClientNeverFollowsRedirectsWithBearer(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	identity := NewProviderDecisionIdentity(goodTestCredential, goodTestExchanger(now, nil), func() time.Time { return now })
	var redirectedCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == providerDecisionPublishPath {
			http.Redirect(w, r, "/unexpected-path", http.StatusTemporaryRedirect)
			return
		}
		redirectedCalls++
		if r.Header.Get("Authorization") != "" {
			t.Error("service bearer followed a redirect")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	redirectClient := server.Client()
	redirectClient.CheckRedirect = func(*http.Request, []*http.Request) error { return nil }
	client := NewProviderDecisionHubClient(identity, redirectClient)
	client.baseURL = func(context.Context) (string, error) { return server.URL, nil }
	if err := client.Publish(context.Background(), "event-test", json.RawMessage(`{}`)); err == nil {
		t.Fatal("redirect response was accepted")
	}
	if redirectedCalls != 0 {
		t.Fatalf("redirect destination received %d calls", redirectedCalls)
	}
}

func goodTestCredential() (string, error) { return "opaque-test-credential", nil }

func goodTestExchanger(now time.Time, mutate func(*ExchangedServiceIdentity)) serviceCredentialExchanger {
	return func(_ context.Context, credential string) (ExchangedServiceIdentity, error) {
		if credential != "opaque-test-credential" {
			return ExchangedServiceIdentity{}, context.Canceled
		}
		identity := ExchangedServiceIdentity{
			AccessToken: "short-lived-test-token", ExpiresAt: now.Add(time.Hour), Subject: providerDecisionPrincipal,
			Audience: providerDecisionAudience, Scopes: []string{providerDecisionScope},
		}
		if mutate != nil {
			mutate(&identity)
		}
		return identity, nil
	}
}
