package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/vrooli/api-core/identity"
)

func TestWebConsoleUIAvailabilityPreservesVerifiedServiceAuthority(t *testing.T) {
	for _, subject := range []string{"agent-manager", "other-service"} {
		t.Run(subject, func(t *testing.T) {
			want := identity.Principal{Kind: identity.ActorService, Subject: subject, Verified: true, Source: identity.SourceScenarioAuthenticator, Scopes: []string{"web-console:read"}}
			var got identity.Principal
			handler := webConsoleUIAvailabilityFallback(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got, _ = identity.PrincipalFromContext(r.Context())
			}))
			req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(identity.WithPrincipal(context.Background(), want))
			handler.ServeHTTP(httptest.NewRecorder(), req)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("service authority changed: got %+v want %+v", got, want)
			}
		})
	}
}

func TestWebConsoleUIAvailabilityRetainsBrowserFallback(t *testing.T) {
	var got identity.Principal
	handler := webConsoleUIAvailabilityFallback(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, _ = identity.PrincipalFromContext(r.Context())
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !got.IsHuman() || got.Source != identity.SourceUIFailOpen {
		t.Fatalf("browser fallback unavailable: %+v", got)
	}
}

func TestWebConsoleUIAvailabilityDoesNotElevateVerifiedAgent(t *testing.T) {
	want := identity.Principal{Kind: identity.ActorAgent, Subject: "bounded-run", Verified: true, Source: identity.SourceAgentProvenance, Scopes: []string{"web-console:read"}}
	var got identity.Principal
	handler := webConsoleUIAvailabilityFallback(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, _ = identity.PrincipalFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(identity.WithPrincipal(context.Background(), want))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("bounded agent authority changed: got %+v want %+v", got, want)
	}
}

func TestWebConsoleUIAvailabilityHeadersDoNotCreateServiceAuthority(t *testing.T) {
	var got identity.Principal
	handler := webConsoleUIAvailabilityFallback(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, _ = identity.PrincipalFromContext(r.Context())
	}))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Authorization", "Bearer forged-service")
	req.Header.Set("X-Human-Owner-Subject", "agent-manager")
	req.Header.Set("X-Agent-Identity-Token", "forged-agent")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if got.Kind == identity.ActorService || got.Source != identity.SourceUIFailOpen {
		t.Fatalf("unverified headers supplied service authority: %+v", got)
	}
}
