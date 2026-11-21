package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/browsercapability"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/principalrole"
)

const browserPaneDelegatorScope = "agent-manager:browser-pane.delegate"

var BrowserPaneDelegatorRole = principalrole.NewRole("scenario-authenticator:agent-manager", browserPaneDelegatorScope,
	"agent-manager", "-browser-pane-delegator", "Web Console browser pane delegator")

func isBrowserPaneDelegator(principal identity.Principal) bool {
	source, err := BrowserPaneDelegatorRole.Source(principal)
	return err == nil && source == "web-console"
}

// BrowserPaneReceiptIssuer is the server-to-server BAS boundary. BAS must
// independently resolve the supplied session selector against its live broker
// and bind the result to this run, grant, profile and requested role.
type BrowserPaneReceiptIssuer interface {
	IssueBrowserPaneReceipt(context.Context, uuid.UUID, string, string, string, string) (BrowserPaneReceipt, error)
	IssueBrowserPaneReceiptUntil(context.Context, uuid.UUID, string, string, string, string, time.Time) (BrowserPaneReceipt, error)
	ListBrowserPaneSessions(context.Context, uuid.UUID, string, string) ([]BrowserPaneSession, error)
}

type BrowserPaneControlOwner interface {
	YieldBrowserPaneControl(context.Context, orchestration.BrowserPaneControlBinding) (orchestration.BrowserPaneControlYield, error)
	ReturnBrowserPaneControl(context.Context, orchestration.BrowserPaneControlBinding, string) error
}

// browserPaneRunLookup is implemented by the orchestration owner using its
// repository's Web Console session index. Keeping the selector here narrow
// prevents browser discovery from scanning a capped, user-facing run list.
type browserPaneRunLookup interface {
	ListRunsByWebConsoleThread(context.Context, string, string) ([]*domain.Run, error)
}

type BrowserPaneSession struct {
	SessionID  string `json:"session_id"`
	Mode       string `json:"mode"`
	CanControl bool   `json:"can_control"`
}

type BrowserPaneReceipt struct {
	SessionID    string    `json:"session_id"`
	Role         string    `json:"role"`
	Receipt      string    `json:"receipt"`
	YieldReceipt string    `json:"yield_receipt,omitempty"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
}

type browserPaneResolveRequest struct {
	RunID               string `json:"run_id"`
	WebConsoleSessionID string `json:"web_console_session_id"`
	ThreadID            string `json:"thread_id"`
	SessionID           string `json:"session_id"`
	Role                string `json:"role"`
	OwnerSubject        string `json:"owner_subject,omitempty"`
	YieldReceipt        string `json:"yield_receipt,omitempty"`
}

type browserPaneListRequest struct {
	WebConsoleSessionID string `json:"web_console_session_id"`
	ThreadID            string `json:"thread_id"`
	OwnerSubject        string `json:"owner_subject,omitempty"`
}

// ListBrowserPaneSessions discovers sessions only inside the one exact, running
// human-owned run. BAS independently filters its live broker by that run grant.
func (h *Handler) ListBrowserPaneSessions(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc.RunService == nil || h.browserPaneReceipts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "browser_pane_authority_unavailable"})
		return
	}
	principal, ok := identity.PrincipalFromContext(r.Context())
	if !ok || !principal.IsVerified() {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "verified_pane_caller_required"})
		return
	}
	owner := ""
	if principal.Kind == identity.ActorHuman {
		if _, err := authn.RequireIndependentHuman(r.Context()); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "independent_human_required"})
			return
		}
		if _, err := authn.RequireCapability(r.Context(), "agent-manager:write"); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_capability_required"})
			return
		}
		owner = principal.Subject
	} else if principal.Kind == identity.ActorService && isBrowserPaneDelegator(principal) {
		if _, err := authn.RequireCapability(r.Context(), browserPaneDelegatorScope); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_delegation_scope_required"})
			return
		}
	} else {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_caller_refused"})
		return
	}
	var body browserPaneListRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	dec.DisallowUnknownFields()
	if dec.Decode(&body) != nil || dec.Decode(new(any)) != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	body.WebConsoleSessionID = strings.TrimSpace(body.WebConsoleSessionID)
	body.ThreadID = strings.TrimSpace(body.ThreadID)
	if principal.Kind == identity.ActorService {
		owner = strings.TrimSpace(body.OwnerSubject)
	} else if strings.TrimSpace(body.OwnerSubject) != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "owner_subject_not_accepted"})
		return
	}
	if body.WebConsoleSessionID == "" || body.ThreadID == "" || owner == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_browser_pane_selector"})
		return
	}
	lookup, ok := h.svc.RunService.(browserPaneRunLookup)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "browser_pane_run_lookup_unavailable"})
		return
	}
	runs, err := lookup.ListRunsByWebConsoleThread(r.Context(), body.WebConsoleSessionID, body.ThreadID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "browser_pane_run_lookup_unavailable"})
		return
	}
	var selected *domain.Run
	for _, run := range runs {
		if run == nil || run.Status != domain.RunStatusRunning || run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive || run.OwnerSubject != owner || run.WebConsoleSessionID != body.WebConsoleSessionID || run.SessionID != body.ThreadID || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
			continue
		}
		if selected != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "ambiguous_browser_pane_run"})
			return
		}
		selected = run
	}
	if selected == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "active_browser_run_unavailable"})
		return
	}
	capability := selected.ResolvedConfig.Features.BrowserCapability
	if !capability.Enabled || strings.TrimSpace(capability.GrantReceipt) == "" || strings.TrimSpace(capability.Permissions.ProfileID) == "" || !capability.Permissions.View {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_permission_refused"})
		return
	}
	sessions, err := h.browserPaneReceipts.ListBrowserPaneSessions(r.Context(), selected.ID, capability.GrantReceipt, capability.Permissions.ProfileID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "browser_pane_session_lookup_unavailable"})
		return
	}
	for index := range sessions {
		sessions[index].CanControl = sessions[index].CanControl && capability.Permissions.Control
	}
	writeJSON(w, http.StatusOK, map[string]any{"run_id": selected.ID.String(), "thread_id": body.ThreadID, "sessions": sessions})
}

// ResolveBrowserPane exchanges a human-owned Web Console/run/thread tuple for
// a short-lived BAS-owned stream receipt. Browser-supplied values are selectors
// only: AM proves ownership and saved permissions, then BAS attests the active
// session/lease. Neither raw grant nor lease identity is returned to the client.
func (h *Handler) ResolveBrowserPane(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc.RunService == nil || h.browserPaneReceipts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "browser_pane_authority_unavailable"})
		return
	}
	principal, ok := identity.PrincipalFromContext(r.Context())
	if !ok || !principal.IsVerified() {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "verified_pane_caller_required"})
		return
	}
	ownerSubject := ""
	if principal.Kind == identity.ActorHuman {
		if _, err := authn.RequireIndependentHuman(r.Context()); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "independent_human_required"})
			return
		}
		if _, err := authn.RequireCapability(r.Context(), "agent-manager:write"); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_capability_required"})
			return
		}
		ownerSubject = principal.Subject
	} else if principal.Kind == identity.ActorService && isBrowserPaneDelegator(principal) {
		if _, err := authn.RequireCapability(r.Context(), browserPaneDelegatorScope); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_delegation_scope_required"})
			return
		}
	} else {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_caller_refused"})
		return
	}
	var request browserPaneResolveRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return
	}
	if principal.Kind == identity.ActorService {
		ownerSubject = strings.TrimSpace(request.OwnerSubject)
		if ownerSubject == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "owner_subject_required"})
			return
		}
	} else if strings.TrimSpace(request.OwnerSubject) != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "owner_subject_not_accepted"})
		return
	}
	request.RunID = strings.TrimSpace(request.RunID)
	request.WebConsoleSessionID = strings.TrimSpace(request.WebConsoleSessionID)
	request.ThreadID = strings.TrimSpace(request.ThreadID)
	request.SessionID = strings.TrimSpace(request.SessionID)
	request.Role = strings.TrimSpace(request.Role)
	runID, err := uuid.Parse(request.RunID)
	if err != nil || request.WebConsoleSessionID == "" || request.ThreadID == "" || request.SessionID == "" || (request.Role != "observe" && request.Role != "control") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_browser_pane_selector"})
		return
	}
	run, err := h.svc.RunService.GetRun(r.Context(), runID)
	activeForRole := run != nil && run.Status == domain.RunStatusRunning
	if run != nil && request.Role == "observe" && run.Status == domain.RunStatusParked && run.AwaitHandle != nil && run.AwaitHandle.Producer == orchestration.ProducerBrowserPaneControl && run.AwaitHandle.Deadline != nil && run.AwaitHandle.Deadline.After(time.Now()) {
		activeForRole = true
	}
	if err != nil || run == nil || run.ID != runID || !activeForRole || run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive || run.OwnerSubject == "" || run.OwnerSubject != ownerSubject || run.WebConsoleSessionID != request.WebConsoleSessionID || run.SessionID != request.ThreadID || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_run_binding_refused"})
		return
	}
	capability := run.ResolvedConfig.Features.BrowserCapability
	if !capability.Enabled || strings.TrimSpace(capability.GrantReceipt) == "" || strings.TrimSpace(capability.Permissions.ProfileID) == "" || !capability.Permissions.View || (request.Role == "control" && (!capability.Permissions.Control || (run.ApprovalState != "" && run.ApprovalState != domain.ApprovalStateNone))) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_permission_refused"})
		return
	}
	if request.Role == "control" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser control requires the exclusive yield route"})
		return
	}
	issued, err := h.browserPaneReceipts.IssueBrowserPaneReceipt(r.Context(), runID, capability.GrantReceipt, capability.Permissions.ProfileID, request.SessionID, request.Role)
	if err != nil || issued.SessionID != request.SessionID || issued.Role != request.Role || strings.TrimSpace(issued.Receipt) == "" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_session_refused"})
		return
	}
	writeJSON(w, http.StatusOK, issued)
}

// YieldBrowserPaneControl stops the exact interactive turn through AM's park
// owner before BAS grants the human's control stream. The yield receipt is
// returned only to Web Console's server-side resolver, never to its browser.
func (h *Handler) YieldBrowserPaneControl(w http.ResponseWriter, r *http.Request) {
	request, ownerSubject, ok := h.browserPaneRequest(w, r, true)
	if !ok {
		return
	}
	if h == nil || h.svc.RunService == nil || h.browserPaneReceipts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "browser_pane_authority_unavailable"})
		return
	}
	if request.Role != "control" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "browser_control_role_required"})
		return
	}
	runID, _ := uuid.Parse(request.RunID)
	run, err := h.svc.RunService.GetRun(r.Context(), runID)
	if err != nil || run == nil || run.ID != runID || run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive || run.OwnerSubject != ownerSubject || run.WebConsoleSessionID != request.WebConsoleSessionID || run.SessionID != request.ThreadID || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_run_binding_refused"})
		return
	}
	capability := run.ResolvedConfig.Features.BrowserCapability
	if !capability.Enabled || !capability.Permissions.View || !capability.Permissions.Control || strings.TrimSpace(capability.GrantReceipt) == "" || strings.TrimSpace(capability.Permissions.ProfileID) == "" || (run.ApprovalState != "" && run.ApprovalState != domain.ApprovalStateNone) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_permission_refused"})
		return
	}
	controlOwner, ok := h.svc.RunService.(BrowserPaneControlOwner)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "browser_control_owner_unavailable"})
		return
	}
	binding := orchestration.BrowserPaneControlBinding{RunID: runID, OwnerSubject: ownerSubject, WebConsoleSessionID: request.WebConsoleSessionID, ThreadID: request.ThreadID, BASSessionID: request.SessionID, GrantReceipt: capability.GrantReceipt, ProfileID: capability.Permissions.ProfileID}
	yield, err := controlOwner.YieldBrowserPaneControl(r.Context(), binding)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "browser_control_yield_refused"})
		return
	}
	issued, err := h.browserPaneReceipts.IssueBrowserPaneReceiptUntil(r.Context(), runID, capability.GrantReceipt, capability.Permissions.ProfileID, request.SessionID, "control", yield.ExpiresAt)
	if err != nil || issued.SessionID != request.SessionID || issued.Role != "control" || strings.TrimSpace(issued.Receipt) == "" || issued.ExpiresAt.IsZero() || issued.ExpiresAt.After(yield.ExpiresAt) {
		// A receipt cannot be delivered without matching BAS lease proof. Return
		// the exact AM run immediately; its bounded timeout remains the backstop.
		_ = controlOwner.ReturnBrowserPaneControl(context.Background(), binding, yield.Receipt)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "browser_control_receipt_unavailable"})
		return
	}
	issued.YieldReceipt = yield.Receipt
	writeJSON(w, http.StatusOK, issued)
}

// ReturnBrowserPaneControl is called by the same-origin Web Console owner only
// after BAS has replaced the control stream cookie with observe and connected
// the observer stream. The opaque yield receipt binds the exact human/run/
// thread/session/profile; WakeRun restores only the same persisted run.
func (h *Handler) ReturnBrowserPaneControl(w http.ResponseWriter, r *http.Request) {
	request, ownerSubject, ok := h.browserPaneRequest(w, r, true)
	if !ok {
		return
	}
	if h == nil || h.svc.RunService == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "browser_control_owner_unavailable"})
		return
	}
	if strings.TrimSpace(request.YieldReceipt) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "browser_control_yield_receipt_required"})
		return
	}
	if request.Role != "observe" {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "browser_observer_stream_required_before_return"})
		return
	}
	runID, _ := uuid.Parse(request.RunID)
	run, err := h.svc.RunService.GetRun(r.Context(), runID)
	if err != nil || run == nil || run.OwnerSubject != ownerSubject || run.WebConsoleSessionID != request.WebConsoleSessionID || run.SessionID != request.ThreadID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_run_binding_refused"})
		return
	}
	capability := (*browsercapability.Capability)(nil)
	if run.ResolvedConfig != nil {
		capability = run.ResolvedConfig.Features.BrowserCapability
	}
	if capability == nil || !capability.Enabled || !capability.Permissions.Control || strings.TrimSpace(capability.Permissions.ProfileID) == "" || strings.TrimSpace(capability.GrantReceipt) == "" || (run.ApprovalState != "" && run.ApprovalState != domain.ApprovalStateNone) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_permission_refused"})
		return
	}
	controlOwner, ok := h.svc.RunService.(BrowserPaneControlOwner)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "browser_control_owner_unavailable"})
		return
	}
	binding := orchestration.BrowserPaneControlBinding{RunID: runID, OwnerSubject: ownerSubject, WebConsoleSessionID: request.WebConsoleSessionID, ThreadID: request.ThreadID, BASSessionID: request.SessionID, GrantReceipt: capability.GrantReceipt, ProfileID: capability.Permissions.ProfileID}
	if err := controlOwner.ReturnBrowserPaneControl(r.Context(), binding, request.YieldReceipt); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "browser_control_return_refused"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"returned": true})
}

func (h *Handler) browserPaneRequest(w http.ResponseWriter, r *http.Request, allowOwnerFromService bool) (browserPaneResolveRequest, string, bool) {
	principal, ok := identity.PrincipalFromContext(r.Context())
	if !ok || !principal.IsVerified() {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "verified_pane_caller_required"})
		return browserPaneResolveRequest{}, "", false
	}
	owner := ""
	if principal.Kind == identity.ActorHuman {
		if _, err := authn.RequireIndependentHuman(r.Context()); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "independent_human_required"})
			return browserPaneResolveRequest{}, "", false
		}
		if _, err := authn.RequireCapability(r.Context(), "agent-manager:write"); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_capability_required"})
			return browserPaneResolveRequest{}, "", false
		}
		owner = principal.Subject
	} else if principal.Kind != identity.ActorService || !isBrowserPaneDelegator(principal) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_caller_refused"})
		return browserPaneResolveRequest{}, "", false
	} else if _, err := authn.RequireCapability(r.Context(), browserPaneDelegatorScope); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_pane_delegation_scope_required"})
		return browserPaneResolveRequest{}, "", false
	}
	var request browserPaneResolveRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || decoder.Decode(new(any)) != io.EOF {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_request"})
		return browserPaneResolveRequest{}, "", false
	}
	if principal.Kind == identity.ActorService && allowOwnerFromService {
		owner = strings.TrimSpace(request.OwnerSubject)
	} else if strings.TrimSpace(request.OwnerSubject) != "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "owner_subject_not_accepted"})
		return browserPaneResolveRequest{}, "", false
	}
	request.RunID = strings.TrimSpace(request.RunID)
	request.WebConsoleSessionID = strings.TrimSpace(request.WebConsoleSessionID)
	request.ThreadID = strings.TrimSpace(request.ThreadID)
	request.SessionID = strings.TrimSpace(request.SessionID)
	request.Role = strings.TrimSpace(request.Role)
	if _, err := uuid.Parse(request.RunID); err != nil || request.WebConsoleSessionID == "" || request.ThreadID == "" || request.SessionID == "" || owner == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_browser_pane_selector"})
		return browserPaneResolveRequest{}, "", false
	}
	return request, owner, true
}
