package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	"program-runtime/internal/programs"
)

func TestAgentBridgeRejectsUnverifiedIdentityAndForwardsVerifiedIdentity(t *testing.T) {
	var received string
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.Header.Get(cliutil.HeaderAgentIdentityToken)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"execution":{"id":"run-1","status":"running"}}`))
	}))
	defer owner.Close()
	delegator := programs.NewHTTPDelegator(owner.URL)
	verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "valid", Claims: &cliutil.VerifiedClaims{RunID: "caller"}}, nil
	})
	bridge := verifiedAgentBridge(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := delegator.Start(r.Context(), programs.DelegationRequest{SessionID: "s", Owner: "owner", WorkflowKey: "workflow"})
		if err != nil {
			t.Errorf("delegation: %v", err)
		}
	}), verifier)
	for _, token := range []string{"valid", "invalid", ""} {
		received = ""
		req := httptest.NewRequest(http.MethodPost, "/internal/program-runtime/agent/start", nil)
		req.Header.Set(cliutil.HeaderAgentIdentityToken, token)
		response := httptest.NewRecorder()
		bridge.ServeHTTP(response, req)
		want := ""
		if token == "valid" {
			want = token
		} else if response.Code != http.StatusUnauthorized {
			t.Errorf("offered %q: status=%d, want 401", token, response.Code)
		}
		if received != want {
			t.Errorf("offered %q: forwarded %q, want %q", token, received, want)
		}
	}
}

func TestTaskBridgeVerifiesAgentProofWithoutInventingOperatorGate(t *testing.T) {
	verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "valid", Claims: &cliutil.VerifiedClaims{RunID: "caller"}}, nil
	})
	var identity provenance.Provenance
	bridge := taskBridgeWithProvenance(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity = provenance.FromContext(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}), verifier)
	for _, test := range []struct {
		token    string
		verified bool
	}{{"valid", true}, {"invalid", false}, {"", false}} {
		request := httptest.NewRequest(http.MethodPost, "/internal/program-runtime/tasks/resolve", nil)
		request.Header.Set(cliutil.HeaderAgentIdentityToken, test.token)
		response := httptest.NewRecorder()
		bridge.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent || identity.IsVerifiedAgent() != test.verified {
			t.Fatalf("token %q: status %d, identity %+v", test.token, response.Code, identity)
		}
	}
}
