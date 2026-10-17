package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	apierrors "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/errors"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/remediation"
)

func (h *Handlers) Incidents(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseIncidentFilters(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	resp, err := h.store.ListIncidents(ctx, filters)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incidents", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		apierrors.LogError("incidents", "encode_response", err)
	}
}

func (h *Handlers) LatestIncidents(w http.ResponseWriter, r *http.Request) {
	r.URL.RawQuery = mergeDefaultQuery(r.URL.RawQuery, "status=open")
	h.Incidents(w, r)
}

func (h *Handlers) IncidentDetail(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	incident, err := h.store.GetIncident(ctx, id)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incident", err))
		return
	}
	if incident == nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("incidents", "incident", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(incident); err != nil {
		apierrors.LogError("incidents", "encode_detail_response", err)
	}
}

func (h *Handlers) IncidentObservations(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	limit := 50
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	observations, err := h.store.ListIncidentObservations(ctx, id, limit)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incident observations", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"observations": observations, "total": len(observations)}); err != nil {
		apierrors.LogError("incidents", "encode_observations_response", err)
	}
}

func (h *Handlers) IncidentRemediations(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	incident, err := h.store.GetIncident(ctx, id)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incident", err))
		return
	}
	if incident == nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("incidents", "incident", id))
		return
	}
	candidates := incident.RemediationCandidates
	if h.remediationService != nil {
		candidates = h.remediationService.Candidates(*incident)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"incidentId": id, "remediations": candidates, "total": len(candidates)}); err != nil {
		apierrors.LogError("incidents", "encode_remediations_response", err)
	}
}

func (h *Handlers) GenerateIncidentRemediation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	remediationID := mux.Vars(r)["remediationId"]
	if h.remediationService == nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "remediation service unavailable", fmt.Errorf("remediation service unavailable")))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	incident, err := h.store.GetIncident(ctx, id)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incident", err))
		return
	}
	if incident == nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("incidents", "incident", id))
		return
	}
	resp, err := h.remediationService.Generate(*incident, remediationID)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "generate remediation", err))
		return
	}
	if _, err := h.store.RecordIncidentRemediationArtifact(ctx, id, resp.Artifact); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "record remediation artifact", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		apierrors.LogError("incidents", "encode_remediation_generation_response", err)
	}
}

func (h *Handlers) RecordIncidentRemediationOutcome(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	remediationID := mux.Vars(r)["remediationId"]
	var req remediation.OutcomeRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "decode remediation outcome", err))
			return
		}
	}
	if h.remediationService == nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "remediation service unavailable", fmt.Errorf("remediation service unavailable")))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	incident, err := h.store.GetIncident(ctx, id)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incident", err))
		return
	}
	if incident == nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("incidents", "incident", id))
		return
	}
	outcome, err := h.remediationService.Outcome(*incident, remediationID, req)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "record remediation outcome", err))
		return
	}
	updated, err := h.store.RecordIncidentRemediationOutcome(ctx, id, outcome)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "record remediation outcome", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		apierrors.LogError("incidents", "encode_remediation_outcome_response", err)
	}
}

// ErrRemediationAlreadyApplied is returned when the single-use authorisation was
// already consumed — a replayed/again-claimed resolution. The event consumer
// treats it as a benign duplicate (acks it) rather than letting vrooli-events
// retry a poison message; the manual REST route surfaces it as a validation
// error. It is distinct from a genuine reject so the two never conflate.
var ErrRemediationAlreadyApplied = errors.New("remediation authorization was already consumed")

// remediationAuthorisationStore is the single-use authorisation record both the
// manual REST route and the event consumer claim through. Keeping it an inline
// interface (rather than widening StoreInterface) lets the in-memory test store
// opt in without every other handler depending on it.
type remediationAuthorisationStore interface {
	RecordRemediationAuthorisation(context.Context, string, string, string, string, string, time.Time) error
	ClaimRemediationAuthorisation(context.Context, string, string, string, string, time.Time) (bool, error)
}

// remediationApprovalInput is the verified-ask-to-incident binding the approval
// service method acts on. AskID, IncidentID, IncidentFingerprint, and CandidateID
// are the four correlation keys the ask carried; the service re-verifies the
// first and re-matches the rest against the live incident at apply time (the
// TOCTOU re-check), never trusting the caller's copy.
//
// RiskGated is the difference between the two callers. The event consumer sets
// it true: an approve auto-executes ONLY a low-risk candidate and otherwise
// records the approval and defers. The manual REST route sets it false: it IS
// the explicit operator apply step, so it executes the (already approved and
// authorised) candidate regardless of risk level — that is what makes it the
// fallback for a high-risk candidate the auto-apply path deferred.
type remediationApprovalInput struct {
	IncidentID          string
	CandidateID         string
	AskID               string
	IncidentFingerprint string
	RiskGated           bool
}

// remediationApprovalOutcome reports what the shared approval path did. Applied
// is true only when a low-risk candidate auto-executed; for a high-risk or
// privileged candidate the approval is recorded and Applied stays false with
// AutoApplyDeferred set, so the caller knows the operator must still trigger the
// explicit apply step.
type remediationApprovalOutcome struct {
	Applied           bool                         `json:"applied"`
	AutoApplyDeferred bool                         `json:"autoApplyDeferred,omitempty"`
	DeferralReason    string                       `json:"deferralReason,omitempty"`
	Execution         *remediation.ExecutionResult `json:"execution,omitempty"`
}

// autoApplyEligible is the low-risk predicate that gates server-side auto-apply.
// Only a candidate declared low risk AND requiring neither privilege nor an
// operator-in-the-loop may execute on an approve. Every host/privileged/
// operator-required candidate records the approval but defers execution to the
// explicit manual apply step (notify-then-manual). The default-deny shape means
// a candidate with an unset/unknown risk level is treated as high risk.
func autoApplyEligible(candidate incidents.RemediationCandidate) bool {
	return strings.EqualFold(strings.TrimSpace(candidate.RiskLevel), "low") &&
		!candidate.RequiresPrivilege && !candidate.RequiresOperator
}

// ApproveRemediationFromDecision is the ONE path that turns an approved operator
// decision into (at most) an executed remediation. Both the manual /approve REST
// route and the notification-hub decision event consumer call it, so the four
// gates and the risk gate never drift between them. In order it:
//  1. verifies the ask against notification-hub (never trusts a caller-asserted
//     approval),
//  2. re-matches the incident fingerprint (TOCTOU staleness re-check at apply
//     time: the incident is re-read here, not at raise time),
//  3. checks autoheal-enabled for the incident's originating check,
//  4. records the single-use authorisation, then claims it (double-spend guard),
//  5. risk-gates: only a low-risk candidate auto-executes; a high-risk one
//     records the approval and returns with AutoApplyDeferred set,
//  6. executes (low-risk only) and records the execution outcome.
func (h *Handlers) ApproveRemediationFromDecision(ctx context.Context, in remediationApprovalInput) (remediationApprovalOutcome, error) {
	if h.remediationService == nil {
		return remediationApprovalOutcome{}, fmt.Errorf("remediation service unavailable")
	}
	if h.remediationAskVerifier == nil {
		return remediationApprovalOutcome{}, remediation.ErrAskVerifierUnavailable
	}
	incident, err := h.store.GetIncident(ctx, in.IncidentID)
	if err != nil {
		return remediationApprovalOutcome{}, fmt.Errorf("retrieve incident: %w", err)
	}
	if incident == nil {
		return remediationApprovalOutcome{}, fmt.Errorf("incident %q not found", in.IncidentID)
	}
	// Gate 2: the ask must be answered-approve in the durable hub store.
	approval, err := h.remediationAskVerifier.Verify(ctx, in.AskID)
	if err != nil {
		return remediationApprovalOutcome{}, fmt.Errorf("notification ask is not approved: %w", err)
	}
	if approval.AskID == "" {
		approval.AskID = in.AskID
	}
	// Gate 3 (TOCTOU): the fingerprint the ask was raised against must still be
	// the incident's current fingerprint. A superseded incident fails here.
	if in.IncidentFingerprint != incident.Fingerprint {
		return remediationApprovalOutcome{}, fmt.Errorf("incident fingerprint does not match: %w", remediation.ErrAskNotApproved)
	}
	// Gate 1: autoheal must be enabled for the originating check.
	autoHealEnabled := h.registry.IsAutoHealEnabled(firstIncidentCheckID(*incident))
	if !autoHealEnabled {
		return remediationApprovalOutcome{}, remediation.ErrAutoHealDisabled
	}
	authorisations, ok := h.store.(remediationAuthorisationStore)
	if !ok {
		return remediationApprovalOutcome{}, fmt.Errorf("remediation authorization storage unavailable")
	}
	if err := authorisations.RecordRemediationAuthorisation(ctx, approval.AskID, incident.ID, incident.Fingerprint, in.CandidateID, approval.Actor, time.Now().UTC()); err != nil {
		return remediationApprovalOutcome{}, fmt.Errorf("record remediation authorization: %w", err)
	}
	// Gate 4: single-use claim. A replay of an already-consumed authorisation is
	// refused here, before any risk decision or execution.
	claimed, err := authorisations.ClaimRemediationAuthorisation(ctx, approval.AskID, incident.ID, incident.Fingerprint, in.CandidateID, time.Now().UTC())
	if err != nil {
		return remediationApprovalOutcome{}, fmt.Errorf("claim remediation authorization: %w", err)
	}
	if !claimed {
		return remediationApprovalOutcome{}, ErrRemediationAlreadyApplied
	}

	// Risk gate (event-consumer path only): an approval of a privileged/high-risk
	// candidate is recorded (the authorisation above stands and is claimed) but
	// does NOT auto-execute. The operator must still trigger the explicit manual
	// apply step. Only a low-risk candidate crosses into execution on an approve.
	// The manual REST route sets RiskGated=false because it IS that explicit
	// apply step.
	candidate, ok := findIncidentCandidate(*incident, in.CandidateID)
	if !ok {
		return remediationApprovalOutcome{}, fmt.Errorf("remediation candidate %q not found", in.CandidateID)
	}
	if in.RiskGated && !autoApplyEligible(candidate) {
		reason := fmt.Sprintf("candidate risk level %q requires an explicit operator apply step", candidate.RiskLevel)
		return remediationApprovalOutcome{AutoApplyDeferred: true, DeferralReason: reason}, nil
	}

	result, err := h.remediationService.Execute(ctx, *incident, in.CandidateID, remediation.Authorisation{
		AskID: approval.AskID, IncidentID: incident.ID, IncidentFingerprint: incident.Fingerprint,
		CandidateID: in.CandidateID, Approved: remediation.ApprovedAsk(approval), AutoHealEnabled: autoHealEnabled,
	})
	if err != nil {
		return remediationApprovalOutcome{}, fmt.Errorf("execute approved remediation: %w", err)
	}
	status := "executed"
	if !result.Success {
		status = "failed"
	}
	if _, err := h.store.RecordIncidentRemediationOutcome(ctx, incident.ID, incidents.Outcome{
		RemediationID: in.CandidateID, Status: status, ReportedAt: time.Now().UTC(), AskID: approval.AskID,
		IncidentFingerprint: incident.Fingerprint, ScriptPath: result.ScriptPath, ExitStatus: result.ExitStatus, Output: result.Output,
	}); err != nil {
		return remediationApprovalOutcome{}, fmt.Errorf("record remediation execution: %w", err)
	}
	return remediationApprovalOutcome{Applied: true, Execution: &result}, nil
}

// recordDeclinedRemediation records a DECLINED outcome WITHOUT executing, for a
// reject/no-answer resolution. It never touches the authorisation store, so a
// later genuine approve is unaffected.
func (h *Handlers) recordDeclinedRemediation(ctx context.Context, incidentID, candidateID, askID, fingerprint, reason string) error {
	if h.store == nil {
		return fmt.Errorf("remediation outcome storage unavailable")
	}
	_, err := h.store.RecordIncidentRemediationOutcome(ctx, incidentID, incidents.Outcome{
		RemediationID: candidateID, Status: "declined", ReportedAt: time.Now().UTC(), AskID: askID,
		IncidentFingerprint: fingerprint, Output: reason,
	})
	return err
}

func findIncidentCandidate(incident incidents.Incident, candidateID string) (incidents.RemediationCandidate, bool) {
	for _, candidate := range incident.RemediationCandidates {
		if candidate.ID == candidateID {
			return candidate, true
		}
	}
	return incidents.RemediationCandidate{}, false
}

// ApproveIncidentRemediation is the manual REST route, kept as the risk-gated
// fallback an operator uses to apply a high-risk candidate the auto-apply path
// deliberately deferred. It is a thin adapter over ApproveRemediationFromDecision:
// the logic lives in the one shared service method. Do NOT retire this route in
// P8 — it is the documented manual apply step for privileged candidates.
func (h *Handlers) ApproveIncidentRemediation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	remediationID := mux.Vars(r)["remediationId"]
	var req struct {
		AskID               string `json:"askId"`
		IncidentFingerprint string `json:"incidentFingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "decode remediation approval", err))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	outcome, err := h.ApproveRemediationFromDecision(ctx, remediationApprovalInput{
		IncidentID:          id,
		CandidateID:         remediationID,
		AskID:               req.AskID,
		IncidentFingerprint: req.IncidentFingerprint,
	})
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "execute approved remediation", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// The manual route keeps returning the execution result verbatim when it
	// executed (back-compatible with the existing client), and the deferral
	// envelope when the risk gate declined to auto-apply.
	if outcome.Applied && outcome.Execution != nil {
		_ = json.NewEncoder(w).Encode(outcome.Execution)
		return
	}
	_ = json.NewEncoder(w).Encode(outcome)
}

func firstIncidentCheckID(incident incidents.Incident) string {
	if len(incident.SourceCheckIDs) == 0 {
		return ""
	}
	return incident.SourceCheckIDs[0]
}

func (h *Handlers) MutateIncidentStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	action := mux.Vars(r)["action"]
	var status incidents.Status
	switch action {
	case "acknowledge":
		status = incidents.StatusAcknowledged
	case "resolve":
		status = incidents.StatusResolved
	case "ignore":
		status = incidents.StatusIgnored
	case "keep-open":
		status = incidents.StatusOpen
	default:
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "invalid incident status action", fmt.Errorf("unsupported action %q", action)))
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	if action == "keep-open" && strings.TrimSpace(body.Note) == "" {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "keep-open requires the next action and owner", fmt.Errorf("disposition note is required")))
		return
	}
	var incident *incidents.Incident
	var err error
	if h.incidentService != nil {
		incident, err = h.incidentService.UpdateIncidentStatus(ctx, id, status, body.Note)
	} else {
		incident, err = h.store.UpdateIncidentStatus(ctx, id, status, body.Note)
	}
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "update incident status", err))
		return
	}
	if incident == nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("incidents", "incident", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(incident); err != nil {
		apierrors.LogError("incidents", "encode_mutation_response", err)
	}
}

func parseIncidentFilters(w http.ResponseWriter, r *http.Request) (incidents.ListFilters, bool) {
	query := r.URL.Query()
	filters := incidents.ListFilters{Limit: 50}
	if status := query.Get("status"); status != "" {
		if !incidents.ValidStatus(status) {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "invalid status filter", fmt.Errorf("invalid status %q", status)))
			return filters, false
		}
		filters.Status = incidents.Status(status)
	}
	if severity := query.Get("severity"); severity != "" {
		if !incidents.ValidSeverity(severity) {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "invalid severity filter", fmt.Errorf("invalid severity %q", severity)))
			return filters, false
		}
		filters.Severity = incidents.Severity(severity)
	}
	if typ := query.Get("type"); typ != "" {
		if !incidents.ValidType(typ) {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "invalid type filter", fmt.Errorf("invalid type %q", typ)))
			return filters, false
		}
		filters.Type = incidents.Type(typ)
	}
	if limitStr := query.Get("limit"); limitStr != "" {
		if parsed, err := parsePositiveInt(limitStr); err == nil && parsed > 0 && parsed <= 200 {
			filters.Limit = parsed
		}
	}
	return filters, true
}

func mergeDefaultQuery(rawQuery, defaults string) string {
	if rawQuery == "" {
		return defaults
	}
	return rawQuery + "&" + defaults
}

// Watchdog returns the OS-level watchdog/service status
// [REQ:WATCH-DETECT-001]
