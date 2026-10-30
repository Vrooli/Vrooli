package provenance

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

func TestCLIResolvedOperationPairSurvivesDefaultForwardingTransport(t *testing.T) {
	const baseline = "broker-baseline"
	const operation = "broker-operation"
	startCredentialBroker(t, baseline, operation)

	var gotIdentity, gotOperation string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIdentity = r.Header.Get(cliutil.HeaderAgentIdentityToken)
		gotOperation = r.Header.Get(cliutil.HeaderAgentOperationToken)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer api.Close()

	client := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{Override: api.URL}})
	client.SetInvocationHeaderSource(cliutil.InvocationHeaders("agent-manager", "run list"))
	if _, err := client.Do(http.MethodGet, "/api/v1/runs", nil, nil); err != nil {
		t.Fatalf("CLI API request failed: %v", err)
	}
	if gotIdentity != baseline || gotOperation != operation {
		t.Fatalf("composed default transports changed the broker pair (baseline_match=%t operation_match=%t)", gotIdentity == baseline, gotOperation == operation)
	}
}

func TestVerifiedInboundContextTakesPrecedenceOverCLIBrokerPair(t *testing.T) {
	const baseline = "broker-baseline"
	const operation = "broker-operation"
	startCredentialBroker(t, baseline, operation)

	var gotIdentity, gotOperation string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotIdentity = r.Header.Get(cliutil.HeaderAgentIdentityToken)
		gotOperation = r.Header.Get(cliutil.HeaderAgentOperationToken)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer api.Close()

	ctx := context.WithValue(context.Background(), contextKey{}, requestIdentity{
		provenance: Provenance{
			Actor: ActorAgent, VerificationStatus: VerificationVerified, RunID: "verified-run",
			Grant:          &identity.OperationGrant{ID: "verified-grant", Approver: "owner"},
			OperationGrant: &identity.OperationGrant{ID: "verified-grant", Approver: "owner"},
		},
		token:          "verified-inbound-baseline",
		operationToken: "verified-inbound-operation",
	})
	client := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{BaseOptions: cliutil.APIBaseOptions{Override: api.URL}})
	client.SetInvocationHeaderSource(cliutil.InvocationHeaders("agent-manager", "run list"))
	if _, err := client.DoWithContext(ctx, http.MethodGet, "/api/v1/runs", nil, nil); err != nil {
		t.Fatalf("CLI API request failed: %v", err)
	}
	if gotIdentity != "verified-inbound-baseline" || gotOperation != "verified-inbound-operation" {
		t.Fatalf("broker pair overrode verified inbound context (baseline_match=%t operation_match=%t)", gotIdentity == "verified-inbound-baseline", gotOperation == "verified-inbound-operation")
	}
}

func startCredentialBroker(t *testing.T, baseline, operation string) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("/tmp", "api-cli-cred-")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", tmpDir)
	t.Setenv(cliutil.EnvIdentityToken, baseline)
	socket := cliutil.AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(cliutil.HeaderAgentIdentityToken) != baseline {
			t.Error("CLI credential broker request lost the inherited baseline")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"baseline_token": baseline, "operation_token": operation})
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		_ = os.RemoveAll(tmpDir)
	})
}

func TestWriteFieldsSeparatesInteractiveAgentFromItsOwner(t *testing.T) {
	p := Provenance{Actor: ActorAgent, VerificationStatus: VerificationVerified, RunID: "approved-run", Subject: "human-owner", ProfileKey: "codex", Grant: &identity.OperationGrant{Approver: "human-owner"}}
	actor, kind, _, verification, run, _ := p.WriteFields()
	if actor != "approved-run" || kind != ActorAgent || verification != VerificationVerified || run != "approved-run" {
		t.Fatalf("interactive actor attribution = %q/%q/%q/%q", actor, kind, verification, run)
	}
	p.Grant = nil
	actor, _, _, _, _, _ = p.WriteFields()
	if actor != "human-owner" {
		t.Fatalf("legacy subject attribution changed: %q", actor)
	}
}

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

func TestWorkflowCoordinatesUseCanonicalVerifiedIssuerMetadata(t *testing.T) {
	for _, tc := range []struct {
		name  string
		meta  map[string]string
		valid bool
		want  string
	}{
		{"canonical", map[string]string{"workflowExecutionId": "execution-a", "workflowNodeId": "draft", "workflowAttemptId": "63c110db-d5c3-46f1-94ca-37e154535d64"}, true, "execution-a"},
		{"observation keys are not issuer proof", map[string]string{"workflow_execution_id": "execution-a", "workflow_node_id": "draft"}, true, ""},
		{"invalid token cannot project", map[string]string{"workflowExecutionId": "execution-a", "workflowNodeId": "draft"}, false, ""},
		{"unsigned headers cannot project", nil, true, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got Provenance
			handler := Middleware(VerifierFunc(func(string) (*cliutil.VerifyResult, error) {
				return &cliutil.VerifyResult{Valid: tc.valid, Claims: &cliutil.VerifiedClaims{RunID: "run-a", Scopes: []string{"git-control-tower:read"}, Meta: tc.meta}}, nil
			}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { got = FromContext(r.Context()) }))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(cliutil.HeaderAgentIdentityToken, "fixture-credential")
			req.Header.Set("X-Vrooli-Workflow-Execution-Id", "execution-a")
			handler.ServeHTTP(httptest.NewRecorder(), req)
			if got.WorkflowExecutionID != tc.want || got.Attempt != 0 {
				t.Fatalf("workflow projection: execution=%q numericAttempt=%d", got.WorkflowExecutionID, got.Attempt)
			}
		})
	}
}

func TestVerifyOwnWebConsoleSessionUsesVerifiedInboundToken(t *testing.T) {
	var gotToken string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token   string `json:"token"`
			Session bool   `json:"verify_session_binding"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		gotToken = body.Token
		if !body.Session {
			t.Error("missing session proof request")
		}
		_ = json.NewEncoder(w).Encode(cliutil.VerifyResult{Valid: true, Claims: &cliutil.VerifiedClaims{RunID: "run-a", Meta: map[string]string{"verified_web_console_session_id": "wc-a"}}})
	}))
	defer api.Close()
	t.Setenv("AGENT_MANAGER_API_BASE", api.URL)
	t.Setenv(cliutil.EnvIdentityToken, "ambient-token")
	ctx := context.WithValue(context.Background(), contextKey{}, requestIdentity{provenance: Provenance{Actor: ActorAgent, VerificationStatus: VerificationVerified, RunID: "run-a"}, token: "verified-token"})
	id, err := VerifyOwnWebConsoleSession(ctx)
	if err != nil || id != "wc-a" || gotToken != "verified-token" {
		t.Fatalf("session %q token-match %t error %v", id, gotToken == "verified-token", err)
	}
}
