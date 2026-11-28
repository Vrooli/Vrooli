package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/browsercapability"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/identity"
)

type paneReceiptIssuerFake struct {
	called  bool
	listed  bool
	grant   string
	profile string
	session string
	role    string
	err     error
}

func (f *paneReceiptIssuerFake) IssueBrowserPaneReceipt(_ context.Context, _ uuid.UUID, grant, profile, session, role string) (BrowserPaneReceipt, error) {
	f.called = true
	f.grant, f.profile, f.session, f.role = grant, profile, session, role
	return BrowserPaneReceipt{SessionID: session, Role: role, Receipt: "opaque-pane-receipt"}, f.err
}
func (f *paneReceiptIssuerFake) IssueBrowserPaneReceiptUntil(ctx context.Context, run uuid.UUID, grant, profile, session, role string, notAfter time.Time) (BrowserPaneReceipt, error) {
	issued, err := f.IssueBrowserPaneReceipt(ctx, run, grant, profile, session, role)
	issued.ExpiresAt = notAfter.Add(-time.Second)
	return issued, err
}
func (f *paneReceiptIssuerFake) ListBrowserPaneSessions(_ context.Context, _ uuid.UUID, grant, profile string) ([]BrowserPaneSession, error) {
	f.listed = true
	f.grant = grant
	f.profile = profile
	return []BrowserPaneSession{{SessionID: "bas-active", Mode: "managed", CanControl: true}}, nil
}

type paneListRunService struct {
	orchestration.RunService
	runs []*domain.Run
}

func (f paneListRunService) ListRunsByWebConsoleThread(_ context.Context, webSession, thread string) ([]*domain.Run, error) {
	matched := make([]*domain.Run, 0, len(f.runs))
	for _, run := range f.runs {
		if run != nil && run.WebConsoleSessionID == webSession && run.SessionID == thread {
			matched = append(matched, run)
		}
	}
	return matched, nil
}

func TestResolveBrowserPaneRequiresExactHumanOwnedRunThreadAndPermissions(t *testing.T) {
	runID := uuid.New()
	grant := "private-run-grant"
	run := &domain.Run{
		ID: runID, Status: domain.RunStatusRunning, ExecutionMode: domain.ExecutionModeInteractive,
		OwnerSubject: "human-1", WebConsoleSessionID: "web-1", SessionID: "thread-1",
		ResolvedConfig: &domain.RunConfig{Features: domain.FeatureFlags{BrowserCapability: &browsercapability.Capability{
			Enabled: true, GrantReceipt: grant, Permissions: browsercapability.PermissionProfileSpec{ProfileID: "profile-1", View: true, Control: true},
		}}},
	}
	issuer := &paneReceiptIssuerFake{}
	h := &Handler{svc: orchestration.HandlerServices{RunService: browserRunFake{run: run}}, browserPaneReceipts: issuer}
	call := func(subject, session, thread, role string) *httptest.ResponseRecorder {
		body, _ := json.Marshal(browserPaneResolveRequest{RunID: runID.String(), WebConsoleSessionID: session, ThreadID: thread, SessionID: "bas-session-1", Role: role})
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/pane/resolve", strings.NewReader(string(body)))
		principal := identity.Principal{Kind: identity.ActorHuman, Subject: subject, Verified: true, Scopes: []string{"agent-manager:write"}}
		req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
		res := httptest.NewRecorder()
		h.ResolveBrowserPane(res, req)
		return res
	}
	if got := call("other-human", "web-1", "thread-1", "observe"); got.Code != http.StatusForbidden || issuer.called {
		t.Fatalf("foreign human: status=%d issuer_called=%t", got.Code, issuer.called)
	}
	if got := call("human-1", "web-other", "thread-1", "observe"); got.Code != http.StatusForbidden || issuer.called {
		t.Fatalf("wrong Web Console session: status=%d issuer_called=%t", got.Code, issuer.called)
	}
	if got := call("human-1", "web-1", "thread-other", "observe"); got.Code != http.StatusForbidden || issuer.called {
		t.Fatalf("wrong thread: status=%d issuer_called=%t", got.Code, issuer.called)
	}
	got := call("human-1", "web-1", "thread-1", "observe")
	if got.Code != http.StatusOK || !issuer.called {
		t.Fatalf("authorized observer: status=%d issuer_called=%t body=%s", got.Code, issuer.called, got.Body.String())
	}
	if issuer.grant != grant || issuer.profile != "profile-1" || issuer.session != "bas-session-1" || issuer.role != "observe" {
		t.Fatalf("issuer binding mismatch: %#v", issuer)
	}
	if strings.Contains(got.Body.String(), grant) || strings.Contains(got.Body.String(), "lease") {
		t.Fatalf("response disclosed authority internals: %s", got.Body.String())
	}
	run.ApprovalState = domain.ApprovalStatePending
	issuer.called = false
	if got := call("human-1", "web-1", "thread-1", "control"); got.Code != http.StatusForbidden || issuer.called {
		t.Fatalf("control during unresolved approval: status=%d issuer_called=%t body=%s", got.Code, issuer.called, got.Body.String())
	}
	run.ApprovalState = domain.ApprovalStateNone
	run.ResolvedConfig.Features.BrowserCapability.Permissions.Control = false
	if got := call("human-1", "web-1", "thread-1", "control"); got.Code != http.StatusForbidden || issuer.called {
		t.Fatalf("control without persisted grant: status=%d issuer_called=%t body=%s", got.Code, issuer.called, got.Body.String())
	}
}

func TestResolveBrowserPaneFailsClosedWithoutReceiptIssuerOrViewPermission(t *testing.T) {
	runID := uuid.New()
	run := &domain.Run{
		ID: runID, Status: domain.RunStatusRunning, ExecutionMode: domain.ExecutionModeInteractive,
		OwnerSubject: "human-1", WebConsoleSessionID: "web-1", SessionID: "thread-1",
		ResolvedConfig: &domain.RunConfig{Features: domain.FeatureFlags{BrowserCapability: &browsercapability.Capability{
			Enabled: true, GrantReceipt: "grant", Permissions: browsercapability.PermissionProfileSpec{ProfileID: "profile-1"},
		}}},
	}
	request := func(h *Handler, role string) *httptest.ResponseRecorder {
		body := `{"run_id":"` + runID.String() + `","web_console_session_id":"web-1","thread_id":"thread-1","session_id":"bas-session","role":"` + role + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/pane/resolve", strings.NewReader(body))
		principal := identity.Principal{Kind: identity.ActorHuman, Subject: "human-1", Verified: true, Scopes: []string{"agent-manager:write"}}
		req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
		res := httptest.NewRecorder()
		h.ResolveBrowserPane(res, req)
		return res
	}
	if got := request(&Handler{svc: orchestration.HandlerServices{RunService: browserRunFake{run: run}}}, "observe"); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing BAS issuer status=%d", got.Code)
	}
	issuer := &paneReceiptIssuerFake{}
	if got := request(&Handler{svc: orchestration.HandlerServices{RunService: browserRunFake{run: run}}, browserPaneReceipts: issuer}, "observe"); got.Code != http.StatusForbidden || issuer.called {
		t.Fatalf("missing view permission status=%d issuer_called=%t", got.Code, issuer.called)
	}
}

func TestResolveBrowserPaneAcceptsOnlyDedicatedWebConsoleServiceRole(t *testing.T) {
	runID := uuid.New()
	run := &domain.Run{ID: runID, Status: domain.RunStatusRunning, ExecutionMode: domain.ExecutionModeInteractive,
		OwnerSubject: "human-1", WebConsoleSessionID: "web-1", SessionID: "thread-1",
		ResolvedConfig: &domain.RunConfig{Features: domain.FeatureFlags{BrowserCapability: &browsercapability.Capability{
			Enabled: true, GrantReceipt: "private-grant", Permissions: browsercapability.PermissionProfileSpec{ProfileID: "profile-1", View: true},
		}}}}
	issuer := &paneReceiptIssuerFake{}
	h := &Handler{svc: orchestration.HandlerServices{RunService: browserRunFake{run: run}}, browserPaneReceipts: issuer}
	body := `{"run_id":"` + runID.String() + `","web_console_session_id":"web-1","thread_id":"thread-1","session_id":"bas-1","role":"observe","owner_subject":"human-1"}`
	call := func(subject string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/pane/resolve", strings.NewReader(body))
		principal := identity.Principal{Kind: identity.ActorService, Subject: subject, Verified: true, Audience: "scenario-authenticator:agent-manager", Scopes: []string{browserPaneDelegatorScope}}
		req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
		res := httptest.NewRecorder()
		h.ResolveBrowserPane(res, req)
		return res
	}
	if got := call("unrelated-service"); got.Code != http.StatusForbidden || issuer.called {
		t.Fatalf("unrelated service: status=%d issuer_called=%t", got.Code, issuer.called)
	}
	if got := call(BrowserPaneDelegatorRole.Principal("web-console")); got.Code != http.StatusOK || !issuer.called {
		t.Fatalf("Web Console role: status=%d issuer_called=%t body=%s", got.Code, issuer.called, got.Body.String())
	}
}

func TestListBrowserPaneSessionsRequiresOneExactRunningOwnerRun(t *testing.T) {
	runID := uuid.New()
	run := &domain.Run{ID: runID, Status: domain.RunStatusRunning, ExecutionMode: domain.ExecutionModeInteractive, OwnerSubject: "human-1", WebConsoleSessionID: "web-1", SessionID: "thread-1", ResolvedConfig: &domain.RunConfig{Features: domain.FeatureFlags{BrowserCapability: &browsercapability.Capability{Enabled: true, GrantReceipt: "private-grant", Permissions: browsercapability.PermissionProfileSpec{ProfileID: "profile-1", View: true, Control: true}}}}}
	issuer := &paneReceiptIssuerFake{}
	h := &Handler{svc: orchestration.HandlerServices{RunService: paneListRunService{runs: []*domain.Run{run}}}, browserPaneReceipts: issuer}
	call := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/browser/pane/sessions", strings.NewReader(`{"web_console_session_id":"web-1","thread_id":"thread-1"}`))
		req = req.WithContext(identity.WithPrincipal(req.Context(), identity.Principal{Kind: identity.ActorHuman, Subject: "human-1", Verified: true, Scopes: []string{"agent-manager:write"}}))
		res := httptest.NewRecorder()
		h.ListBrowserPaneSessions(res, req)
		return res
	}
	got := call()
	if got.Code != http.StatusOK || !issuer.listed || issuer.grant != "private-grant" || issuer.profile != "profile-1" {
		t.Fatalf("list status=%d issued=%t body=%s", got.Code, issuer.listed, got.Body.String())
	}
	if strings.Contains(got.Body.String(), "private-grant") || strings.Contains(got.Body.String(), "lease") {
		t.Fatalf("pane list leaked authority: %s", got.Body.String())
	}
	if !strings.Contains(got.Body.String(), `"can_control":true`) {
		t.Fatalf("AM control permission was not reflected in candidate: %s", got.Body.String())
	}
	capability := run.ResolvedConfig.Features.BrowserCapability
	capability.Permissions.Control = false
	issuer.listed = false
	got = call()
	if got.Code != http.StatusOK || !issuer.listed || !strings.Contains(got.Body.String(), `"can_control":false`) {
		t.Fatalf("candidate exceeded AM's persisted control permission: status=%d listed=%t body=%s", got.Code, issuer.listed, got.Body.String())
	}
	capability.Permissions.Control = true
	issuer.listed = false
	h.svc.RunService = paneListRunService{runs: []*domain.Run{run, run}}
	got = call()
	if got.Code != http.StatusConflict || issuer.listed {
		t.Fatalf("ambiguous match status=%d issued=%t body=%s", got.Code, issuer.listed, got.Body.String())
	}
	issuer.listed = false
	tooMany := make([]*domain.Run, 1001)
	for i := range tooMany {
		tooMany[i] = run
	}
	h.svc.RunService = paneListRunService{runs: tooMany}
	got = call()
	if got.Code != http.StatusConflict || issuer.listed || !strings.Contains(got.Body.String(), "ambiguous_browser_pane_run") {
		t.Fatalf("large exact lookup did not detect ambiguity: status=%d listed=%t body=%s", got.Code, issuer.listed, got.Body.String())
	}
}
