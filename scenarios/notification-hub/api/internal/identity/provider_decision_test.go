package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apiidentity "github.com/vrooli/api-core/identity"
	"notification-hub/internal/hub"
)

type providerDecisionReader struct{ ask hub.AskRecord }

func (r providerDecisionReader) GetAskByRequester(_ context.Context, source, eventType, requesterRef string) (hub.AskRecord, error) {
	if r.ask.Source != source || r.ask.SourceEventType != eventType || r.ask.RequesterRef != requesterRef {
		return hub.AskRecord{}, hub.ErrNotFound
	}
	return r.ask, nil
}

type providerDecisionVerifier struct{ principal apiidentity.Principal }

func (v providerDecisionVerifier) Verify(context.Context, string) (apiidentity.Principal, error) {
	return v.principal, nil
}

func validProviderDecisionPrincipal() apiidentity.Principal {
	return apiidentity.Principal{Kind: apiidentity.ActorService, Subject: ProviderDecisionSubject, Scopes: []string{ProviderDecisionScope}, Verified: true, Audience: ProviderDecisionAudience}
}

func validProviderDecisionAsk() hub.AskRecord {
	return hub.AskRecord{
		ID: "ask-1", Source: "ai-gateway", SourceEventType: "ai_gateway.provider_decision.decision_requested.v1",
		RequesterRef: "event-1", State: hub.AskPending, CreatedAt: "2026-11-01T00:00:00Z", Deadline: "2026-11-02T00:00:00Z",
		Options:                 []hub.AskOption{{Key: "approve"}, {Key: "reject"}},
		ProviderDecisionContext: &hub.ProviderDecisionContext{SchemaVersion: 1, CanonicalTermsSHA256: strings.Repeat("a", 64), Scope: hub.ProviderDecisionScopeRule},
	}
}

func TestProviderDecisionLookupRequiresDedicatedVerifiedServiceIdentity(t *testing.T) {
	principal := validProviderDecisionPrincipal()
	tests := map[string]func(*apiidentity.Principal){
		"wrong subject":      func(p *apiidentity.Principal) { p.Subject = "other" },
		"wrong audience":     func(p *apiidentity.Principal) { p.Audience = "scenario-authenticator:default" },
		"wrong actor":        func(p *apiidentity.Principal) { p.Kind = apiidentity.ActorHuman },
		"missing capability": func(p *apiidentity.Principal) { p.Scopes = nil },
		"unverified":         func(p *apiidentity.Principal) { p.Verified = false },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			p := principal
			change(&p)
			req := httptest.NewRequest(http.MethodPost, ProviderDecisionRoute, strings.NewReader(`{}`))
			req.Header.Set("Authorization", "Bearer test")
			rec := httptest.NewRecorder()
			ProviderDecisionLookupHandler(providerDecisionReader{validProviderDecisionAsk()}, providerDecisionVerifier{p}).ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestProviderDecisionLookupReturnsOnlyExactImmutableDecision(t *testing.T) {
	input := ProviderDecisionLookupRequest{
		RequesterRef: "event-1", Digest: strings.Repeat("a", 64), Scope: hub.ProviderDecisionScopeRule,
	}
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, ProviderDecisionRoute, strings.NewReader(string(body)))
	req.Header.Set("Authorization", "Bearer test")
	rec := httptest.NewRecorder()
	ProviderDecisionLookupHandler(providerDecisionReader{validProviderDecisionAsk()}, providerDecisionVerifier{validProviderDecisionPrincipal()}).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var got ProviderDecisionLookupResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.AskID != "ask-1" || got.Source != "ai-gateway" || got.EventType != "ai_gateway.provider_decision.decision_requested.v1" || got.Requester != input.RequesterRef || got.EventID != input.RequesterRef || got.Digest != input.Digest || got.Scope != input.Scope || got.State != hub.AskPending {
		t.Fatalf("lookup result = %#v", got)
	}
}

func TestProviderDecisionLookupRejectsMismatchesAndMissingConfiguration(t *testing.T) {
	input := `{"requester_ref":"event-1","canonical_terms_sha256":"` + strings.Repeat("a", 64) + `","scope":"provider_rule"}`
	task := validProviderDecisionAsk()
	task.RequesterRef = "other-event"
	req := httptest.NewRequest(http.MethodPost, ProviderDecisionRoute, strings.NewReader(input))
	req.Header.Set("Authorization", "Bearer test")
	rec := httptest.NewRecorder()
	ProviderDecisionLookupHandler(providerDecisionReader{task}, providerDecisionVerifier{validProviderDecisionPrincipal()}).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("mismatch status=%d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	ProviderDecisionLookupHandler(providerDecisionReader{validProviderDecisionAsk()}, nil).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, ProviderDecisionRoute, strings.NewReader(input)))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured status=%d", rec.Code)
	}
}
