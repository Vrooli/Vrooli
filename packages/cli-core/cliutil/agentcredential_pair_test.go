//go:build linux

package cliutil

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
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

func TestResolveAgentCredentialsAcceptsOnlySameRunBaselineRenewal(t *testing.T) {
	now := time.Now().Unix()
	meta := map[string]string{"execution_mode": "attached", "harness_kind": "codex", "harness_session": "same"}
	old := renewalClientTestToken(ordinaryIdentityClaims{
		RunID: "run-one", TaskID: "", Scopes: []string{}, ProfileKey: "codex", ScopePath: "",
		IssuedAt: now - 48*60*60, ExpiresAt: now - 24*60*60, Meta: meta,
	})
	current := renewalClientTestToken(ordinaryIdentityClaims{
		RunID: "run-one", TaskID: "", Scopes: []string{}, ProfileKey: "codex", ScopePath: "",
		IssuedAt: now - 60, ExpiresAt: now + 60*60, Meta: meta,
	})
	socket := credentialSocketResponse(t, map[string]string{"baseline_token": current}, func(req *http.Request) {
		if req.Header.Get(HeaderAgentIdentityToken) != old {
			t.Error("broker request did not carry the exact expired inherited token")
		}
	})
	got, err := resolveAgentCredentials(context.Background(), socket, old)
	if err != nil || got.BaselineToken != current || got.LegacyToken != "" || got.OperationToken != "" {
		t.Fatalf("same-run ordinary renewal was not adopted as baseline-only (baseline=%t legacy=%t operation=%t) err=%v", got.BaselineToken != "", got.LegacyToken != "", got.OperationToken != "", err)
	}
}

func TestSameRunBaselineRenewalRejectsScopeAndGenerationChanges(t *testing.T) {
	now := time.Now().Unix()
	meta := map[string]string{"execution_mode": "attached", "harness_kind": "codex", "harness_session": "same"}
	old := ordinaryIdentityClaims{
		RunID: "run-one", Scopes: []string{}, ProfileKey: "codex", Meta: meta,
		IssuedAt: now - 48*60*60, ExpiresAt: now - 24*60*60,
	}
	base := ordinaryIdentityClaims{
		RunID: "run-one", Scopes: []string{}, ProfileKey: "codex", Meta: meta,
		IssuedAt: now - 60, ExpiresAt: now + 60*60,
	}
	cases := map[string]func(*ordinaryIdentityClaims){
		"different run":    func(c *ordinaryIdentityClaims) { c.RunID = "run-two" },
		"different scopes": func(c *ordinaryIdentityClaims) { c.Scopes = []string{"agent-manager:write"} },
		"different harness binding": func(c *ordinaryIdentityClaims) {
			c.Meta = map[string]string{"execution_mode": "attached", "harness_kind": "other", "harness_session": "same"}
		},
		"grant purpose":       func(c *ordinaryIdentityClaims) { c.Meta["authorization_id"] = "grant" },
		"replacement expired": func(c *ordinaryIdentityClaims) { c.ExpiresAt = now - 1 },
		"future issued at":    func(c *ordinaryIdentityClaims) { c.IssuedAt = now + 1 },
		"too long": func(c *ordinaryIdentityClaims) {
			c.ExpiresAt = c.IssuedAt + int64(defaultAgentIdentityTTL/time.Second) + 1
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			replacement := base
			replacement.Meta = make(map[string]string, len(base.Meta))
			for key, value := range base.Meta {
				replacement.Meta[key] = value
			}
			change(&replacement)
			if isSameRunBaselineRenewal(renewalClientTestToken(old), renewalClientTestToken(replacement)) {
				t.Fatal("broker replacement changed an immutable run/authority claim but was accepted")
			}
		})
	}
	validOld := old
	validOld.ExpiresAt = now + 60
	if isSameRunBaselineRenewal(renewalClientTestToken(validOld), renewalClientTestToken(base)) {
		t.Fatal("client accepted replacement while inherited baseline was still valid")
	}
}

func renewalClientTestToken(claims ordinaryIdentityClaims) string {
	encoded, _ := json.Marshal(claims)
	return base64.RawURLEncoding.EncodeToString(encoded) + "." + base64.RawURLEncoding.EncodeToString(make([]byte, 32))
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
