package evidenceretention

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/operatordecision"
)

// Authorizer verifies a human owner credential and returns the subject.
type Authorizer func(*http.Request) (string, int, error)

// Handler serves the evidence retention contract:
//
//	GET  /api/v1/storage/evidence        dry-run classification report
//	POST /api/v1/storage/evidence/prune  {"efforts":[...],"classes":[...],"dry_run":bool,"reason":"..."}
//	GET  /api/v1/storage/evidence/prune/requests/{id}  an agent's approval request
//
// A prune that removes anything requires the owner; a dry run does not. A
// verified agent never removes on its own authority: its real prune raises an
// operator decision (202) that Agent Manager applies only once approved.
type Handler struct {
	service   *Service
	authorize Authorizer
	approvals *Approvals
}

// WebConsoleSessionIDHeader carries an agent's stable web-console session id.
const WebConsoleSessionIDHeader = "X-Wc-Session-Id"

// NewHandler wires the handler. A nil authorizer refuses every real prune; nil
// approvals leave agents refused with a pointer to the owner path.
func NewHandler(service *Service, authorize Authorizer, approvals *Approvals) *Handler {
	return &Handler{service: service, authorize: authorize, approvals: approvals}
}

// Report serves the dry-run classification.
func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	report, err := h.service.Scan(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeBody(w, http.StatusOK, report)
}

// Prune removes regenerable intermediates of operator-named or
// auto-eligible efforts.
func (h *Handler) Prune(w http.ResponseWriter, r *http.Request) {
	var req PruneRequest
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(strings.TrimSpace(string(body))) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	} else {
		req.DryRun = true
	}
	req.Actor = ""
	if !req.DryRun {
		if p, ok := identity.PrincipalFromContext(r.Context()); ok && p.Kind == identity.ActorAgent && p.IsVerified() {
			h.requestApproval(w, r, body, req, p.Subject)
			return
		}
		if h.authorize == nil {
			writeError(w, http.StatusForbidden, errors.New("evidence prune requires owner authorization, which is not configured"))
			return
		}
		subject, status, err := h.authorize(r)
		if err != nil {
			writeError(w, status, err)
			return
		}
		if strings.TrimSpace(req.Reason) == "" || len(req.Reason) > 512 {
			writeError(w, http.StatusBadRequest, errors.New("a real prune requires a reason of 1-512 bytes"))
			return
		}
		req.Actor = subject
	}
	receipt, err := h.service.Prune(r.Context(), req)
	if errors.Is(err, ErrInvalidRequest) {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	status := http.StatusOK
	if err != nil {
		if len(receipt.Removed) == 0 && len(receipt.Errors) == 0 {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		status = http.StatusMultiStatus
	}
	writeBody(w, status, receipt)
}

// requestApproval answers an agent's real prune with an operator decision.
func (h *Handler) requestApproval(w http.ResponseWriter, r *http.Request, raw []byte, req PruneRequest, agent string) {
	if h.approvals == nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("evidence prune approvals are not configured on this host; ask the operator to run the prune with --local-owner"))
		return
	}
	var body struct {
		WebConsoleSessionID string `json:"web_console_session_id"`
	}
	_ = json.Unmarshal(raw, &body)
	session := strings.TrimSpace(body.WebConsoleSessionID)
	if session == "" {
		session = strings.TrimSpace(r.Header.Get(WebConsoleSessionIDHeader))
	}
	ticket, err := h.approvals.Request(r.Context(), ApprovalRequest{
		Efforts: req.Efforts, Classes: req.Classes, Reason: req.Reason, RequestedBy: agent, WebConsoleSessionID: session,
	})
	switch {
	case errors.Is(err, ErrInvalidRequest):
		writeError(w, http.StatusBadRequest, err)
		return
	case errors.Is(err, operatordecision.ErrFlood):
		writeError(w, http.StatusTooManyRequests, errors.New("too many outstanding evidence-prune requests; wait for the operator to decide"))
		return
	case err != nil && !ticket.Raised:
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if ticket.DecisionID != "" && !ticket.Raised && len(ticket.Preview.Removed) == 0 {
		writeBody(w, http.StatusOK, map[string]any{
			"status":  "nothing_to_prune",
			"preview": ticket.Preview,
			"message": "the preview removes nothing, so no operator decision was raised",
		})
		return
	}
	if !ticket.Raised {
		writeError(w, http.StatusServiceUnavailable, errors.New("evidence prune approvals are not configured on this host"))
		return
	}
	out := map[string]any{
		"status":         "approval_pending",
		"decision_id":    ticket.DecisionID,
		"preview_bytes":  ticket.Preview.RemovedBytes,
		"preview_files":  ticket.Preview.RemovedFiles,
		"preview_paths":  len(ticket.Preview.Removed),
		"message":        "an agent cannot prune evidence on its own authority; an operator decision was raised in the web-console decision inbox, and Agent Manager applies exactly this previewed prune once approved",
		"status_command": "agent-manager storage evidence-prune --status " + ticket.DecisionID,
		"await": map[string]any{
			"producer":      "operator-decision",
			"key":           ticket.DecisionID,
			"park_command":  "agent-manager run park <run-id> --producer operator-decision --key " + ticket.DecisionID + " --timeout 12h",
			"park_optional": true,
			"note":          "The answer is delivered back to this run automatically; parking only saves tokens while waiting.",
		},
	}
	if err != nil {
		out["publish_error"] = err.Error()
	}
	writeBody(w, http.StatusAccepted, out)
}

// RequestStatus serves one agent approval request.
func (h *Handler) RequestStatus(w http.ResponseWriter, r *http.Request) {
	if h.approvals == nil {
		writeError(w, http.StatusNotFound, errors.New("evidence prune approvals are not configured"))
		return
	}
	status, err := h.approvals.Status(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if status == nil {
		writeError(w, http.StatusNotFound, errors.New("no such evidence prune request"))
		return
	}
	writeBody(w, http.StatusOK, status)
}

func writeError(w http.ResponseWriter, status int, err error) {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	writeBody(w, status, map[string]string{"error": err.Error()})
}

func writeBody(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
