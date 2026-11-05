package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

func TestRunIdentityVerifiesBrokerRenewedSameRunBaseline(t *testing.T) {
	const runID = "same-attached-run"
	now := time.Now().Unix()
	inherited := identityTestToken(t, runID, now-3600, now-1)
	renewed := identityTestToken(t, runID, now, now+86400)
	startIdentityTestBrokerWithPurpose(t, inherited, `{"baseline_token":"`+renewed+`"}`, http.StatusOK, "ordinary_identity")
	apiCalls := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/identity/verify" {
			t.Errorf("unexpected verification request: %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode verify body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch body.Token {
		case inherited:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"valid":false,"error":"token expired"}`)
		case renewed:
			_, _ = io.WriteString(w, `{"valid":true,"claims":{"run_id":"`+runID+`","task_id":"task","profile_key":"profile"}}`)
		default:
			t.Errorf("unexpected credential verification request (inherited=%t renewed=%t)", body.Token == inherited, body.Token == renewed)
			w.WriteHeader(http.StatusUnauthorized)
		}
	}))
	defer api.Close()
	t.Setenv("AGENT_MANAGER_API_BASE", api.URL)
	t.Setenv(cliutil.EnvIdentityToken, inherited)

	output := captureRunIdentityStdout(t, func() error { return (&App{}).runIdentity([]string{"--json"}) })
	if apiCalls != 2 {
		t.Fatalf("identity verification calls=%d, want expired inherited plus renewed baseline", apiCalls)
	}
	if !strings.Contains(output, runID) {
		t.Fatalf("verified run id missing from output: %q", output)
	}
	if strings.Contains(output, inherited) || strings.Contains(output, renewed) || strings.Contains(output, "operation-token") {
		t.Fatal("identity output exposed credential material")
	}
	if got := os.Getenv(cliutil.EnvIdentityToken); got != inherited {
		t.Fatal("run identity mutated the inherited parent credential")
	}
}

func TestRunIdentityRefusesGrantOnlyCredentialAfterInheritedExpiry(t *testing.T) {
	const runID = "original-parent-run"
	now := time.Now().Unix()
	inherited := identityTestToken(t, runID, now-3600, now-1)
	const legacyToken = "independent-grant-token"
	startIdentityTestBrokerWithPurpose(t, inherited, `{"token":"`+legacyToken+`"}`, http.StatusOK, "ordinary_identity")
	apiCalls := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode verify body: %v", err)
		}
		if body.Token != inherited {
			t.Errorf("grant-only credential reached identity verify (isInherited=%t)", body.Token == inherited)
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"valid":false,"error":"token expired"}`)
	}))
	defer api.Close()
	t.Setenv("AGENT_MANAGER_API_BASE", api.URL)
	t.Setenv(cliutil.EnvIdentityToken, inherited)
	got := captureRunIdentityStdoutAllowError(t, func() error { return (&App{}).runIdentity(nil) })
	if apiCalls != 1 {
		t.Fatalf("identity verification calls=%d, want only inherited token", apiCalls)
	}
	if got.err == nil || !strings.Contains(got.err.Error(), "ordinary agent-manager run identity is unavailable") {
		t.Fatalf("grant-only response error=%v, want unavailable ordinary identity", got.err)
	}
	if strings.Contains(got.err.Error(), legacyToken) || strings.Contains(got.stdout, legacyToken) {
		t.Fatal("grant-only credential leaked through output")
	}
}

func TestRunIdentityPreservesValidInheritedRunOverGrantOnlyBrokerResult(t *testing.T) {
	const runID = "valid-original-run"
	now := time.Now().Unix()
	inherited := identityTestToken(t, runID, now, now+86400)
	brokerCalls := 0
	startIdentityTestBrokerWithCall(t, inherited, `{"token":"independent-grant"}`, http.StatusOK, &brokerCalls)
	apiCalls := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		var body struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode verify body: %v", err)
		}
		if body.Token != inherited {
			t.Errorf("valid inherited identity was replaced (inherited=%t)", body.Token == inherited)
		}
		_, _ = io.WriteString(w, `{"valid":true,"claims":{"run_id":"`+runID+`","task_id":"task","profile_key":"profile"}}`)
	}))
	defer api.Close()
	t.Setenv("AGENT_MANAGER_API_BASE", api.URL)
	t.Setenv(cliutil.EnvIdentityToken, inherited)
	output := captureRunIdentityStdout(t, func() error { return (&App{}).runIdentity([]string{"--json"}) })
	if apiCalls != 1 || brokerCalls != 0 {
		t.Fatalf("identity calls=%d broker calls=%d, want 1 and 0", apiCalls, brokerCalls)
	}
	if !strings.Contains(output, runID) || strings.Contains(output, "independent-grant") {
		t.Fatalf("identity output did not preserve original run: %q", output)
	}
}

func identityTestToken(t *testing.T, runID string, issuedAt, expiresAt int64) string {
	t.Helper()
	claims := struct {
		RunID       string            `json:"run_id"`
		TaskID      string            `json:"task_id"`
		Subject     string            `json:"subject"`
		WorkspaceID string            `json:"workspace_id"`
		Scopes      []string          `json:"scopes"`
		ProfileKey  string            `json:"profile_key"`
		ScopePath   string            `json:"scope_path"`
		IssuedAt    int64             `json:"iat"`
		ExpiresAt   int64             `json:"exp"`
		Meta        map[string]string `json:"meta"`
	}{
		RunID: runID, TaskID: "task", Scopes: []string{}, ProfileKey: "profile", ScopePath: "/agent/test",
		IssuedAt: issuedAt, ExpiresAt: expiresAt, Meta: map[string]string{},
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	signature := make([]byte, sha256.Size)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(signature)
}

func startIdentityTestBroker(t *testing.T, inherited, response string, status int) {
	t.Helper()
	startIdentityTestBrokerWithCall(t, inherited, response, status, nil)
}

func startIdentityTestBrokerWithPurpose(t *testing.T, inherited, response string, status int, purpose string) {
	t.Helper()
	startIdentityTestBrokerWithPurposeAndCall(t, inherited, response, status, purpose, nil)
}

func startIdentityTestBrokerWithCall(t *testing.T, inherited, response string, status int, calls *int) {
	t.Helper()
	startIdentityTestBrokerWithPurposeAndCall(t, inherited, response, status, "", calls)
}

func startIdentityTestBrokerWithPurposeAndCall(t *testing.T, inherited, response string, status int, expectedPurpose string, calls *int) {
	t.Helper()
	dir, err := os.MkdirTemp("", "amri-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	t.Setenv("TMPDIR", dir)
	socket := cliutil.AgentCredentialSocket()
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/credential", func(w http.ResponseWriter, r *http.Request) {
		if calls != nil {
			*calls = *calls + 1
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected broker method %s", r.Method)
		}
		if got := r.URL.Query().Get("purpose"); got != expectedPurpose {
			t.Errorf("credential purpose=%q, want %q", got, expectedPurpose)
		}
		if got := r.Header.Get(cliutil.HeaderAgentIdentityToken); got != inherited {
			t.Errorf("broker did not receive the inherited identity (match=%t)", got == inherited)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	})
	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_ = server.Shutdown(ctx)
	})
}

type runIdentityCallResult struct {
	stdout string
	err    error
}

func captureRunIdentityStdout(t *testing.T, call func() error) string {
	t.Helper()
	got := captureRunIdentityStdoutAllowError(t, call)
	if got.err != nil {
		t.Fatalf("run identity: %v", got.err)
	}
	return got.stdout
}

func captureRunIdentityStdoutAllowError(t *testing.T, call func() error) runIdentityCallResult {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	callErr := call()
	_ = w.Close()
	os.Stdout = old
	data, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	return runIdentityCallResult{stdout: string(data), err: callErr}
}
