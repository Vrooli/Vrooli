package identity

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/provenance"
)

type sessionResolverStub struct {
	ref SessionReference
}

func (s sessionResolverStub) ResolveSessionForRun(context.Context, string) (SessionReference, bool, error) {
	return s.ref, true, nil
}

func TestSessionMiddlewareEnrichesSharedProvenance(t *testing.T) {
	var got provenance.Provenance
	handler := SessionMiddleware(sessionResolverStub{ref: SessionReference{SessionID: "sess-1", SessionKind: "operations", Source: "session/sess-1"}})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = provenance.FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(provenance.NewContext(req.Context(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1"}))
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got.SessionID != "sess-1" || got.SessionKind != "operations" || got.Source != "session/sess-1" {
		t.Fatalf("shared provenance = %+v", got)
	}
}

func TestSessionMiddlewareFailsOpenForOperator(t *testing.T) {
	called := false
	handler := SessionMiddleware(sessionResolverStub{})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		called = true
		if got := provenance.FromContext(r.Context()); got.Actor != provenance.ActorOperator {
			t.Fatalf("provenance = %+v", got)
		}
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	if !called {
		t.Fatal("handler was rejected")
	}
}
