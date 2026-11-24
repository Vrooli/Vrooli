package bindings

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
)

func verifiedBrowserCaller(t *testing.T) context.Context {
	t.Helper()
	var verified context.Context
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set(cliutil.HeaderAgentIdentityToken, "verified-token")
	provenance.Middleware(provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		return &cliutil.VerifyResult{Valid: token == "verified-token", RunStatus: "running", Claims: &cliutil.VerifiedClaims{RunID: "run-1"}}, nil
	}))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { verified = r.Context() })).ServeHTTP(httptest.NewRecorder(), request)
	return verified
}

func TestBrowserContextRelayResolvesFreshGrantAndAttachesOnlyToAllowlistedBinding(t *testing.T) {
	var resolverCalls int
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resolverCalls++
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Errorf("resolve-context body=%q err=%v, want empty", body, err)
		}
		if r.Method != http.MethodPost || r.URL.Path != browserContextPath || r.Header.Get(cliutil.HeaderAgentIdentityToken) != "verified-token" {
			t.Errorf("resolve-context request=%s %s identity=%q", r.Method, r.URL.Path, r.Header.Get(cliutil.HeaderAgentIdentityToken))
		}
		_ = json.NewEncoder(w).Encode(browserContext{CallerID: "run:run-1", RunID: "run-1", GrantReceipt: "opaque-grant", Harness: "codex", Capability: json.RawMessage(`{"enabled":true,"grantReceipt":"opaque-grant","mode":"managed","permissions":{"profileId":"profile-1","profileMode":"fresh"}}`)})
	}))
	defer owner.Close()
	preparer := NewBrowserContextRequestPreparer(owner.Client(), func(context.Context, string) (string, error) { return owner.URL, nil })

	for _, id := range []string{"browser-automation-studio/session/open", BrowserRunTaskBindingID, "browser-automation-studio/task/replay-flow"} {
		request := httptest.NewRequest(http.MethodPost, "http://bas.test/connect", nil)
		if err := preparer(verifiedBrowserCaller(t), id, request); err != nil {
			t.Fatalf("prepare %s: %v", id, err)
		}
		if got := request.Header.Values(cliutil.HeaderAgentIdentityToken); len(got) != 1 || got[0] != "verified-token" {
			t.Fatalf("identity headers=%q", got)
		}
		if got := request.Header.Values(browserGrantReceiptHeader); len(got) != 1 || got[0] != "opaque-grant" {
			t.Fatalf("grant headers=%q", got)
		}
	}
	if resolverCalls != 3 {
		t.Fatalf("resolve-context calls=%d, want fresh context per invocation", resolverCalls)
	}

	ordinary := httptest.NewRequest(http.MethodPost, "http://bas.test/connect", nil)
	if err := preparer(verifiedBrowserCaller(t), "browser-automation-studio/workflows/get", ordinary); err != nil {
		t.Fatal(err)
	}
	if resolverCalls != 3 || len(ordinary.Header.Values(cliutil.HeaderAgentIdentityToken)) != 0 || len(ordinary.Header.Values(browserGrantReceiptHeader)) != 0 {
		t.Fatalf("ordinary binding inherited browser proof: headers=%v/%v resolve_calls=%d", ordinary.Header.Values(cliutil.HeaderAgentIdentityToken), ordinary.Header.Values(browserGrantReceiptHeader), resolverCalls)
	}
}

func TestBrowserContextRelayFailsClosedWithoutVerifiedIdentityOrCompleteContext(t *testing.T) {
	var calls int
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = io.WriteString(w, `{"caller_id":"run:run-1","run_id":"run-1"}`)
	}))
	defer owner.Close()
	preparer := NewBrowserContextRequestPreparer(owner.Client(), func(context.Context, string) (string, error) { return owner.URL, nil })

	request := httptest.NewRequest(http.MethodPost, "http://bas.test/connect", nil)
	if err := preparer(context.Background(), "browser-automation-studio/session/open", request); err == nil {
		t.Fatal("unverified caller was accepted")
	}
	if calls != 0 {
		t.Fatalf("unverified caller reached Agent Manager %d times", calls)
	}

	if err := preparer(verifiedBrowserCaller(t), "browser-automation-studio/session/open", request); err == nil {
		t.Fatal("incomplete Agent Manager context was accepted")
	}
	if calls != 1 || len(request.Header.Values(cliutil.HeaderAgentIdentityToken)) != 0 || len(request.Header.Values(browserGrantReceiptHeader)) != 0 {
		t.Fatalf("failed relay attached proof: calls=%d headers=%v/%v", calls, request.Header.Values(cliutil.HeaderAgentIdentityToken), request.Header.Values(browserGrantReceiptHeader))
	}
}

func TestBrowserContextRelayRequiresMatchingPersistedCapability(t *testing.T) {
	cases := []struct {
		name       string
		capability string
	}{
		{name: "disabled", capability: `{"enabled":false,"grantReceipt":"opaque-grant","mode":"managed","permissions":{"profileId":"profile-1","profileMode":"fresh"}}`},
		{name: "different grant", capability: `{"enabled":true,"grantReceipt":"other-grant","mode":"managed","permissions":{"profileId":"profile-1","profileMode":"fresh"}}`},
		{name: "missing profile", capability: `{"enabled":true,"grantReceipt":"opaque-grant","mode":"managed","permissions":{"profileMode":"fresh"}}`},
		{name: "malformed", capability: `[]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, `{"caller_id":"run:run-1","run_id":"run-1","grant_receipt":"opaque-grant","capability":`+tc.capability+`}`)
			}))
			defer owner.Close()
			preparer := NewBrowserContextRequestPreparer(owner.Client(), func(context.Context, string) (string, error) { return owner.URL, nil })
			request := httptest.NewRequest(http.MethodPost, "http://bas.test/connect", nil)
			if err := preparer(verifiedBrowserCaller(t), "browser-automation-studio/session/observe", request); err == nil {
				t.Fatal("invalid persisted browser capability was accepted")
			}
			if len(request.Header.Values(cliutil.HeaderAgentIdentityToken)) != 0 || len(request.Header.Values(browserGrantReceiptHeader)) != 0 {
				t.Fatalf("failed capability validation attached proof headers: %v/%v", request.Header.Values(cliutil.HeaderAgentIdentityToken), request.Header.Values(browserGrantReceiptHeader))
			}
		})
	}
}

func TestBrowserContextRelayRejectsGrantForAnotherRun(t *testing.T) {
	owner := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"caller_id":"run:other-run","run_id":"other-run","grant_receipt":"other-grant"}`)
	}))
	defer owner.Close()
	preparer := NewBrowserContextRequestPreparer(owner.Client(), func(context.Context, string) (string, error) { return owner.URL, nil })
	request := httptest.NewRequest(http.MethodPost, "http://bas.test/connect", nil)
	if err := preparer(verifiedBrowserCaller(t), "browser-automation-studio/session/observe", request); err == nil {
		t.Fatal("grant from another run was accepted")
	}
	if len(request.Header.Values(cliutil.HeaderAgentIdentityToken)) != 0 || len(request.Header.Values(browserGrantReceiptHeader)) != 0 {
		t.Fatalf("mismatched run attached proof headers: %v/%v", request.Header.Values(cliutil.HeaderAgentIdentityToken), request.Header.Values(browserGrantReceiptHeader))
	}
}
