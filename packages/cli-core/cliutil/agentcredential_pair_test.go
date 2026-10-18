//go:build linux

package cliutil

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func credentialSocketResponse(t *testing.T, payload any, verifyRequest func(*http.Request)) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "clicred-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "identity.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if verifyRequest != nil {
			verifyRequest(r)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close() })
	return socket
}

func TestResolveAgentCredentialsPreservesPairAndLegacyRoles(t *testing.T) {
	var sawBaseline bool
	socket := credentialSocketResponse(t, map[string]string{"baseline_token": "baseline-proof", "operation_token": "operation-proof"}, func(req *http.Request) {
		sawBaseline = req.Header.Get(HeaderAgentIdentityToken) == "baseline-proof"
	})
	pair, err := resolveAgentCredentials(context.Background(), socket, "baseline-proof")
	if err != nil || !sawBaseline || pair.BaselineToken != "baseline-proof" || pair.OperationToken != "operation-proof" || pair.LegacyToken != "" {
		t.Fatalf("paired credentials were merged or lost: pair=%+v sawBaseline=%v err=%v", pair, sawBaseline, err)
	}

	legacySocket := credentialSocketResponse(t, map[string]string{"token": "legacy-grant-only"}, nil)
	legacy, err := resolveAgentCredentials(context.Background(), legacySocket, "")
	if err != nil || legacy.BaselineToken != "" || legacy.OperationToken != "" || legacy.LegacyToken != "legacy-grant-only" {
		t.Fatalf("legacy grant-only response changed roles: %+v err=%v", legacy, err)
	}

	baselineOnlySocket := credentialSocketResponse(t, map[string]string{"baseline_token": "baseline-proof", "operation_refusal": "revoked"}, nil)
	baselineOnly, err := resolveAgentCredentials(context.Background(), baselineOnlySocket, "baseline-proof")
	if err != nil || baselineOnly.BaselineToken != "baseline-proof" || baselineOnly.OperationToken != "" || baselineOnly.OperationRefusal != "revoked" {
		t.Fatalf("supplement refusal suppressed baseline identity: %+v err=%v", baselineOnly, err)
	}
}

func TestResolveAgentCredentialsRejectsRoleConfusionAndBaselineSubstitution(t *testing.T) {
	for name, payload := range map[string]any{
		"baseline plus legacy":       map[string]string{"baseline_token": "baseline", "token": "grant"},
		"substituted baseline":       map[string]string{"baseline_token": "other-baseline"},
		"operation without baseline": map[string]string{"operation_token": "operation"},
	} {
		t.Run(name, func(t *testing.T) {
			socket := credentialSocketResponse(t, payload, nil)
			if _, err := resolveAgentCredentials(context.Background(), socket, "baseline"); err == nil {
				t.Fatal("ambiguous or substituted credential roles were accepted")
			}
		})
	}
}

func TestResolveAgentCredentialsMissingBrokerKeepsOnlyInheritedBaseline(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing", "identity.sock")
	baseline := credentialRoleTestToken(nil)
	credentials, err := resolveAgentCredentials(context.Background(), missing, baseline)
	if err != nil || credentials.BaselineToken != baseline || credentials.OperationToken != "" || credentials.LegacyToken != "" || credentials.OperationRefusal != "authority_unavailable" {
		t.Fatalf("missing broker changed inherited baseline: %+v err=%v", credentials, err)
	}
	if _, err := os.Stat(filepath.Dir(missing)); !os.IsNotExist(err) {
		t.Fatal("fixture unexpectedly created the broker directory")
	}
	if credentials, err := resolveAgentCredentials(context.Background(), missing, "baseline-proof"); err == nil || credentials.BaselineToken != "" || credentials.LegacyToken != "" {
		t.Fatalf("malformed inherited credential bypassed unavailable authority: %+v err=%v", credentials, err)
	}
}

func TestPinnedAgentCredentialChecksExactPairAndRemoteRequestsStripBoth(t *testing.T) {
	current := AgentCredentials{BaselineToken: "baseline-pinned", OperationToken: "operation-pinned"}
	pin := &agentCredentialPin{token: current.BaselineToken, operationToken: current.OperationToken, resolvePair: func(context.Context) (AgentCredentials, error) { return current, nil }}
	request := httptest.NewRequest(http.MethodPost, "http://localhost/protected", nil)
	request = request.WithContext(context.WithValue(request.Context(), agentCredentialPinKey{}, pin))
	request.Header.Set(HeaderAgentIdentityToken, current.BaselineToken)
	request.Header.Set(HeaderAgentOperationToken, current.OperationToken)
	if err := ApplyAgentIdentity(request); err != nil {
		t.Fatalf("stable credential pair refused: %v", err)
	}
	current.OperationToken = "replaced-operation"
	if err := ApplyAgentIdentity(request); err == nil {
		t.Fatal("changed operation half of a pinned pair was accepted")
	}
	remote := httptest.NewRequest(http.MethodPost, "https://untrusted.invalid/collect", nil)
	remote.Header.Set(HeaderAgentIdentityToken, "baseline-secret")
	remote.Header.Set(HeaderAgentOperationToken, "operation-secret")
	if err := ApplyAgentIdentity(remote); err != nil {
		t.Fatal(err)
	}
	if remote.Header.Get(HeaderAgentIdentityToken) != "" || remote.Header.Get(HeaderAgentOperationToken) != "" {
		t.Fatal("credential pair left the trusted scenario boundary")
	}
}
