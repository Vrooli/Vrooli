package agentaccess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

func TestPaneStreamReceiptAuthorityRejectsWrongBrokerBindingAndInactiveSession(t *testing.T) {
	runID := uuid.New()
	e := &entry{sessionID: "bas-session", leaseID: "lease-1", ownerID: "owner-1", callerID: "run:" + runID.String(),
		grantReceipt: "grant-1", permissionRef: "permission-ref", profile: sessionpermission.Profile{ProfileID: "profile-1", View: true}}
	authority := NewPaneStreamReceiptAuthority(&Broker{entries: map[string]*entry{e.sessionID: e}})
	if _, _, err := authority.IssueBrowserPaneReceipt(context.Background(), runID, e.grantReceipt, "profile-1", e.sessionID, "observe"); err == nil || !strings.Contains(err.Error(), "session") {
		t.Fatalf("session without BAS Hub stream should fail closed, err=%v", err)
	}
	if _, _, err := authority.IssueBrowserPaneReceipt(context.Background(), uuid.New(), e.grantReceipt, "profile-1", e.sessionID, "observe"); err == nil {
		t.Fatal("wrong run unexpectedly received a pane receipt")
	}
}

func TestBrowserPaneCandidateFilteringBindsLiveViewGrantAndProfile(t *testing.T) {
	runID := uuid.New()
	newEntry := func() *entry {
		return &entry{sessionID: "session-a", callerID: "run:" + runID.String(), grantReceipt: "grant-a", permissionRef: "profile-a", profile: sessionpermission.Profile{ProfileID: "profile-a", View: true, Approval: sessionpermission.ApprovalNone}, session: &session.Session{}}
	}
	if !browserPaneEntryMatches(newEntry(), runID, "grant-a", "profile-a") {
		t.Fatal("exact active view binding should match")
	}
	cases := []struct {
		name   string
		mutate func(*entry)
	}{
		{"wrong run", func(e *entry) { e.callerID = "run:" + uuid.NewString() }},
		{"wrong grant", func(e *entry) { e.grantReceipt = "grant-b" }},
		{"wrong profile reference", func(e *entry) { e.permissionRef = "profile-b" }},
		{"wrong profile id", func(e *entry) { e.profile.ProfileID = "profile-b" }},
		{"no view", func(e *entry) { e.profile.View = false }},
		{"per-action approval", func(e *entry) { e.profile.Approval = sessionpermission.ApprovalEveryAction }},
		{"closed", func(e *entry) { e.closed = true }},
		{"revoking", func(e *entry) { e.revoking.Store(true) }},
		{"no owned session", func(e *entry) { e.session = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := newEntry()
			tc.mutate(candidate)
			if browserPaneEntryMatches(candidate, runID, "grant-a", "profile-a") {
				t.Fatal("mismatched or inactive broker entry was listed")
			}
		})
	}
	if base := newEntry(); browserPaneEntryMatches(base, uuid.Nil, "grant-a", "profile-a") || browserPaneEntryMatches(base, runID, "", "profile-a") || browserPaneEntryMatches(base, runID, "grant-a", "") {
		t.Fatal("empty selector matched a broker entry")
	}
}

func TestBrowserPaneListRequiresManagerOwnedSession(t *testing.T) {
	runID := uuid.New()
	e := &entry{sessionID: "session-a", callerID: "run:" + runID.String(), grantReceipt: "grant-a", permissionRef: "profile-a", profile: sessionpermission.Profile{ProfileID: "profile-a", View: true, Approval: sessionpermission.ApprovalNone}, session: &session.Session{}}
	authority := NewPaneStreamReceiptAuthority(&Broker{entries: map[string]*entry{e.sessionID: e}})
	if candidates := authority.ListBrowserPaneSessions(runID, "grant-a", "profile-a"); len(candidates) != 0 {
		t.Fatalf("session without owner manager was listed: %#v", candidates)
	}
}

func TestPaneStreamReceiptIsRoleBoundAndSingleUse(t *testing.T) {
	runID := uuid.New()
	expires := time.Now().Add(time.Minute)
	receipt := "synthetic-owner-issued-opaque-receipt"
	digest := sha256.Sum256([]byte(receipt))
	key := hex.EncodeToString(digest[:])
	authority := NewPaneStreamReceiptAuthority(&Broker{})
	authority.records[key] = browserPaneReceiptRecord{runID: runID, claims: StreamReceiptClaims{
		SessionID: "session-1", LeaseID: "lease-1", OwnerID: "owner-1", CallerID: "run:" + runID.String(),
		PermissionRef: "profile-ref", Role: StreamObserve, ExpiresAt: expires,
	}}
	claims, err := authority.VerifyBrowserStreamReceipt(context.Background(), receipt, "session-1", StreamObserve)
	if err != nil || claims.LeaseID != "lease-1" || claims.Role != StreamObserve {
		t.Fatalf("valid receipt claims=%+v err=%v", claims, err)
	}
	if _, err := authority.VerifyBrowserStreamReceipt(context.Background(), receipt, "session-1", StreamObserve); err == nil {
		t.Fatal("receipt replay unexpectedly verified")
	}

	authority.records[key] = browserPaneReceiptRecord{runID: runID, claims: StreamReceiptClaims{SessionID: "session-1", Role: StreamControl, ExpiresAt: expires}}
	if _, err := authority.VerifyBrowserStreamReceipt(context.Background(), receipt, "session-1", StreamObserve); err == nil {
		t.Fatal("role-mismatched receipt unexpectedly verified")
	}
}

func TestPaneReceiptRouteRequiresDedicatedIssuerRole(t *testing.T) {
	router := chi.NewRouter()
	RegisterPaneStreamReceiptRoutes(router, NewPaneStreamReceiptAuthority(&Broker{}), nil)
	req := httptest.NewRequest(http.MethodPost, "/browser/internal/pane-stream-receipts/issue", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer ignored")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured role should fail closed, status=%d body=%s", res.Code, res.Body.String())
	}

	verifier := browserEvidenceTestTokenVerifier{principal: identity.Principal{Kind: identity.ActorService, Subject: "agent-manager-browser-evidence-verifier", Verified: true}}
	router = chi.NewRouter()
	RegisterPaneStreamReceiptRoutes(router, NewPaneStreamReceiptAuthority(&Broker{}), verifier)
	req = httptest.NewRequest(http.MethodPost, "/browser/internal/pane-stream-receipts/issue", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer test")
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("evidence verifier must not satisfy pane issuer: status=%d body=%s", res.Code, res.Body.String())
	}
	issuer := identity.Principal{Kind: identity.ActorService, Subject: BrowserPaneIssuerRole.Principal("agent-manager"), Verified: true, Audience: BrowserPaneIssuerRole.Audience, Scopes: []string{BrowserPaneIssuerRole.Scope}}
	router = chi.NewRouter()
	RegisterPaneStreamReceiptRoutes(router, NewPaneStreamReceiptAuthority(&Broker{}), browserEvidenceTestTokenVerifier{principal: issuer})
	req = httptest.NewRequest(http.MethodPost, "/browser/internal/pane-stream-receipts/list", strings.NewReader(`{"run_id":"`+uuid.NewString()+`","grant_receipt":"grant","profile_id":"profile"}`))
	req.Header.Set("Authorization", "Bearer test")
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"sessions":[]`) {
		t.Fatalf("dedicated issuer should read only an empty safe candidate list: status=%d body=%s", res.Code, res.Body.String())
	}
}
