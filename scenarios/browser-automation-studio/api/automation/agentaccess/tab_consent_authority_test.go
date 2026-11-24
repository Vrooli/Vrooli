package agentaccess

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

func TestAgentManagerTabConsentAuthorityUsesDedicatedServiceRoleAndRunGrant(t *testing.T) {
	var issueSeen, verifySeen, revokeSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tab-consent-service-token" {
			t.Errorf("request omitted separate service and run credentials: headers=%v", r.Header)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		switch r.URL.Path {
		case "/api/v1/browser/tab-consent/issue":
			if r.Header.Get("X-Agent-Identity-Token") != "signed-run-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "grant-1" {
				t.Errorf("issue request omitted run credentials: headers=%v", r.Header)
			}
			issueSeen = true
			if body["tab_id"] != "tab-5" || body["pairing_nonce"] != "nonce-1" || body["profile_id"] != "profile-1" {
				t.Errorf("issue request not bound to selected tab: %#v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"consent_receipt": "opaque-consent"})
		case "/api/v1/browser/tab-consent/verify":
			if r.Header.Get("X-Agent-Identity-Token") != "signed-run-token" || r.Header.Get("X-Vrooli-Browser-Grant-Receipt") != "grant-1" {
				t.Errorf("verify request omitted run credentials: headers=%v", r.Header)
			}
			verifySeen = true
			if body["consent_receipt"] != "opaque-consent" || body["tab_id"] != "tab-5" || body["session_id"] != "session-1" {
				t.Errorf("verify request not bound to selected tab: %#v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		case "/api/v1/browser/tab-consent/revoke":
			revokeSeen = true
			if body["consent_receipt"] != "opaque-consent" || body["run_id"] != "run-1" || body["session_id"] != "session-1" {
				t.Errorf("revoke request not bound to issued lease: %#v", body)
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			http.Error(w, "unexpected route", http.StatusNotFound)
		}
	}))
	defer server.Close()
	authority := &AgentManagerTabConsentAuthority{RunContext: &AgentManagerRunContext{BaseURL: server.URL}, ServiceTokens: browserEvidenceTestTokenSource("tab-consent-service-token")}
	ctx := WithBrowserAgentCredentials(context.Background(), "signed-run-token", "grant-1")
	receipt, err := authority.IssueBrowserTabConsent(ctx, TabConsentRequest{
		CallerID: "run:run-1", GrantReceipt: "grant-1", SessionID: "session-1", LeaseID: "lease-1",
		OwnerID: "run:run-1", PermissionRef: "profile-1", TabID: "tab-5", URL: "https://example.test/form",
		PairingNonce: "nonce-1", ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	if err != nil || receipt != "opaque-consent" {
		t.Fatalf("issue receipt=%q error=%v", receipt, err)
	}
	if err := authority.VerifyBrowserTabConsent(ctx, receipt, sessionpermission.ActionRequest{
		SessionID: "session-1", LeaseID: "lease-1", OwnerID: "run:run-1", TabID: "tab-5", CurrentURL: "https://example.test/form",
	}); err != nil {
		t.Fatalf("verify tab receipt: %v", err)
	}
	if err := authority.RevokeBrowserTabConsent(context.Background(), receipt, "run-1", "session-1"); err != nil {
		t.Fatalf("revoke tab receipt: %v", err)
	}
	if !issueSeen || !verifySeen || !revokeSeen {
		t.Fatalf("issue seen=%v verify seen=%v revoke seen=%v", issueSeen, verifySeen, revokeSeen)
	}
}

func TestAgentManagerTabConsentAuthorityFailsClosedWithoutServiceIdentity(t *testing.T) {
	authority := &AgentManagerTabConsentAuthority{RunContext: &AgentManagerRunContext{BaseURL: "http://127.0.0.1:1"}}
	_, err := authority.IssueBrowserTabConsent(WithBrowserAgentCredentials(context.Background(), "signed-run-token", "grant-1"), TabConsentRequest{GrantReceipt: "grant-1"})
	if err == nil {
		t.Fatal("missing BAS service identity unexpectedly authorized consent issue")
	}
}
