package backlog

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/repairbudget"
)

type effortRepairReserveRequest struct {
	AttemptID   string   `json:"attempt_id"`
	Fingerprint string   `json:"fingerprint"`
	Component   string   `json:"component"`
	Kind        string   `json:"kind,omitempty"`
	Class       string   `json:"class,omitempty"`
	Hypothesis  string   `json:"hypothesis,omitempty"`
	NewEvidence bool     `json:"new_evidence,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
	OwnerWork   string   `json:"owner_work,omitempty"`
	OwnerRun    string   `json:"owner_run,omitempty"`
}

type effortRepairFinishRequest struct {
	Outcome  string   `json:"outcome,omitempty"`
	Evidence []string `json:"evidence,omitempty"`
}

type effortRepairReserveResponse struct {
	EffortID  string                 `json:"effort_id"`
	Admission repairbudget.Admission `json:"admission"`
}

type effortRepairFinishResponse struct {
	EffortID string             `json:"effort_id"`
	Event    repairbudget.Event `json:"event"`
}

type effortRepairTotalsResponse struct {
	EffortID string              `json:"effort_id"`
	Totals   repairbudget.Totals `json:"totals"`
}

func attemptIDFromRequest(r *http.Request) string {
	return strings.TrimSpace(mux.Vars(r)["attemptID"])
}

// BeginEffortRepair reserves cumulative repair/retry/status investment before
// the coordinator contacts an owner. A refused reservation means no new repair
// is admitted; independent permitted work stays eligible.
func (h *Handler) BeginEffortRepair(w http.ResponseWriter, r *http.Request) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, "[backlog] begin-effort-repair", apierr.BadRequest("effort_id is required"))
		return
	}
	if h.effortControl == nil {
		apierr.MapError(w, "[backlog] begin-effort-repair", apierr.Internal("effort control service is not configured"))
		return
	}
	request := effortRepairReserveRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, "[backlog] begin-effort-repair", apierr.BadRequest("invalid request body"))
		return
	}
	admission, err := h.effortControl.BeginRepair(effortID, repairbudget.Event{
		AttemptID:   request.AttemptID,
		Fingerprint: request.Fingerprint,
		Component:   request.Component,
		Kind:        request.Kind,
		Class:       request.Class,
		Hypothesis:  request.Hypothesis,
		NewEvidence: request.NewEvidence,
		Evidence:    request.Evidence,
		OwnerWork:   request.OwnerWork,
		OwnerRun:    request.OwnerRun,
	})
	if err != nil {
		mapEffortRepairError(w, "[backlog] begin-effort-repair", err)
		return
	}
	_ = httputil.JSON(w, effortRepairReserveResponse{EffortID: effortID, Admission: admission})
}

// FinishEffortRepair records the disposition of a reserved intervention. It
// never charges twice, so an interrupted attempt remains charged from Begin.
func (h *Handler) FinishEffortRepair(w http.ResponseWriter, r *http.Request) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, "[backlog] finish-effort-repair", apierr.BadRequest("effort_id is required"))
		return
	}
	attemptID := attemptIDFromRequest(r)
	if attemptID == "" {
		apierr.MapError(w, "[backlog] finish-effort-repair", apierr.BadRequest("attempt_id is required"))
		return
	}
	if h.effortControl == nil {
		apierr.MapError(w, "[backlog] finish-effort-repair", apierr.Internal("effort control service is not configured"))
		return
	}
	request := effortRepairFinishRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, "[backlog] finish-effort-repair", apierr.BadRequest("invalid request body"))
		return
	}
	event, err := h.effortControl.FinishRepair(effortID, attemptID, request.Outcome, request.Evidence...)
	if err != nil {
		mapEffortRepairError(w, "[backlog] finish-effort-repair", err)
		return
	}
	_ = httputil.JSON(w, effortRepairFinishResponse{EffortID: effortID, Event: event})
}

// GetEffortRepairs returns the cumulative investment summary for one effort.
func (h *Handler) GetEffortRepairs(w http.ResponseWriter, r *http.Request) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, "[backlog] get-effort-repairs", apierr.BadRequest("effort_id is required"))
		return
	}
	if h.effortControl == nil {
		apierr.MapError(w, "[backlog] get-effort-repairs", apierr.Internal("effort control service is not configured"))
		return
	}
	totals, err := h.effortControl.RepairTotals(effortID)
	if err != nil {
		mapEffortRepairError(w, "[backlog] get-effort-repairs", err)
		return
	}
	_ = httputil.JSON(w, effortRepairTotalsResponse{EffortID: effortID, Totals: totals})
}

type effortRepairRouteRequest struct {
	OwnerWork string `json:"owner_work,omitempty"`
	OwnerRun  string `json:"owner_run,omitempty"`
}

type effortRepairReasonRequest struct {
	Reason string `json:"reason,omitempty"`
}

type effortRepairReconcileRequest struct {
	OwnerActive bool   `json:"owner_active"`
	OwnerRun    string `json:"owner_run,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type effortRepairEventResponse struct {
	EffortID string             `json:"effort_id"`
	Event    repairbudget.Event `json:"event"`
}

type effortRepairDispatchResponse struct {
	EffortID string                     `json:"effort_id"`
	Dispatch repairbudget.DispatchState `json:"dispatch"`
}

func (h *Handler) effortRepairTarget(w http.ResponseWriter, r *http.Request, action string) (string, string, bool) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, action, apierr.BadRequest("effort_id is required"))
		return "", "", false
	}
	attemptID := attemptIDFromRequest(r)
	if attemptID == "" {
		apierr.MapError(w, action, apierr.BadRequest("attempt_id is required"))
		return "", "", false
	}
	if h.effortControl == nil {
		apierr.MapError(w, action, apierr.Internal("effort control service is not configured"))
		return "", "", false
	}
	return effortID, attemptID, true
}

// AcknowledgeEffortRepairDispatch records an owner's start acknowledgement for
// a reserved operation so a lost callback can be reconciled later.
func (h *Handler) AcknowledgeEffortRepairDispatch(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] acknowledge-effort-repair-dispatch"
	effortID, attemptID, ok := h.effortRepairTarget(w, r, action)
	if !ok {
		return
	}
	request := effortRepairRouteRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	event, err := h.effortControl.AcknowledgeDispatch(effortID, attemptID, request.OwnerWork, request.OwnerRun)
	if err != nil {
		mapEffortRepairError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortRepairEventResponse{EffortID: effortID, Event: event})
}

// MarkEffortRepairUncertain records that a dispatch start response was lost.
// A replacement stays refused until the operation is reconciled.
func (h *Handler) MarkEffortRepairUncertain(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] mark-effort-repair-uncertain"
	effortID, attemptID, ok := h.effortRepairTarget(w, r, action)
	if !ok {
		return
	}
	request := effortRepairReasonRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	event, err := h.effortControl.MarkDispatchUncertain(effortID, attemptID, request.Reason)
	if err != nil {
		mapEffortRepairError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortRepairEventResponse{EffortID: effortID, Event: event})
}

// ReconcileEffortRepairDispatch records an owner observation and returns the
// resolved dispatch state.
func (h *Handler) ReconcileEffortRepairDispatch(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] reconcile-effort-repair-dispatch"
	effortID, attemptID, ok := h.effortRepairTarget(w, r, action)
	if !ok {
		return
	}
	request := effortRepairReconcileRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	state, err := h.effortControl.ReconcileDispatch(effortID, attemptID, request.OwnerActive, request.OwnerRun, request.Reason)
	if err != nil {
		mapEffortRepairError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortRepairDispatchResponse{EffortID: effortID, Dispatch: state})
}

// FallbackEffortRepairDispatch re-dispatches a reconciled operation under the
// same charged identity.
func (h *Handler) FallbackEffortRepairDispatch(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] fallback-effort-repair-dispatch"
	effortID, attemptID, ok := h.effortRepairTarget(w, r, action)
	if !ok {
		return
	}
	request := effortRepairRouteRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	event, err := h.effortControl.FallbackDispatch(effortID, attemptID, request.OwnerWork, request.OwnerRun)
	if err != nil {
		mapEffortRepairError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortRepairEventResponse{EffortID: effortID, Event: event})
}

// CancelEffortRepairDispatch fences a reserved operation terminal.
func (h *Handler) CancelEffortRepairDispatch(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] cancel-effort-repair-dispatch"
	effortID, attemptID, ok := h.effortRepairTarget(w, r, action)
	if !ok {
		return
	}
	request := effortRepairReasonRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	event, err := h.effortControl.CancelDispatch(effortID, attemptID, request.Reason)
	if err != nil {
		mapEffortRepairError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortRepairEventResponse{EffortID: effortID, Event: event})
}

// GetEffortRepairDispatch returns the route-owner state for one operation.
func (h *Handler) GetEffortRepairDispatch(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] get-effort-repair-dispatch"
	effortID, attemptID, ok := h.effortRepairTarget(w, r, action)
	if !ok {
		return
	}
	state, err := h.effortControl.DispatchState(effortID, attemptID)
	if err != nil {
		mapEffortRepairError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortRepairDispatchResponse{EffortID: effortID, Dispatch: state})
}

// mapEffortRepairError translates repair-accounting sentinels into transport
// status. An exhausted allowance is a 429: the request is coherent but the
// cumulative waste bound refuses a new intervention.
func mapEffortRepairError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierr.MapError(w, action, apierr.NotFound("effort control revision not found"))
	case errors.Is(err, ErrRepairAccountingUnavailable):
		apierr.MapError(w, action, apierr.Unavailable("repair accounting is not configured"))
	case errors.Is(err, repairbudget.ErrBudgetExhausted):
		apierr.MapError(w, action, apierr.Wrap(repairbudget.ErrBudgetExhausted, http.StatusTooManyRequests, err.Error()))
	case errors.Is(err, repairbudget.ErrIdentityConflict):
		apierr.MapError(w, action, apierr.Conflict("attempt identity is reused with different work"))
	case errors.Is(err, repairbudget.ErrStaleEvidence):
		apierr.MapError(w, action, apierr.Conflict("repeat repair fingerprint requires new evidence"))
	case errors.Is(err, repairbudget.ErrReclassificationRefused):
		apierr.MapError(w, action, apierr.Conflict("repair fingerprint charge class cannot change"))
	case errors.Is(err, repairbudget.ErrUnknownAttempt):
		apierr.MapError(w, action, apierr.NotFound("repair attempt is not reserved"))
	case errors.Is(err, repairbudget.ErrDispatchUncertain):
		apierr.MapError(w, action, apierr.Conflict("dispatch is uncertain; reconcile the original operation before replacement"))
	case errors.Is(err, repairbudget.ErrExecutorActive):
		apierr.MapError(w, action, apierr.Conflict("an executor is already active for this operation"))
	case errors.Is(err, repairbudget.ErrAttemptTerminal):
		apierr.MapError(w, action, apierr.Conflict("attempt is already terminal"))
	default:
		apierr.MapError(w, action, apierr.BadRequest("%s", err.Error()))
	}
}
