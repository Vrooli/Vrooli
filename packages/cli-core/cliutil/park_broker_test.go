//go:build linux

package cliutil

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Unix socket addresses are limited to roughly 108 bytes. Go's per-test
// directory includes the long test name, so use a short private namespace.
func parkBrokerTestNamespace(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "am-park-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func parkOrdinaryToken(runID string, issuedAt, expiresAt int64) string {
	claims := map[string]any{
		"run_id": runID, "task_id": "task-fixture", "subject": "agent-fixture",
		"workspace_id": "workspace-fixture", "scopes": []string{}, "profile_key": "profile-fixture",
		"scope_path": "/fixture", "iat": issuedAt, "exp": expiresAt, "meta": map[string]string{},
	}
	payload, _ := json.Marshal(claims)
	signature := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	return base64.RawURLEncoding.EncodeToString(payload) + "." + signature
}

func TestParkForAwaitUsesBrokerResolvedOrdinaryBaseline(t *testing.T) {
	const runID = "run-ordinary-refresh-fixture"
	oldNow := time.Now().Add(-48 * time.Hour)
	inherited := parkOrdinaryToken(runID, oldNow.Add(-time.Hour).Unix(), oldNow.Unix())
	now := time.Now()
	resolved := parkOrdinaryToken(runID, now.Unix(), now.Add(time.Hour).Unix())

	t.Setenv("TMPDIR", parkBrokerTestNamespace(t))
	if got := AgentCredentialSocket(); !strings.HasPrefix(got, os.TempDir()) {
		t.Fatalf("test broker socket %q is outside temporary directory %q", got, os.TempDir())
	}
	socket := AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(socket), 0700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	broker := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/credential" || r.URL.Query().Get("purpose") != "ordinary_identity" {
			t.Errorf("unexpected broker request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			http.Error(w, "unexpected broker request", http.StatusBadRequest)
			return
		}
		if r.Header.Get(HeaderAgentIdentityToken) != inherited {
			t.Error("broker did not receive inherited identity as renewal context")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"baseline_token": resolved})
	})}
	go func() { _ = broker.Serve(listener) }()
	t.Cleanup(func() { _ = broker.Close() })

	verifiedToken, parkedToken := "", ""
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/identity/verify":
			var body map[string]string
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode verify body: %v", err)
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			verifiedToken = body["token"]
			if verifiedToken != resolved {
				t.Errorf("verified token did not come from ordinary broker response")
				http.Error(w, `{"valid":false,"error":"wrong token"}`, http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(VerifyResult{Valid: true, Claims: &VerifiedClaims{RunID: runID}, RunStatus: "running"})
		case "/api/v1/runs/" + runID + "/park":
			if r.Header.Get(HeaderAgentIdentityToken) != resolved || r.Header.Get(HeaderAgentOperationToken) != "" {
				t.Error("self-park caller must carry the exact verified ordinary baseline and no operation token")
				http.Error(w, "wrong caller credential", http.StatusForbidden)
				return
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode park body: %v", err)
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			parkedToken, _ = body["identity_token"].(string)
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "message": "parked"})
		default:
			t.Errorf("unexpected API request: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer api.Close()
	t.Setenv("AGENT_MANAGER_API_BASE", api.URL)
	t.Setenv(EnvIdentityToken, inherited)

	result, parked, err := ParkForAwait(ParkRequest{Producer: ParkProducerTestGenie, Key: "demo/run"})
	if err != nil || !parked || result == nil {
		t.Fatalf("ParkForAwait() = (%+v, %v, %v)", result, parked, err)
	}
	if verifiedToken != resolved || parkedToken != resolved {
		t.Fatal("park verification and park request must use the same broker-resolved ordinary baseline")
	}
}

func TestParkForAwaitBrokerRefusalStopsBeforeIdentityOrParkRequest(t *testing.T) {
	const runID = "run-ordinary-refusal-fixture"
	oldNow := time.Now().Add(-48 * time.Hour)
	inherited := parkOrdinaryToken(runID, oldNow.Add(-time.Hour).Unix(), oldNow.Unix())

	t.Setenv("TMPDIR", parkBrokerTestNamespace(t))
	socket := AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(socket), 0700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	broker := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/credential" || r.URL.Query().Get("purpose") != "ordinary_identity" {
			t.Errorf("unexpected broker request: %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
			http.Error(w, "unexpected broker request", http.StatusBadRequest)
			return
		}
		if r.Header.Get(HeaderAgentIdentityToken) != inherited {
			t.Error("broker did not receive inherited identity as renewal context")
		}
		http.Error(w, `{"code":"ordinary_identity_refused"}`, http.StatusForbidden)
	})}
	go func() { _ = broker.Serve(listener) }()
	t.Cleanup(func() { _ = broker.Close() })

	var apiRequests atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiRequests.Add(1)
		http.Error(w, "API must not be reached after broker refusal", http.StatusUnauthorized)
	}))
	defer api.Close()
	t.Setenv("AGENT_MANAGER_API_BASE", api.URL)
	t.Setenv(EnvIdentityToken, inherited)

	result, parked, err := ParkForAwait(ParkRequest{Producer: ParkProducerTestGenie, Key: "demo/run"})
	if err == nil || !parked || result != nil {
		t.Fatalf("ParkForAwait() after broker refusal = (%+v, %v, %v), want in-run error", result, parked, err)
	}
	if got := apiRequests.Load(); got != 0 {
		t.Fatalf("identity verification or park request reached Agent Manager after broker refusal: %d requests", got)
	}
}
