package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash/fnv"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	eventstore "agent-manager/internal/adapters/event"
	"agent-manager/internal/browsercapability"
	"agent-manager/internal/domain"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/cli-core/cliutil"
)

type browserRunReader interface {
	GetRun(context.Context, uuid.UUID) (*domain.Run, error)
}

// BrowserEvidenceObservation is returned only by the BAS-owned proof verifier,
// which resolves an opaque proof reference against the active broker entry.
// Agent Manager never copies measurements from the BAS append request body.
type BrowserEvidenceObservation struct {
	Kind           string `json:"kind"`
	SessionID      string `json:"session_id"`
	State          string `json:"state"`
	Action         string `json:"action,omitempty"`
	Outcome        string `json:"outcome,omitempty"`
	SessionMinutes int64  `json:"session_minutes"`
	Actions        int64  `json:"actions"`
	Bytes          int64  `json:"bytes"`
}

// BrowserEvidenceProofVerifier is the authenticated Agent Manager → BAS
// callback boundary. Implementations must authenticate as the dedicated
// Agent Manager verifier service, enforce proof expiry/revocation, resolve the
// closed broker entry, and return only its measured summary. Consumption is a
// separate idempotent step after the caller validates the summary. Transcript
// ingestion may append after consume; LPBS settlement attestation deliberately
// does not append. Without this owner implementation, both paths fail closed.
type BrowserEvidenceProofVerifier interface {
	VerifyBrowserEvidenceProof(ctx context.Context, runID uuid.UUID, grantReceipt, proofReference string) (BrowserEvidenceObservation, error)
	ConsumeBrowserEvidenceProof(ctx context.Context, runID uuid.UUID, proofReference string) error
}

type authorizeBrowserOpenRequest struct {
	Mode         browsercapability.Mode `json:"mode"`
	ProfileID    string                 `json:"profile_id"`
	InitialURL   string                 `json:"initial_url"`
	ResourceRole string                 `json:"resource_role,omitempty"`
}

const browserGrantReceiptHeader = "X-Vrooli-Browser-Grant-Receipt"

// The single Agent Manager API process serializes transcript state checks and
// appends per run/session. Evidence remains run-reported, not BAS attestation.
var browserEvidenceWriteLocks [64]sync.Mutex

type authorizeBrowserOpenResponse struct {
	Authorized   bool                         `json:"authorized"`
	RunID        string                       `json:"run_id"`
	GrantReceipt string                       `json:"grant_receipt"`
	Harness      string                       `json:"harness"`
	ResourceRole string                       `json:"resource_role"`
	Capability   browsercapability.Capability `json:"capability"`
}

type resolveBrowserCallerResponse struct {
	CallerID     string `json:"caller_id"`
	RunID        string `json:"run_id"`
	GrantReceipt string `json:"grant_receipt"`
	ProfileID    string `json:"profile_id"`
}

// ResolveBrowserContext is the trusted relay entry for Program Runtime and
// other server-side bindings: an active signed run token alone resolves the
// principal and receipt. No JSON caller/run/grant data is accepted.
func (h *Handler) ResolveBrowserContext(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc.IdentityService == nil || h.svc.RunService == nil {
		writeSimpleError(w, r, "unavailable", "browser run context resolution is unavailable")
		return
	}
	var body struct{}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&body); err != io.EOF && err != nil {
		writeSimpleError(w, r, "body", "browser context request accepts no identity fields")
		return
	} else if err == nil && decoder.Decode(new(any)) != io.EOF {
		writeSimpleError(w, r, "body", "browser context request must contain at most one empty object")
		return
	}
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "run_identity_required"})
		return
	}
	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "run_identity_invalid"})
		return
	}
	if verified.RunStatus != domain.RunStatusRunning {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "run_not_active"})
		return
	}
	reader, ok := h.svc.RunService.(browserRunReader)
	if !ok {
		writeSimpleError(w, r, "unavailable", "Agent Manager cannot read the authorized run")
		return
	}
	runID := verified.Claims.RunID
	run, err := reader.GetRun(r.Context(), runID)
	if err != nil || run == nil || run.ID != runID || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_capability_not_enabled"})
		return
	}
	capability := run.ResolvedConfig.Features.BrowserCapability.Clone()
	if !capability.Enabled || browserGrantReceipt(&capability) == "" {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_grant_not_available"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"caller_id": "run:" + runID.String(), "run_id": runID.String(), "grant_receipt": capability.GrantReceipt,
		"harness": string(run.ResolvedConfig.RunnerType), "capability": capability,
	})
}

type browserEvidenceRequest struct {
	Kind              string                 `json:"kind"`
	SessionID         string                 `json:"session_id"`
	Mode              browsercapability.Mode `json:"mode"`
	PermissionProfile string                 `json:"permission_profile"`
	State             string                 `json:"state,omitempty"`
	SessionMinutes    int64                  `json:"session_minutes,omitempty"`
	Actions           int64                  `json:"actions,omitempty"`
	Bytes             int64                  `json:"bytes,omitempty"`
	Action            string                 `json:"action,omitempty"`
	Outcome           string                 `json:"outcome,omitempty"`
}

type browserEvidenceProofReference struct {
	ProofReference string `json:"proof_reference"`
}

type resolveBrowserProfileRequest struct {
	ProfileRef string `json:"profile_ref"`
}

const browserEvidenceAuthorizationOperation = "browser-evidence.record"

const (
	browserSettlementAttestationScope   = "agent-manager:browser-evidence.attest"
	browserSettlementAttestationSubject = "landing-page-business-suite-browser-evidence-attester"
)

type browserSettlementAttestationRequest struct {
	GrantReceipt   string `json:"grant_receipt"`
	ProofReference string `json:"proof_reference"`
	ReservationID  string `json:"reservation_id"`
}

type browserSettlementAttestationResponse struct {
	RunID                   string                       `json:"run_id"`
	ReservationID           string                       `json:"reservation_id"`
	GrantReceiptFingerprint string                       `json:"grant_receipt_fingerprint"`
	ProofFingerprint        string                       `json:"proof_fingerprint"`
	Observation             browserSettlementObservation `json:"observation"`
}

type browserSettlementObservation struct {
	Kind           string `json:"kind"`
	SessionID      string `json:"session_id"`
	State          string `json:"state"`
	SessionMinutes int64  `json:"session_minutes"`
	Actions        int64  `json:"actions"`
	Bytes          int64  `json:"bytes"`
}

// AttestBrowserSettlement resolves a BAS broker proof for LPBS settlement.
// This route is intentionally separate from RecordBASBrowserEvidence: it
// requires a dedicated LPBS service identity, writes no transcript event,
// creates no operation grant, and accepts no caller-reported measurements.
// LPBS remains the authority for whether ReservationID is the active, unique
// account hold bound to ProofReference; the returned binding only correlates
// the broker observation with that existing reservation.
func (h *Handler) AttestBrowserSettlement(w http.ResponseWriter, r *http.Request) {
	principal, ok := identity.PrincipalFromContext(r.Context())
	if !ok || !principal.IsVerified() || principal.Kind != identity.ActorService || principal.Subject != browserSettlementAttestationSubject {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "verified_lpbs_browser_settlement_service_required"})
		return
	}
	if _, err := authn.RequireCapability(r.Context(), browserSettlementAttestationScope); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_settlement_attestation_scope_required"})
		return
	}
	if h == nil || h.svc.RunService == nil {
		writeSimpleError(w, r, "unavailable", "browser settlement attestation is unavailable")
		return
	}
	var request browserSettlementAttestationRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&request) != nil || decoder.Decode(new(any)) != io.EOF {
		writeSimpleError(w, r, "body", "browser settlement request is invalid")
		return
	}
	runID, err := uuid.Parse(strings.TrimSpace(mux.Vars(r)["run_id"]))
	if err != nil || runID == uuid.Nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "valid_run_uuid_required"})
		return
	}
	reservationID, err := uuid.Parse(strings.TrimSpace(request.ReservationID))
	if err != nil || reservationID == uuid.Nil || reservationID.String() != request.ReservationID {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "valid_lpbs_reservation_uuid_required"})
		return
	}
	if strings.TrimSpace(request.ProofReference) == "" || len(request.ProofReference) > 256 || strings.ContainsAny(request.ProofReference, "\r\n") || strings.TrimSpace(request.GrantReceipt) == "" || len(request.GrantReceipt) > 512 || strings.ContainsAny(request.GrantReceipt, "\r\n") {
		writeSimpleError(w, r, "body", "browser settlement proof and grant receipt are required")
		return
	}
	reader, ok := h.svc.RunService.(browserRunReader)
	if !ok {
		writeSimpleError(w, r, "unavailable", "Agent Manager cannot read the authorized run")
		return
	}
	run, err := reader.GetRun(r.Context(), runID)
	if err != nil || run == nil || run.ID != runID || run.Status != domain.RunStatusRunning || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "active_browser_capability_required"})
		return
	}
	capability := run.ResolvedConfig.Features.BrowserCapability
	if !capability.Enabled || browsercapability.Validate(*capability) != nil || browserGrantReceipt(capability) != request.GrantReceipt {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_grant_mismatch"})
		return
	}
	if h.browserEvidenceProof == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "bas_broker_proof_verifier_not_configured"})
		return
	}
	observation, err := h.browserEvidenceProof.VerifyBrowserEvidenceProof(r.Context(), runID, browserGrantReceipt(capability), request.ProofReference)
	if err != nil || observation.Kind != "session" || observation.State != "closed" || strings.TrimSpace(observation.SessionID) == "" || observation.SessionMinutes < 0 || observation.Actions < 0 || observation.Bytes < 0 || observation.SessionMinutes > capability.Budget.SessionMinutes || observation.Actions > capability.Budget.Actions || observation.Bytes > capability.Budget.Bytes {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "closed_broker_session_summary_required"})
		return
	}
	if err := h.browserEvidenceProof.ConsumeBrowserEvidenceProof(r.Context(), runID, request.ProofReference); err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "browser_broker_proof_not_consumed"})
		return
	}
	writeJSON(w, http.StatusOK, browserSettlementAttestationResponse{
		RunID: runID.String(), ReservationID: reservationID.String(),
		GrantReceiptFingerprint: browserEvidenceValueFingerprint(request.GrantReceipt),
		ProofFingerprint:        browserEvidenceValueFingerprint(request.ProofReference),
		Observation: browserSettlementObservation{
			Kind: observation.Kind, SessionID: observation.SessionID, State: observation.State,
			SessionMinutes: observation.SessionMinutes, Actions: observation.Actions, Bytes: observation.Bytes,
		},
	})
}

func browserEvidenceValueFingerprint(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

// AuthorizeBrowserEvidence asks for or checks a one-run Agent Manager grant.
// It records no evidence. Agent Manager requires a separate BAS broker proof
// before any future trusted evidence ingestion can append or bill measurements.
func (h *Handler) AuthorizeBrowserEvidence(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc.IdentityService == nil || h.svc.RunService == nil {
		writeSimpleError(w, r, "unavailable", "browser evidence authorization is unavailable")
		return
	}
	principal, ok := identity.PrincipalFromContext(r.Context())
	if !ok || !principal.IsVerified() || principal.Kind != identity.ActorService || principal.Subject != orchestration.BrowserEvidenceServiceSubject {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "verified_browser_automation_studio_service_required"})
		return
	}
	if _, err := authn.RequireCapability(r.Context(), "agent-manager:write"); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_evidence_service_write_capability_required"})
		return
	}
	if r.Body != nil {
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1))
		if err != nil || len(body) != 0 {
			writeSimpleError(w, r, "body", "browser evidence authorization accepts no body")
			return
		}
	}
	runID, err := uuid.Parse(strings.TrimSpace(mux.Vars(r)["run_id"]))
	if err != nil || runID == uuid.Nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "valid_run_uuid_required"})
		return
	}
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil || verified.Claims.RunID != runID {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "active_run_identity_must_match_target"})
		return
	}
	if verified.RunStatus != domain.RunStatusRunning {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "run_not_active"})
		return
	}
	reader, ok := h.svc.RunService.(browserRunReader)
	if !ok {
		writeSimpleError(w, r, "unavailable", "Agent Manager cannot read the authorized run")
		return
	}
	run, err := reader.GetRun(r.Context(), runID)
	if err != nil || run == nil || run.Status != domain.RunStatusRunning || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil || browsercapability.Validate(*run.ResolvedConfig.Features.BrowserCapability) != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "active_browser_capability_required"})
		return
	}
	capability := run.ResolvedConfig.Features.BrowserCapability
	if strings.TrimSpace(r.Header.Get(browserGrantReceiptHeader)) != browserGrantReceipt(capability) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_grant_mismatch"})
		return
	}
	target := "agent-manager:run/" + runID.String()
	if _, err := authn.RequireOperation(r.Context(), "agent-manager:write", browserEvidenceAuthorizationOperation, target); err != nil {
		if !errors.Is(err, authn.ErrNeedsApproval) {
			writeServiceAdmissionRefusal(w, http.StatusForbidden, "operation_not_admitted")
			return
		}
		raiser, ok := ApprovalRaiserFromContext(r.Context())
		if !ok {
			writeServiceAdmissionRefusal(w, http.StatusForbidden, "approval_raiser_unavailable")
			return
		}
		verifier, ok := raiser.(interface {
			IsBrowserEvidenceRun(context.Context, uuid.UUID) bool
		})
		if !ok || !verifier.IsBrowserEvidenceRun(r.Context(), runID) {
			writeServiceAdmissionRefusal(w, http.StatusForbidden, "active_browser_capability_required")
			return
		}
		raised, raiseErr := raiser.RaiseAuthorizationApproval(r.Context(), orchestration.AuthorizationApprovalRequest{
			Subject: principal.Subject, Operation: browserEvidenceAuthorizationOperation,
			Target: target, Kind: domain.PendingAuthorizationKindOneTime,
		})
		if errors.Is(raiseErr, orchestration.ErrAuthorizationFlood) {
			writeServiceAdmissionRefusal(w, http.StatusTooManyRequests, "approval_raise_limited")
			return
		}
		if raiseErr != nil || !raised {
			writeServiceAdmissionRefusal(w, http.StatusForbidden, "approval_raise_failed")
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status": "approval_pending", "subject": principal.Subject, "operation": browserEvidenceAuthorizationOperation,
			"target": target, "expires_in_seconds": int64(time.Hour.Seconds()), "evidence_recorded": false,
		})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RecordBASBrowserEvidence is the trusted-ingest boundary. The service grant
// authorizes the operation, while the proof verifier independently resolves
// one opaque BAS broker reference into an active-session observation. No
// measurement or session ID from the request body is trusted. The verifier is
// intentionally unconfigured until BAS and Agent Manager have their reciprocal
// owner-issued service-auth path.
func (h *Handler) RecordBASBrowserEvidence(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc.IdentityService == nil || h.svc.RunService == nil || h.svc.BrowserEvidenceService == nil {
		writeSimpleError(w, r, "unavailable", "BAS browser evidence recording is unavailable")
		return
	}
	principal, ok := identity.PrincipalFromContext(r.Context())
	if !ok || !principal.IsVerified() || principal.Kind != identity.ActorService || principal.Subject != orchestration.BrowserEvidenceServiceSubject {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "verified_browser_automation_studio_service_required"})
		return
	}
	if _, err := authn.RequireCapability(r.Context(), "agent-manager:write"); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_evidence_service_write_capability_required"})
		return
	}
	var proof browserEvidenceProofReference
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&proof) != nil || decoder.Decode(new(any)) != io.EOF || strings.TrimSpace(proof.ProofReference) == "" || len(proof.ProofReference) > 256 || strings.ContainsAny(proof.ProofReference, "\r\n") {
		writeSimpleError(w, r, "body", "a bounded opaque broker proof reference is required")
		return
	}
	runID, err := uuid.Parse(strings.TrimSpace(mux.Vars(r)["run_id"]))
	if err != nil || runID == uuid.Nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "valid_run_uuid_required"})
		return
	}
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil || verified.Claims.RunID != runID {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "active_run_identity_must_match_target"})
		return
	}
	if verified.RunStatus != domain.RunStatusRunning {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "run_not_active"})
		return
	}
	reader, ok := h.svc.RunService.(browserRunReader)
	if !ok {
		writeSimpleError(w, r, "unavailable", "Agent Manager cannot read the authorized run")
		return
	}
	run, err := reader.GetRun(r.Context(), runID)
	if err != nil || run == nil || run.Status != domain.RunStatusRunning || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil || browsercapability.Validate(*run.ResolvedConfig.Features.BrowserCapability) != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "active_browser_capability_required"})
		return
	}
	capability := run.ResolvedConfig.Features.BrowserCapability
	if strings.TrimSpace(r.Header.Get(browserGrantReceiptHeader)) != browserGrantReceipt(capability) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_grant_mismatch"})
		return
	}
	target := "agent-manager:run/" + runID.String()
	if _, err := authn.RequireOperation(r.Context(), "agent-manager:write", browserEvidenceAuthorizationOperation, target); err != nil {
		writeServiceAdmissionRefusal(w, http.StatusForbidden, "operation_not_admitted")
		return
	}
	if h.browserEvidenceProof == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "bas_broker_proof_verifier_not_configured"})
		return
	}
	observation, err := h.browserEvidenceProof.VerifyBrowserEvidenceProof(r.Context(), runID, browserGrantReceipt(capability), strings.TrimSpace(proof.ProofReference))
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_broker_proof_invalid"})
		return
	}
	if observation.Kind != "session" || observation.State != "closed" {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "closed_broker_session_summary_required"})
		return
	}
	request := browserEvidenceRequest{
		Kind: observation.Kind, SessionID: observation.SessionID, Mode: capability.Mode,
		PermissionProfile: capability.Permissions.ProfileID, State: observation.State,
		SessionMinutes: observation.SessionMinutes, Actions: observation.Actions, Bytes: observation.Bytes,
		Action: observation.Action, Outcome: observation.Outcome,
	}
	payload, err := browserEvidencePayload(request, capability, "browser-automation-studio-broker")
	if err != nil {
		writeSimpleError(w, r, "evidence", "BAS broker observation is invalid")
		return
	}
	lock := browserEvidenceWriteLock(runID, request.SessionID)
	lock.Lock()
	defer lock.Unlock()
	duplicate, err := h.validateBrowserEvidenceTransition(r.Context(), runID, request, capability.Budget.Actions, "browser-automation-studio-broker", false)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "browser_session_state_invalid"})
		return
	}
	if duplicate {
		if err := h.browserEvidenceProof.ConsumeBrowserEvidenceProof(r.Context(), runID, strings.TrimSpace(proof.ProofReference)); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "browser_broker_proof_consumption_pending", "recorded": true})
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"recorded": true, "idempotent": true, "run_id": runID.String(), "session_id": request.SessionID, "reported_by": "browser-automation-studio-broker"})
		return
	}
	if err := h.svc.RecordBrowserEvidence(r.Context(), runID, payload); err != nil {
		writeSimpleError(w, r, "evidence", "BAS broker evidence could not be recorded")
		return
	}
	if err := h.browserEvidenceProof.ConsumeBrowserEvidenceProof(r.Context(), runID, strings.TrimSpace(proof.ProofReference)); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "browser_broker_proof_consumption_pending", "recorded": true})
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"recorded": true, "run_id": runID.String(), "session_id": request.SessionID, "reported_by": "browser-automation-studio-broker"})
}

func browserEvidencePayload(req browserEvidenceRequest, capability *browsercapability.Capability, reportedBy string) (eventlog.Payload, error) {
	if capability == nil || len(req.SessionID) == 0 || len(req.SessionID) > 128 || strings.ContainsAny(req.SessionID, "\r\n") {
		return nil, errors.New("invalid browser evidence identifiers")
	}
	switch req.Kind {
	case "session":
		budget := capability.Budget
		if (req.State != "opened" && req.State != "closed") || req.SessionMinutes < 0 || req.Actions < 0 || req.Bytes < 0 ||
			(req.State == "opened" && (req.SessionMinutes != 0 || req.Actions != 0 || req.Bytes != 0)) ||
			(req.State == "closed" && ((budget.SessionMinutes == 0 && req.SessionMinutes > 0) || (budget.Actions == 0 && req.Actions > 0) || (budget.Bytes == 0 && req.Bytes > 0) || (budget.SessionMinutes > 0 && req.SessionMinutes > budget.SessionMinutes) || (budget.Actions > 0 && req.Actions > budget.Actions) || (budget.Bytes > 0 && req.Bytes > budget.Bytes))) {
			return nil, errors.New("invalid browser session evidence")
		}
		return eventlog.BrowserSessionPayload{SessionID: req.SessionID, Mode: string(req.Mode), PermissionProfile: req.PermissionProfile, State: req.State, SessionMinutes: req.SessionMinutes, Actions: req.Actions, Bytes: req.Bytes, ReportedBy: reportedBy}, nil
	case "action":
		if req.SessionMinutes != 0 || req.Actions != 0 || req.Bytes != 0 || req.State != "" || !oneOf(req.Action, "observe", "navigate", "click", "type", "scroll", "screenshot", "download", "upload", "attach", "close") || !oneOf(req.Outcome, "allowed", "denied", "completed", "failed") {
			return nil, errors.New("invalid browser action evidence")
		}
		return eventlog.BrowserActionPayload{SessionID: req.SessionID, Mode: string(req.Mode), PermissionProfile: req.PermissionProfile, Action: req.Action, Outcome: req.Outcome, ReportedBy: reportedBy}, nil
	default:
		return nil, errors.New("kind must be session or action")
	}
}

// ResolveBrowserProfile returns the latest run-bound permission data and
// explicitly lists BAS policy fields that the Agent Manager schema cannot
// represent, so the caller can default-deny rather than infer them.
func (h *Handler) ResolveBrowserProfile(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc.IdentityService == nil || h.svc.RunService == nil {
		writeSimpleError(w, r, "unavailable", "browser profile resolution is unavailable")
		return
	}
	var req resolveBrowserProfileRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&req) != nil || decoder.Decode(new(any)) != io.EOF {
		writeSimpleError(w, r, "body", "invalid browser profile request")
		return
	}
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "run_identity_invalid"})
		return
	}
	if verified.RunStatus != domain.RunStatusRunning {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "run_not_active"})
		return
	}
	runID := verified.Claims.RunID
	reader, ok := h.svc.RunService.(browserRunReader)
	if !ok {
		writeSimpleError(w, r, "unavailable", "Agent Manager cannot read the authorized run")
		return
	}
	run, err := reader.GetRun(r.Context(), runID)
	if err != nil || run == nil || run.ID != runID || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil || browsercapability.Validate(*run.ResolvedConfig.Features.BrowserCapability) != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_capability_not_enabled"})
		return
	}
	capability := run.ResolvedConfig.Features.BrowserCapability
	receipt := browserGrantReceipt(run.ResolvedConfig.Features.BrowserCapability)
	if strings.TrimSpace(r.Header.Get(browserGrantReceiptHeader)) != receipt || strings.TrimSpace(req.ProfileRef) != capability.Permissions.ProfileID {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_profile_or_grant_mismatch"})
		return
	}
	profileMode, hasProfileMode := sessionPermissionProfileMode(capability.Permissions.ProfileMode)
	unsupportedFields := []string{"URLDeny", "Approval", "ActionSpecificApproval", "HistoryExport", "UserStateExport", "HistoryRetention", "Handoff", "MaxCredits"}
	if !hasProfileMode {
		unsupportedFields = append(unsupportedFields, "ProfileMode")
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "active", "run_id": runID.String(), "grant_receipt": receipt, "mode": capability.Mode,
		"permission_profile": capability.Permissions, "budget": capability.Budget,
		"session_permission_mapping":           map[string]any{"view": capability.Permissions.View, "control": capability.Permissions.Control, "profile_mode": profileMode, "extension_access": len(capability.Permissions.ExtensionIDs) > 0, "extension_permissions": emptyExtensionPermissions(capability.Permissions.ExtensionIDs)},
		"unsupported_sessionpermission_fields": unsupportedFields,
	})
}

func sessionPermissionProfileMode(mode browsercapability.SessionProfileMode) (string, bool) {
	switch mode {
	case browsercapability.SessionProfileFresh, browsercapability.SessionProfileIsolated,
		browsercapability.SessionProfileUserAttached, browsercapability.SessionProfileRemote:
		return string(mode), true
	default:
		return "", false
	}
}

func emptyExtensionPermissions(ids []string) map[string][]string {
	permissions := make(map[string][]string, len(ids))
	for _, id := range ids {
		permissions[id] = []string{}
	}
	return permissions
}

// RecordBrowserEvidence accepts bounded, run-reported lifecycle/action metadata
// for transcript continuity only. It deliberately remains separate from the
// BAS service-authorized evidence path: records are labeled run-agent, untrusted,
// and cannot support browser qualification or billing. No caller, owner, URL,
// page content or typed values are accepted from JSON.
func (h *Handler) RecordBrowserEvidence(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc.IdentityService == nil || h.svc.RunService == nil || h.svc.BrowserEvidenceService == nil {
		writeSimpleError(w, r, "unavailable", "browser evidence recording is unavailable")
		return
	}
	var req browserEvidenceRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&req) != nil || decoder.Decode(new(any)) != io.EOF {
		writeSimpleError(w, r, "body", "invalid browser evidence request")
		return
	}
	if len(req.SessionID) == 0 || len(req.SessionID) > 128 || strings.ContainsAny(req.SessionID, "\r\n") || len(req.PermissionProfile) > 128 {
		writeSimpleError(w, r, "body", "invalid browser evidence identifiers")
		return
	}
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "run_identity_invalid"})
		return
	}
	if verified.RunStatus != domain.RunStatusRunning {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "run_not_active"})
		return
	}
	runID := verified.Claims.RunID
	principal, ok := identity.PrincipalFromContext(r.Context())
	if !ok || !principal.IsVerified() || principal.Kind != identity.ActorAgent || principal.Subject != runID.String() {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "matching_verified_run_identity_required"})
		return
	}
	if _, err := authn.RequireCapability(r.Context(), "agent-manager:write"); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "run_write_capability_required"})
		return
	}
	reader, ok := h.svc.RunService.(browserRunReader)
	if !ok {
		writeSimpleError(w, r, "unavailable", "Agent Manager cannot read the authorized run")
		return
	}
	run, err := reader.GetRun(r.Context(), runID)
	if err != nil || run == nil || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_capability_not_enabled"})
		return
	}
	capability := run.ResolvedConfig.Features.BrowserCapability
	if strings.TrimSpace(r.Header.Get(browserGrantReceiptHeader)) != browserGrantReceipt(capability) || req.Mode != capability.Mode || req.PermissionProfile != capability.Permissions.ProfileID {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_grant_or_profile_mismatch"})
		return
	}
	var payload eventlog.Payload
	switch req.Kind {
	case "session":
		budget := capability.Budget
		if (req.State != "opened" && req.State != "closed") || req.SessionMinutes < 0 || req.Actions < 0 || req.Bytes < 0 ||
			(req.State == "opened" && (req.SessionMinutes != 0 || req.Actions != 0 || req.Bytes != 0)) ||
			(req.State == "closed" && ((budget.SessionMinutes == 0 && req.SessionMinutes > 0) || (budget.Actions == 0 && req.Actions > 0) || (budget.Bytes == 0 && req.Bytes > 0) || (budget.SessionMinutes > 0 && req.SessionMinutes > budget.SessionMinutes) || (budget.Actions > 0 && req.Actions > budget.Actions) || (budget.Bytes > 0 && req.Bytes > budget.Bytes))) {
			writeSimpleError(w, r, "body", "invalid browser session evidence")
			return
		}
		payload = eventlog.BrowserSessionPayload{SessionID: req.SessionID, Mode: string(req.Mode), PermissionProfile: req.PermissionProfile, State: req.State, SessionMinutes: req.SessionMinutes, Actions: req.Actions, Bytes: req.Bytes, ReportedBy: "run-agent"}
	case "action":
		if req.SessionMinutes != 0 || req.Actions != 0 || req.Bytes != 0 || req.State != "" || !oneOf(req.Action, "observe", "navigate", "click", "type", "scroll", "screenshot", "download", "upload", "attach", "close") || !oneOf(req.Outcome, "allowed", "denied", "completed", "failed") {
			writeSimpleError(w, r, "body", "invalid browser action evidence")
			return
		}
		payload = eventlog.BrowserActionPayload{SessionID: req.SessionID, Mode: string(req.Mode), PermissionProfile: req.PermissionProfile, Action: req.Action, Outcome: req.Outcome, ReportedBy: "run-agent"}
	default:
		writeSimpleError(w, r, "body", "kind must be session or action")
		return
	}
	lock := browserEvidenceWriteLock(runID, req.SessionID)
	lock.Lock()
	defer lock.Unlock()
	duplicate, err := h.validateBrowserEvidenceTransition(r.Context(), runID, req, capability.Budget.Actions, "run-agent", true)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "browser_session_state_invalid", "message": err.Error()})
		return
	}
	if duplicate {
		writeJSON(w, http.StatusAccepted, map[string]any{"recorded": true, "idempotent": true, "run_id": runID.String(), "session_id": req.SessionID})
		return
	}
	if err := h.svc.RecordBrowserEvidence(r.Context(), runID, payload); err != nil {
		writeSimpleError(w, r, "evidence", "browser evidence could not be recorded")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"recorded": true, "run_id": runID.String(), "session_id": req.SessionID})
}

func browserEvidenceWriteLock(runID uuid.UUID, sessionID string) *sync.Mutex {
	hash := fnv.New32a()
	_, _ = hash.Write(runID[:])
	_, _ = hash.Write([]byte{0})
	_, _ = hash.Write([]byte(sessionID))
	return &browserEvidenceWriteLocks[hash.Sum32()%uint32(len(browserEvidenceWriteLocks))]
}

func (h *Handler) validateBrowserEvidenceTransition(ctx context.Context, runID uuid.UUID, req browserEvidenceRequest, maxActions int64, reporter string, requireOpen bool) (bool, error) {
	if h.svc.EventService == nil {
		return false, errors.New("run transcript is unavailable")
	}
	events, err := h.svc.GetRunEvents(ctx, runID, eventstore.GetOptions{EventTypes: []domain.RunEventType{domain.EventTypeBrowserSession, domain.EventTypeBrowserAction}})
	if err != nil {
		return false, err
	}
	opened, closed := false, false
	var actionCount int64
	var existingClose *eventlog.BrowserSessionPayload
	for _, evt := range events {
		if evt == nil || evt.Data == nil {
			continue
		}
		typed, ok := evt.Data.(*domain.TypedEventData)
		if !ok {
			if evt.EventType == domain.EventTypeBrowserSession || evt.EventType == domain.EventTypeBrowserAction {
				return false, errors.New("browser transcript contains an invalid typed event")
			}
			continue
		}
		decoded, decodeErr := eventlog.Decode(evt.EventType, evt.SchemaVersion, typed.Body)
		if decodeErr != nil {
			if evt.EventType == domain.EventTypeBrowserSession || evt.EventType == domain.EventTypeBrowserAction {
				return false, decodeErr
			}
			continue
		}
		switch p := decoded.(type) {
		case *eventlog.BrowserSessionPayload:
			if p.SessionID != req.SessionID || p.ReportedBy != reporter {
				continue
			}
			if p.State == "opened" {
				opened = true
			}
			if p.State == "closed" {
				closed = true
				clone := *p
				existingClose = &clone
			}
		case *eventlog.BrowserActionPayload:
			if p.SessionID == req.SessionID && p.ReportedBy == reporter {
				actionCount++
				if !opened {
					return false, errors.New("action evidence precedes session open")
				}
			}
		}
	}
	if req.Kind == "session" {
		if req.State == "opened" {
			if opened || closed {
				return false, errors.New("session ID already has a lifecycle")
			}
			return false, nil
		}
		if !opened && requireOpen {
			return false, errors.New("session close requires a recorded open")
		}
		if req.Actions < actionCount {
			return false, errors.New("reported action count is below recorded action events")
		}
		if closed {
			if existingClose != nil && existingClose.Mode == string(req.Mode) && existingClose.PermissionProfile == req.PermissionProfile && existingClose.SessionMinutes == req.SessionMinutes && existingClose.Actions == req.Actions && existingClose.Bytes == req.Bytes {
				return true, nil
			}
			return false, errors.New("session ID is already closed with different counters")
		}
		return false, nil
	}
	if !opened || closed {
		return false, errors.New("action evidence requires an open session")
	}
	if maxActions > 0 && actionCount >= maxActions {
		return false, errors.New("browser action budget is exhausted")
	}
	return false, nil
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

// ResolveBrowserCaller maps the authenticated active run identity to the
// stable browser principal consumed by BAS. The principal is run-scoped so
// delegated runs cannot impersonate their parent; BAS separately owns the
// browser session owner and resolves the current caller on every operation.
func (h *Handler) ResolveBrowserCaller(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc.IdentityService == nil || h.svc.RunService == nil {
		writeSimpleError(w, r, "unavailable", "browser caller resolution is unavailable")
		return
	}
	token := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "run_identity_required"})
		return
	}
	verified, err := h.svc.VerifyIdentityToken(r.Context(), token)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "run_identity_invalid"})
		return
	}
	if verified.RunStatus != domain.RunStatusRunning {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "run_not_active"})
		return
	}
	reader, ok := h.svc.RunService.(browserRunReader)
	if !ok {
		writeSimpleError(w, r, "unavailable", "Agent Manager cannot read the authorized run")
		return
	}
	run, err := reader.GetRun(r.Context(), verified.Claims.RunID)
	if err != nil || run == nil || run.ID != verified.Claims.RunID || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_capability_not_enabled"})
		return
	}
	capability := run.ResolvedConfig.Features.BrowserCapability
	receipt := browserGrantReceipt(capability)
	if !capability.Enabled || receipt == "" {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_grant_not_available"})
		return
	}
	if strings.TrimSpace(r.Header.Get(browserGrantReceiptHeader)) != receipt {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "browser_grant_mismatch"})
		return
	}
	writeJSON(w, http.StatusOK, resolveBrowserCallerResponse{
		CallerID: "run:" + verified.Claims.RunID.String(), RunID: verified.Claims.RunID.String(),
		GrantReceipt: receipt, ProfileID: strings.TrimSpace(capability.Permissions.ProfileID),
	})
}

// AuthorizeBrowserOpen is the run-scoped Agent Manager decision called by BAS
// before it creates a managed, installed, attached or remote browser session.
// The caller identity comes only from the verified X-Agent-Identity-Token;
// request JSON has no run, owner, caller identity, or grant receipt field.
func (h *Handler) AuthorizeBrowserOpen(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.svc.IdentityService == nil || h.svc.RunService == nil {
		writeSimpleError(w, r, "unavailable", "browser capability authorization is unavailable")
		return
	}
	var req authorizeBrowserOpenRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeSimpleError(w, r, "body", "invalid browser open authorization request")
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeSimpleError(w, r, "body", "browser open authorization request must contain one JSON object")
		return
	}
	identityToken := strings.TrimSpace(r.Header.Get(cliutil.HeaderAgentIdentityToken))
	if identityToken == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"authorized": false, "error": "run_identity_required"})
		return
	}
	verified, err := h.svc.VerifyIdentityToken(r.Context(), identityToken)
	if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]any{"authorized": false, "error": "run_identity_invalid"})
		return
	}
	if verified.RunStatus != domain.RunStatusRunning {
		writeJSON(w, http.StatusForbidden, map[string]any{"authorized": false, "error": "run_not_active"})
		return
	}
	reader, ok := h.svc.RunService.(browserRunReader)
	if !ok {
		writeSimpleError(w, r, "unavailable", "Agent Manager cannot read the authorized run")
		return
	}
	runID := verified.Claims.RunID
	run, err := reader.GetRun(r.Context(), runID)
	if err != nil || run == nil || run.ID != runID || run.ResolvedConfig == nil || run.ResolvedConfig.Features.BrowserCapability == nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"authorized": false, "error": "browser_capability_not_enabled"})
		return
	}
	capability := run.ResolvedConfig.Features.BrowserCapability.Clone()
	selectedRole, ok := selectBrowserResourceRole(capability, string(run.ResolvedConfig.RunnerType), req.ResourceRole)
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]any{"authorized": false, "error": "no_eligible_browser_candidate"})
		return
	}
	grantReceipt := browserGrantReceipt(&capability)
	presentedGrantReceipt := strings.TrimSpace(r.Header.Get(browserGrantReceiptHeader))
	if presentedGrantReceipt != grantReceipt {
		writeJSON(w, http.StatusForbidden, map[string]any{"authorized": false, "error": "browser_grant_mismatch"})
		return
	}
	if err := browsercapability.ValidateOpenCapability(capability, browsercapability.OpenRequest{
		GrantReceipt: presentedGrantReceipt, Mode: req.Mode, ProfileID: req.ProfileID, InitialURL: req.InitialURL,
	}); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]any{"authorized": false, "error": "browser_open_not_permitted", "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, authorizeBrowserOpenResponse{
		Authorized: true, RunID: runID.String(), GrantReceipt: grantReceipt, Harness: string(run.ResolvedConfig.RunnerType), ResourceRole: selectedRole, Capability: capability,
	})
}

func selectBrowserResourceRole(capability browsercapability.Capability, harness, requested string) (string, bool) {
	if !capability.Enabled || strings.TrimSpace(harness) == "" {
		return "", false
	}
	var eligible []string
	for _, candidate := range capability.Candidates {
		if strings.EqualFold(strings.TrimSpace(candidate.Harness), strings.TrimSpace(harness)) && strings.TrimSpace(candidate.ResourceRole) != "" {
			eligible = append(eligible, strings.TrimSpace(candidate.ResourceRole))
		}
	}
	requested = strings.TrimSpace(requested)
	if requested != "" {
		for _, role := range eligible {
			if role == requested {
				return role, true
			}
		}
		return "", false
	}
	if len(eligible) == 1 {
		return eligible[0], true
	}
	return "", false
}

// browserGrantReceipt is the stable reference injected by Agent Manager for a
// run with an enabled browser capability. The signed run identity remains the
// authentication credential; this value is only an object-bound grant label.
func browserGrantReceipt(capability *browsercapability.Capability) string {
	if capability == nil {
		return ""
	}
	return strings.TrimSpace(capability.GrantReceipt)
}
