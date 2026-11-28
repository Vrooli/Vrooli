package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/browsercapability"
	"agent-manager/internal/domain"
	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

const (
	BrowserTabConsentBrokerServiceSubject = "browser-automation-studio-browser-tab-consent"
	BrowserTabConsentBrokerScope          = "agent-manager:browser-tab-consent"
)

type issueBrowserTabConsentRequest struct {
	SessionID    string    `json:"session_id"`
	LeaseID      string    `json:"lease_id"`
	OwnerID      string    `json:"owner_id"`
	ProfileID    string    `json:"profile_id"`
	TabID        string    `json:"tab_id"`
	URL          string    `json:"url"`
	PairingNonce string    `json:"pairing_nonce"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type verifyBrowserTabConsentRequest struct {
	Receipt    string `json:"consent_receipt"`
	SessionID  string `json:"session_id"`
	LeaseID    string `json:"lease_id"`
	OwnerID    string `json:"owner_id"`
	TabID      string `json:"tab_id"`
	CurrentURL string `json:"current_url"`
}

type revokeBrowserTabConsentRequest struct {
	Receipt   string `json:"consent_receipt"`
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id"`
}

func (h *Handler) readBrowserCapabilityRequest(w http.ResponseWriter, r *http.Request) (string, string, *browsercapability.Capability, bool) {
	identityToken := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	if identityToken == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "run_identity_required"})
		return "", "", nil, false
	}
	verified, err := h.svc.VerifyIdentityToken(r.Context(), identityToken)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil || verified.RunStatus != domain.RunStatusRunning {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "run_identity_invalid_or_inactive"})
		return "", "", nil, false
	}
	reader, ok := h.svc.RunService.(browserRunReader)
	if !ok {
		writeSimpleError(w, r, "unavailable", "Agent Manager cannot read the authorized run")
		return "", "", nil, false
	}
	runID := verified.Claims.RunID
	run, err := reader.GetRun(r.Context(), runID)
	if err != nil || run == nil || run.ID != runID || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_capability_not_enabled"})
		return "", "", nil, false
	}
	capability := run.ResolvedConfig.Features.BrowserCapability.Clone()
	grantReceipt := browserGrantReceipt(&capability)
	if !capability.Enabled || grantReceipt == "" || strings.TrimSpace(r.Header.Get(browserGrantReceiptHeader)) != grantReceipt {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "browser_grant_mismatch"})
		return "", "", nil, false
	}
	return runID.String(), grantReceipt, &capability, true
}

// IssueBrowserTabConsent is reached only after the bridge extension submits
// its one-time pairing code and selected tab. Agent Manager validates that the
// active run grant explicitly permits this profile, mode and origin, then
// issues one short-lived lease-bound receipt.
func (h *Handler) IssueBrowserTabConsent(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.browserTabConsents == nil || h.svc.IdentityService == nil || h.svc.RunService == nil {
		writeSimpleError(w, r, "unavailable", "browser tab consent authority is unavailable")
		return
	}
	if !requireBrowserTabConsentBroker(w, r) {
		return
	}
	var req issueBrowserTabConsentRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid selected-tab consent request")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeSimpleError(w, r, "body", "selected-tab consent request must contain one JSON object")
		return
	}
	runID, grantReceipt, capability, ok := h.readBrowserCapabilityRequest(w, r)
	if !ok {
		return
	}
	if capability.Mode != browsercapability.ModeAttached || capability.Permissions.ProfileMode != browsercapability.SessionProfileUserAttached || capability.Permissions.ProfileID != strings.TrimSpace(req.ProfileID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "selected_tab_profile_mismatch"})
		return
	}
	receipt, err := h.browserTabConsents.Issue(browsercapability.TabConsentIssue{
		RunID: runID, GrantReceipt: grantReceipt, Profile: capability.Permissions,
		SessionID: req.SessionID, LeaseID: req.LeaseID, OwnerID: req.OwnerID,
		ProfileID: req.ProfileID, TabID: req.TabID, URL: req.URL,
		PairingNonce: req.PairingNonce, ExpiresAt: req.ExpiresAt,
	})
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "selected_tab_consent_denied"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]string{"consent_receipt": receipt})
}

// VerifyBrowserTabConsent checks a receipt against the live run permission and
// exact BAS session/tab binding on each action. A revoked or inactive run, a
// different lease, or a URL outside the run's origin scope fails closed.
func (h *Handler) VerifyBrowserTabConsent(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.browserTabConsents == nil || h.svc.IdentityService == nil || h.svc.RunService == nil {
		writeSimpleError(w, r, "unavailable", "browser tab consent authority is unavailable")
		return
	}
	if !requireBrowserTabConsentBroker(w, r) {
		return
	}
	var req verifyBrowserTabConsentRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid selected-tab consent verification request")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeSimpleError(w, r, "body", "selected-tab consent verification must contain one JSON object")
		return
	}
	runID, grantReceipt, capability, ok := h.readBrowserCapabilityRequest(w, r)
	if !ok {
		return
	}
	if capability.Mode != browsercapability.ModeAttached || capability.Permissions.ProfileMode != browsercapability.SessionProfileUserAttached {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "selected_tab_profile_mismatch"})
		return
	}
	err := h.browserTabConsents.Verify(req.Receipt, browsercapability.TabConsentBinding{
		RunID: runID, GrantReceipt: grantReceipt, Profile: capability.Permissions,
		SessionID: req.SessionID, LeaseID: req.LeaseID, OwnerID: req.OwnerID,
		ProfileID: capability.Permissions.ProfileID, TabID: req.TabID, CurrentURL: req.CurrentURL,
	})
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "selected_tab_consent_denied"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// RevokeBrowserTabConsent removes a receipt from the existing consent owner.
// The dedicated BAS service identity and the exact run/session binding are
// both required; no agent or caller can revoke another session's consent.
func (h *Handler) RevokeBrowserTabConsent(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.browserTabConsents == nil {
		writeSimpleError(w, r, "unavailable", "browser tab consent authority is unavailable")
		return
	}
	if !requireBrowserTabConsentBroker(w, r) {
		return
	}
	var req revokeBrowserTabConsentRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid selected-tab consent revocation request")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeSimpleError(w, r, "body", "selected-tab consent revocation must contain one JSON object")
		return
	}
	if strings.TrimSpace(req.Receipt) == "" || strings.TrimSpace(req.RunID) == "" || strings.TrimSpace(req.SessionID) == "" {
		writeSimpleError(w, r, "body", "selected-tab consent revocation requires receipt, run and session")
		return
	}
	if err := h.browserTabConsents.RevokeSession(req.Receipt, req.RunID, req.SessionID); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "selected_tab_consent_denied"})
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func requireBrowserTabConsentBroker(w http.ResponseWriter, r *http.Request) bool {
	principal, ok := requestidentity.PrincipalFromContext(r.Context())
	if !ok || principal.Kind != requestidentity.ActorService || !principal.IsVerified() || principal.Subject != BrowserTabConsentBrokerServiceSubject {
		http.Error(w, "trusted browser tab consent service identity required", http.StatusForbidden)
		return false
	}
	if _, err := authn.RequireCapability(r.Context(), BrowserTabConsentBrokerScope); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return false
	}
	return true
}
