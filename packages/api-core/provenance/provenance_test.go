package provenance

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestRequireScopeRejectsUnverifiedAndMissingScopes(t *testing.T) {
	for _, p := range []Provenance{
		{},
		{Actor: ActorAgent, VerificationStatus: VerificationVerified, RunID: "run-1", Scopes: []string{"vrooli-bridge:read"}},
	} {
		if err := p.RequireScope("vrooli-bridge:write"); err == nil {
			t.Fatal("expected scope refusal")
		} else {
			var denied ScopeDeniedError
			if !errors.As(err, &denied) {
				t.Fatalf("error = %T, want ScopeDeniedError", err)
			}
		}
	}
}

func TestRequireScopeAcceptsVerifiedWildcard(t *testing.T) {
	p := Provenance{Actor: ActorAgent, VerificationStatus: VerificationVerified, RunID: "run-1", Scopes: []string{"vrooli-bridge:*"}}
	if err := p.RequireScope("vrooli-bridge:dispatch"); err != nil {
		t.Fatal(err)
	}
}

// TestMiddlewareDoesNotVerifyTheVerificationEndpoint pins the invariant that
// broke production: cli-core's forwarding transport re-attaches the ambient
// identity token to the middleware's own verification call, so verifying the
// verification endpoint re-enters it without a base case. One inbound request
// must cause zero verifier calls when it targets that path.
func TestMiddlewareDoesNotVerifyTheVerificationEndpoint(t *testing.T) {
	var calls int
	verifier := VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		calls++
		return &cliutil.VerifyResult{Valid: false}, nil
	})

	var got Provenance
	handler := Middleware(verifier)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodPost, VerifyPath, nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "a-token")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if calls != 0 {
		t.Fatalf("verification endpoint must not be verified by the middleware; verifier called %d times", calls)
	}
	if got.VerificationStatus != VerificationAbsent {
		t.Fatalf("verification status = %q, want %q", got.VerificationStatus, VerificationAbsent)
	}
	if got.IsVerifiedAgent() {
		t.Fatal("an unverified request must never be reported as a verified agent")
	}
}

// TestMiddlewareStillVerifiesOtherPaths guards the fix from over-reaching: the
// exemption is for the verification endpoint alone, not a general bypass.
func TestMiddlewareStillVerifiesOtherPaths(t *testing.T) {
	var calls int
	verifier := VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		calls++
		return &cliutil.VerifyResult{Valid: false}, nil
	})
	handler := Middleware(verifier)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/runs", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "a-token")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if calls != 1 {
		t.Fatalf("non-verification paths must still be verified; verifier called %d times, want 1", calls)
	}
}
