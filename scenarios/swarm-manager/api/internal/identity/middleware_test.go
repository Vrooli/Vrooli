package identity

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

// stubVerifier implements Verifier for tests.
type stubVerifier struct {
	result *cliutil.VerifyResult
	err    error
}

func (s *stubVerifier) Verify(_ string) (*cliutil.VerifyResult, error) {
	return s.result, s.err
}

func TestMiddleware_NoHeader_OperatorProvenance(t *testing.T) {
	var got Provenance
	handler := Middleware(&stubVerifier{})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got.Type != TypeOperator {
		t.Errorf("expected operator, got %q", got.Type)
	}
}

func TestMiddleware_ValidToken_AgentProvenance(t *testing.T) {
	verifier := &stubVerifier{
		result: &cliutil.VerifyResult{
			Valid: true,
			Claims: &cliutil.VerifiedClaims{
				RunID:      "run-123",
				TaskID:     "task-456",
				ProfileKey: "default",
			},
		},
	}

	var got Provenance
	handler := Middleware(verifier)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(headerAgentIdentityToken, "valid-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got.Type != TypeAgent {
		t.Fatalf("expected agent, got %q", got.Type)
	}
	if got.RunID != "run-123" {
		t.Errorf("expected RunID run-123, got %q", got.RunID)
	}
	if got.TaskID != "task-456" {
		t.Errorf("expected TaskID task-456, got %q", got.TaskID)
	}
	if got.ProfileKey != "default" {
		t.Errorf("expected ProfileKey default, got %q", got.ProfileKey)
	}
}

func TestMiddleware_InvalidToken_FallsBackToOperator(t *testing.T) {
	verifier := &stubVerifier{
		result: &cliutil.VerifyResult{
			Valid: false,
			Error: "token expired",
		},
	}

	var got Provenance
	handler := Middleware(verifier)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(headerAgentIdentityToken, "bad-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got.Type != TypeOperator {
		t.Errorf("expected operator fallback, got %q", got.Type)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestMiddleware_VerificationError_FallsBackToOperator(t *testing.T) {
	verifier := &stubVerifier{
		err: fmt.Errorf("network timeout"),
	}

	var got Provenance
	handler := Middleware(verifier)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(headerAgentIdentityToken, "some-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got.Type != TypeOperator {
		t.Errorf("expected operator fallback on error, got %q", got.Type)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestProvenance_FormatStartedBy(t *testing.T) {
	tests := []struct {
		prov Provenance
		want string
	}{
		{Provenance{Type: TypeOperator}, "operator"},
		{Provenance{Type: TypeAgent, ProfileKey: "default", RunID: "run-1"}, "agent:default/run-1"},
	}
	for _, tt := range tests {
		got := tt.prov.FormatStartedBy()
		if got != tt.want {
			t.Errorf("FormatStartedBy(%+v) = %q, want %q", tt.prov, got, tt.want)
		}
	}
}

func TestFromContext_NoProvenance_ReturnsOperator(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	got := FromContext(req.Context())
	if got.Type != TypeOperator {
		t.Errorf("expected operator default, got %q", got.Type)
	}
}
