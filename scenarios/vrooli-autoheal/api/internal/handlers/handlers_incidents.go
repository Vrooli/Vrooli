package handlers

import (
	"context"
	"encoding/json"
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

// ApproveIncidentRemediation executes only a previously generated artifact.
// The check policy and the ask response are independent gates. This handler
// does not elevate privileges; privileged scripts return their own operator
// action requirement instead.
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
	if h.remediationService == nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "remediation service unavailable", fmt.Errorf("remediation service unavailable")))
		return
	}
	if h.remediationAskVerifier == nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "notification ask verifier unavailable", remediation.ErrAskVerifierUnavailable))
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
	approval, err := h.remediationAskVerifier.Verify(ctx, req.AskID)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "notification ask is not approved", err))
		return
	}
	if approval.AskID == "" {
		approval.AskID = req.AskID
	}
	if req.IncidentFingerprint != incident.Fingerprint {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "incident fingerprint does not match", remediation.ErrAskNotApproved))
		return
	}
	autoHealEnabled := h.registry.IsAutoHealEnabled(firstIncidentCheckID(*incident))
	if !autoHealEnabled {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "execute approved remediation", remediation.ErrAutoHealDisabled))
		return
	}
	authorisations, ok := h.store.(interface {
		RecordRemediationAuthorisation(context.Context, string, string, string, string, string, time.Time) error
		ClaimRemediationAuthorisation(context.Context, string, string, string, string, time.Time) (bool, error)
	})
	if !ok {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "remediation authorization storage unavailable", fmt.Errorf("remediation authorization storage unavailable")))
		return
	}
	if err := authorisations.RecordRemediationAuthorisation(ctx, approval.AskID, incident.ID, incident.Fingerprint, remediationID, approval.Actor, time.Now().UTC()); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "record remediation authorization", err))
		return
	}
	claimed, err := authorisations.ClaimRemediationAuthorisation(ctx, approval.AskID, incident.ID, incident.Fingerprint, remediationID, time.Now().UTC())
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "claim remediation authorization", err))
		return
	}
	if !claimed {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "remediation authorization was already consumed", remediation.ErrAskNotApproved))
		return
	}
	result, err := h.remediationService.Execute(ctx, *incident, remediationID, remediation.Authorisation{
		AskID: approval.AskID, IncidentID: incident.ID, IncidentFingerprint: req.IncidentFingerprint,
		CandidateID: remediationID, Approved: remediation.ApprovedAsk(approval), AutoHealEnabled: autoHealEnabled,
	})
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "execute approved remediation", err))
		return
	}
	status := "executed"
	if !result.Success {
		status = "failed"
	}
	_, err = h.store.RecordIncidentRemediationOutcome(ctx, id, incidents.Outcome{
		RemediationID: remediationID, Status: status, ReportedAt: time.Now().UTC(), AskID: approval.AskID,
		IncidentFingerprint: incident.Fingerprint, ScriptPath: result.ScriptPath, ExitStatus: result.ExitStatus, Output: result.Output,
	})
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "record remediation execution", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
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
