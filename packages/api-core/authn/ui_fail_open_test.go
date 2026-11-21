package authn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/scopecatalog"
)

func TestUIAvailabilityFallbackAdmitsSameOriginAfterIdentityFailure(t *testing.T) {
	var got identity.Principal
	handler := UIAvailabilityFallback(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, _ = identity.PrincipalFromContext(r.Context())
	}))
	request := httptest.NewRequest(http.MethodPost, "https://console.example/api/mutate", nil)
	request.Host = "console.example"
	request.Header.Set("Origin", "https://console.example")
	request = request.WithContext(identity.WithStatus(context.Background(), identity.Status{State: identity.StateError, FailureClass: identity.FailureUnavailable}))

	handler.ServeHTTP(httptest.NewRecorder(), request)
	if !got.IsHuman() || got.Source != identity.SourceUIFailOpen || !scopecatalog.Resolve(got.Scopes, "git-control-tower:destructive") {
		t.Fatalf("fallback principal = %+v", got)
	}
}

func TestUIAvailabilityFallbackAdmitsAgentThroughBrowserUIAndKeepsProvenance(t *testing.T) {
	var got identity.Principal
	var proof provenance.Provenance
	var gotCtx context.Context
	handler := UIAvailabilityFallback(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		gotCtx = r.Context()
		got, _ = identity.PrincipalFromContext(r.Context())
		proof = provenance.FromContext(r.Context())
	}))
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Verified: true, Source: identity.SourceAgentProvenance})
	ctx = provenance.NewContext(ctx, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1"})
	request := httptest.NewRequest(http.MethodPost, "http://localhost:16382/api/mutate", nil).WithContext(ctx)
	request.Header.Set("Origin", "http://localhost:21233")

	handler.ServeHTTP(httptest.NewRecorder(), request)
	if !got.IsHuman() || got.Source != identity.SourceUIFailOpen {
		t.Fatalf("fallback principal = %+v", got)
	}
	if !proof.IsVerifiedAgent() {
		t.Fatalf("agent provenance was lost: %+v", proof)
	}
	if _, err := RequireIndependentHuman(gotCtx); err != nil {
		t.Fatalf("UI fallback did not satisfy the configured human UI policy: %v", err)
	}
}

func TestUIAvailabilityFallbackDoesNotDependOnOriginHeaders(t *testing.T) {
	for _, tc := range []struct{ name, origin string }{{name: "missing origin"}, {name: "other origin", origin: "https://attacker.example"}} {
		t.Run(tc.name, func(t *testing.T) {
			var got identity.Principal
			handler := UIAvailabilityFallback(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got, _ = identity.PrincipalFromContext(r.Context())
			}))
			request := httptest.NewRequest(http.MethodPost, "https://console.example/api/mutate", nil)
			request.Host = "console.example"
			if tc.origin != "" {
				request.Header.Set("Origin", tc.origin)
			}
			handler.ServeHTTP(httptest.NewRecorder(), request)
			if !got.IsHuman() || got.Source != identity.SourceUIFailOpen {
				t.Fatalf("fallback principal = %+v", got)
			}
		})
	}
}
