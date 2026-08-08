package provenance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestForwardingTransportCopiesOnlyVerifiedIdentity(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() *http.Request
		want string
	}{
		{
			name: "verified",
			ctx: func() *http.Request {
				request := httptest.NewRequest(http.MethodPost, "http://example.test/", nil)
				return request.WithContext(NewContext(request.Context(), Provenance{Actor: ActorAgent, VerificationStatus: VerificationVerified, RunID: "run-1"}))
			},
			want: "verified-token",
		},
		{
			name: "unbacked provenance",
			ctx: func() *http.Request {
				request := httptest.NewRequest(http.MethodPost, "http://example.test/", nil)
				return request.WithContext(NewContext(request.Context(), Provenance{Actor: ActorAgent, VerificationStatus: VerificationVerified, RunID: "run-1"}))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			transport := ForwardingTransport{Base: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
				got = request.Header.Get(cliutil.HeaderAgentIdentityToken)
				return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: http.NoBody, Request: request}, nil
			})}
			request := tt.ctx()
			if tt.want != "" {
				request = request.WithContext(contextWithForwardedToken(request.Context(), tt.want))
			}
			if _, err := transport.RoundTrip(request); err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("forwarded token = %q, want %q", got, tt.want)
			}
		})
	}
}

func contextWithForwardedToken(ctx context.Context, token string) context.Context {
	identity, _ := ctx.Value(contextKey{}).(requestIdentity)
	identity.token = token
	return context.WithValue(ctx, contextKey{}, identity)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestMiddlewareUsesOnlyVerifiedClaimsForAgentProvenance(t *testing.T) {
	verifier := VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		if token != "valid" {
			return &cliutil.VerifyResult{Valid: false}, nil
		}
		return &cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run-1", TaskID: "task-1", ProfileKey: "codex", ScopePath: "root/task"}}, nil
	})
	var got Provenance
	var forwarded string
	handler := Middleware(verifier)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
		forwarded = ForwardedIdentityToken(r.Context())
	}))

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(cliutil.HeaderAgentIdentityToken, "valid")
	req.Header.Set(cliutil.HeaderInvocationScenario, "plan-manager")
	req.Header.Set(cliutil.HeaderInvocationCommand, "plans create")
	req.Header.Set(cliutil.HeaderInvocationID, "cli-1")
	req.Header.Set(cliutil.HeaderHarnessSessionID, "claude-session-1")
	req.Header.Set(cliutil.HeaderHarnessKind, "claude-code")
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if !got.IsVerifiedAgent() || got.RunID != "run-1" || got.Invocation.Command != "plans create" || got.Invocation.HarnessSessionID != "claude-session-1" || got.Invocation.HarnessKind != "claude-code" {
		t.Fatalf("provenance = %+v", got)
	}
	if forwarded != "valid" {
		t.Fatalf("forwarded token = %q", forwarded)
	}
}

func TestMiddlewareCapturesHarnessObservationWithoutAttribution(t *testing.T) {
	var got Provenance
	handler := Middleware(CLIUtilVerifier{})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = FromContext(r.Context()) }))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(cliutil.HeaderHarnessSessionID, "codex-thread-1")
	req.Header.Set(cliutil.HeaderHarnessKind, "codex")
	req.Header.Set(cliutil.HeaderInvocationScenario, "swarm-manager")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if got.VerificationStatus != VerificationAbsent || got.IsVerifiedAgent() || got.RunID != "" {
		t.Fatalf("observation provenance = %+v", got)
	}
	if got.Invocation.HarnessSessionID != "codex-thread-1" || got.Invocation.HarnessKind != "codex" {
		t.Fatalf("observation fields = %+v", got.Invocation)
	}
}

func TestMiddlewarePreservesInvalidAndUnavailableStates(t *testing.T) {
	tests := []struct {
		name     string
		verifier Verifier
		want     string
	}{
		{name: "invalid", verifier: VerifierFunc(func(string) (*cliutil.VerifyResult, error) { return &cliutil.VerifyResult{Valid: false}, nil }), want: VerificationInvalid},
		{name: "unavailable", verifier: VerifierFunc(func(string) (*cliutil.VerifyResult, error) { return nil, assertError{} }), want: VerificationUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got Provenance
			handler := Middleware(tt.verifier)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = FromContext(r.Context()) }))
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.Header.Set(cliutil.HeaderAgentIdentityToken, "token")
			handler.ServeHTTP(httptest.NewRecorder(), req)
			if got.VerificationStatus != tt.want || got.IsVerifiedAgent() {
				t.Fatalf("provenance = %+v", got)
			}
		})
	}
}

func TestMiddlewareMarksAbsentWithoutRejecting(t *testing.T) {
	var got Provenance
	handler := Middleware(VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
		t.Fatal("absent token must not invoke verifier")
		return nil, nil
	}))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if rec.Code != http.StatusOK || got.VerificationStatus != VerificationAbsent || got.IsVerifiedAgent() {
		t.Fatalf("absent provenance = %+v, status=%d", got, rec.Code)
	}
}

type assertError struct{}

func (assertError) Error() string { return "unavailable" }
