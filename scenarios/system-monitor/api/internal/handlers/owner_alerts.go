package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/agentmanager"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

const ownerAlertListLimit = 1000

var attachedSessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

type attachedIdentityVerifier interface {
	VerifyAttachedIdentity(context.Context, string) (*agentmanager.AttachedIdentity, error)
}

// OwnerAlertsHandler exposes only the caller's attached-session alerts. The
// identity token is verified by Agent Manager, including persisted session
// binding; an owner cannot acknowledge another session's alert.
type OwnerAlertsHandler struct {
	source     AlertSource
	acks       AlertAcknowledger
	identities attachedIdentityVerifier
}

func NewOwnerAlertsHandler(source AlertSource, acks AlertAcknowledger, identities attachedIdentityVerifier) *OwnerAlertsHandler {
	return &OwnerAlertsHandler{source: source, acks: acks, identities: identities}
}

// Register mounts the owner-scoped read and acknowledgement routes.
func (h *OwnerAlertsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/owner/alerts", h.List)
	mux.HandleFunc("POST /api/v1/owner/alerts/{id}/acknowledge", h.Acknowledge)
}

type ownerAlertListResponse struct {
	OwnerID string          `json:"owner_id"`
	Alerts  []*models.Alert `json:"alerts"`
	// Complete is false when the shared active-alert list hit its bound before
	// owner filtering. The handler never presents a truncated view as complete.
	Complete bool `json:"complete"`
}

func (h *OwnerAlertsHandler) List(w http.ResponseWriter, r *http.Request) {
	identity, status := h.verifyRequestIdentity(r)
	if status != 0 {
		writeOwnerAlertError(w, status)
		return
	}
	if h.source == nil {
		writeOwnerAlertError(w, http.StatusServiceUnavailable)
		return
	}
	alerts, err := h.source.ListAlerts(r.Context(), services.AlertQuery{Status: "active", Limit: ownerAlertListLimit})
	if err != nil {
		writeOwnerAlertError(w, http.StatusInternalServerError)
		return
	}
	owned := make([]*models.Alert, 0)
	for _, alert := range alerts {
		if ownerAlertOwnerID(alert) == identity.OwnerID {
			owned = append(owned, alert)
		}
	}
	writeOwnerJSON(w, http.StatusOK, ownerAlertListResponse{
		OwnerID: identity.OwnerID, Alerts: owned, Complete: len(alerts) < ownerAlertListLimit,
	})
}

type ownerAlertAckResponse struct {
	AlertID           string `json:"alert_id"`
	OwnerID           string `json:"owner_id"`
	AcknowledgedByRun string `json:"acknowledged_by_run"`
	Acknowledged      bool   `json:"acknowledged"`
	FixReceiptNeeded  bool   `json:"fix_receipt_needed"`
}

func (h *OwnerAlertsHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	identity, status := h.verifyRequestIdentity(r)
	if status != 0 {
		writeOwnerAlertError(w, status)
		return
	}
	if h.acks == nil {
		writeOwnerAlertError(w, http.StatusServiceUnavailable)
		return
	}
	alertID := strings.TrimSpace(r.PathValue("id"))
	if alertID == "" || len(alertID) > 256 {
		writeOwnerAlertError(w, http.StatusBadRequest)
		return
	}
	alert, err := h.acks.GetAlert(r.Context(), alertID)
	if err != nil || !ownerAckEligible(alert) {
		writeOwnerAlertError(w, http.StatusNotFound)
		return
	}
	if ownerAlertOwnerID(alert) != identity.OwnerID {
		// Do not reveal whether an alert owned by another session exists.
		writeOwnerAlertError(w, http.StatusNotFound)
		return
	}
	actor := "agent-run:" + identity.RunID
	if alert.AckedAt != nil {
		if alert.AckedBy != actor {
			writeOwnerAlertError(w, http.StatusConflict)
			return
		}
		writeOwnerJSON(w, http.StatusOK, ownerAlertAckResponse{AlertID: alert.ID, OwnerID: identity.OwnerID, AcknowledgedByRun: identity.RunID, Acknowledged: true, FixReceiptNeeded: true})
		return
	}
	if err := h.acks.AcknowledgeAlert(r.Context(), alert.ID, actor); err != nil {
		writeOwnerAlertError(w, http.StatusConflict)
		return
	}
	updated, err := h.acks.GetAlert(r.Context(), alert.ID)
	if err != nil || updated.AckedAt == nil || updated.AckedBy != actor {
		writeOwnerAlertError(w, http.StatusInternalServerError)
		return
	}
	writeOwnerJSON(w, http.StatusOK, ownerAlertAckResponse{AlertID: updated.ID, OwnerID: identity.OwnerID, AcknowledgedByRun: identity.RunID, Acknowledged: true, FixReceiptNeeded: true})
}

type verifiedAttachedSession struct {
	RunID   string
	OwnerID string
}

func (h *OwnerAlertsHandler) verifyRequestIdentity(r *http.Request) (verifiedAttachedSession, int) {
	if h.identities == nil {
		return verifiedAttachedSession{}, http.StatusServiceUnavailable
	}
	values := r.Header.Values("X-Agent-Identity-Token")
	if len(values) != 1 {
		return verifiedAttachedSession{}, http.StatusUnauthorized
	}
	verifyCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	identity, err := h.identities.VerifyAttachedIdentity(verifyCtx, values[0])
	if err != nil {
		if errors.Is(err, agentmanager.ErrAgentIdentityRejected) {
			return verifiedAttachedSession{}, http.StatusUnauthorized
		}
		return verifiedAttachedSession{}, http.StatusServiceUnavailable
	}
	ownerID, ok := attachedSessionOwnerID(identity)
	if !ok {
		return verifiedAttachedSession{}, http.StatusUnauthorized
	}
	return verifiedAttachedSession{RunID: identity.RunID, OwnerID: ownerID}, 0
}

func attachedSessionOwnerID(identity *agentmanager.AttachedIdentity) (string, bool) {
	if identity == nil || identity.RunID == "" || identity.HarnessKind == "" || identity.HarnessSession == "" || identity.WebConsoleSessionID == "" || !attachedSessionIDPattern.MatchString(identity.HarnessSession) {
		return "", false
	}
	if !strings.HasPrefix(identity.HarnessSession, identity.HarnessKind+"-") {
		return "", false
	}
	return "agent-scope:vrooli-agent-" + identity.HarnessSession, true
}

func ownerAlertOwnerID(alert *models.Alert) string {
	if alert == nil || alert.Details == nil {
		return ""
	}
	for _, key := range []string{"owner_id", "owner"} {
		if owner, ok := alert.Details[key].(string); ok && owner != "" && owner != "unknown" {
			return owner
		}
	}
	return ""
}

func ownerAckEligible(alert *models.Alert) bool {
	if alert == nil || alert.Type != services.ConditionAlertType || alert.ID == "" || alert.ResolvedAt != nil || ownerAlertOwnerID(alert) == "" {
		return false
	}
	if alert.MetricName == services.RuleOwnerActionAge {
		return false
	}
	if alert.Details != nil {
		if ruleID, ok := alert.Details["rule_id"].(string); ok && ruleID == services.RuleOwnerActionAge {
			return false
		}
	}
	return true
}

func writeOwnerAlertError(w http.ResponseWriter, status int) {
	http.Error(w, http.StatusText(status), status)
}

func writeOwnerJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
