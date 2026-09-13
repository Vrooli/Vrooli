package backlog

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/identity"
)

type effortCompletionEvidenceRequest struct {
	Refs []string `json:"refs,omitempty"`
}

type effortCompletionAcceptRequest struct {
	Actor string `json:"actor,omitempty"`
}

type effortControlResponse struct {
	Effort          identity.EffortControl `json:"effort"`
	AuthorityDigest string                 `json:"authority_digest"`
}

func effortIDFromRequest(r *http.Request) string {
	return strings.TrimSpace(mux.Vars(r)["effortID"])
}

// AdmitEffort creates the first revision of an effort. It is version-checked
// against durable owner state and refuses to replace an existing effort.
func (h *Handler) AdmitEffort(w http.ResponseWriter, r *http.Request) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, "[backlog] admit-effort", apierr.BadRequest("effort_id is required"))
		return
	}
	if h.effortControl == nil {
		apierr.MapError(w, "[backlog] admit-effort", apierr.Internal("effort control service is not configured"))
		return
	}
	control := identity.EffortControl{EffortID: effortID}
	if err := httputil.DecodeJSONStrict(r, &control); err != nil {
		apierr.MapError(w, "[backlog] admit-effort", apierr.BadRequest("invalid request body"))
		return
	}
	if control.EffortID == "" {
		control.EffortID = effortID
	} else if control.EffortID != effortID {
		apierr.MapError(w, "[backlog] admit-effort", apierr.Conflict("request effort_id does not match the addressed effort"))
		return
	}
	admitted, err := h.effortControl.Admit(control)
	if err != nil {
		mapEffortControlError(w, "[backlog] admit-effort", err)
		return
	}
	_ = httputil.JSON(w, effortControlResponse{Effort: admitted, AuthorityDigest: admitted.AuthorityDigest()})
}

// AmendEffort changes reviewed authority at the exact next revision.
func (h *Handler) AmendEffort(w http.ResponseWriter, r *http.Request) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, "[backlog] amend-effort", apierr.BadRequest("effort_id is required"))
		return
	}
	if h.effortControl == nil {
		apierr.MapError(w, "[backlog] amend-effort", apierr.Internal("effort control service is not configured"))
		return
	}
	control := identity.EffortControl{EffortID: effortID}
	if err := httputil.DecodeJSONStrict(r, &control); err != nil {
		apierr.MapError(w, "[backlog] amend-effort", apierr.BadRequest("invalid request body"))
		return
	}
	if control.EffortID == "" {
		control.EffortID = effortID
	} else if control.EffortID != effortID {
		apierr.MapError(w, "[backlog] amend-effort", apierr.Conflict("request effort_id does not match the addressed effort"))
		return
	}
	amended, err := h.effortControl.Amend(control)
	if err != nil {
		mapEffortControlError(w, "[backlog] amend-effort", err)
		return
	}
	_ = httputil.JSON(w, effortControlResponse{Effort: amended, AuthorityDigest: amended.AuthorityDigest()})
}

// GetEffort returns the current admitted revision.
func (h *Handler) GetEffort(w http.ResponseWriter, r *http.Request) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, "[backlog] get-effort", apierr.BadRequest("effort_id is required"))
		return
	}
	if h.effortControl == nil {
		apierr.MapError(w, "[backlog] get-effort", apierr.Internal("effort control service is not configured"))
		return
	}
	control, err := h.effortControl.Get(effortID)
	if err != nil {
		mapEffortControlError(w, "[backlog] get-effort", err)
		return
	}
	_ = httputil.JSON(w, effortControlResponse{Effort: control, AuthorityDigest: control.AuthorityDigest()})
}

// MarkEffortEvidenceComplete records owner evidence completion. It can never
// set human acceptance.
func (h *Handler) MarkEffortEvidenceComplete(w http.ResponseWriter, r *http.Request) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, "[backlog] mark-effort-evidence", apierr.BadRequest("effort_id is required"))
		return
	}
	if h.effortControl == nil {
		apierr.MapError(w, "[backlog] mark-effort-evidence", apierr.Internal("effort control service is not configured"))
		return
	}
	request := effortCompletionEvidenceRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, "[backlog] mark-effort-evidence", apierr.BadRequest("invalid request body"))
		return
	}
	control, err := h.effortControl.MarkEvidenceComplete(effortID, request.Refs...)
	if err != nil {
		mapEffortControlError(w, "[backlog] mark-effort-evidence", err)
		return
	}
	_ = httputil.JSON(w, effortControlResponse{Effort: control, AuthorityDigest: control.AuthorityDigest()})
}

// AcceptEffortCompletion records the authenticated human product disposition.
// A blank actor falls back to verified request provenance; if both are blank
// the owner refuses rather than treating evidence completion as acceptance.
func (h *Handler) AcceptEffortCompletion(w http.ResponseWriter, r *http.Request) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, "[backlog] accept-effort", apierr.BadRequest("effort_id is required"))
		return
	}
	if h.effortControl == nil {
		apierr.MapError(w, "[backlog] accept-effort", apierr.Internal("effort control service is not configured"))
		return
	}
	request := effortCompletionAcceptRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, "[backlog] accept-effort", apierr.BadRequest("invalid request body"))
		return
	}
	actor := strings.TrimSpace(request.Actor)
	if actor == "" {
		actor = strings.TrimSpace(identity.FromContext(r.Context()).FormatStartedBy())
	}
	control, err := h.effortControl.HumanAccept(effortID, actor)
	if err != nil {
		mapEffortControlError(w, "[backlog] accept-effort", err)
		return
	}
	_ = httputil.JSON(w, effortControlResponse{Effort: control, AuthorityDigest: control.AuthorityDigest()})
}

// mapEffortControlError translates identity sentinels into transport status.
func mapEffortControlError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierr.MapError(w, action, apierr.NotFound("effort control revision not found"))
	case errors.Is(err, identity.ErrEffortIdentityMismatch):
		apierr.MapError(w, action, apierr.Conflict("effort identity mismatch"))
	case errors.Is(err, identity.ErrRevisionConflict):
		apierr.MapError(w, action, apierr.Conflict("effort revision conflict: an existing revision changes only through amendment"))
	case errors.Is(err, identity.ErrCandidatePolicyImmutable):
		apierr.MapError(w, action, apierr.Conflict("candidate policy binding is immutable within a revision"))
	case errors.Is(err, identity.ErrAdditionalCapacityDenied):
		apierr.MapError(w, action, apierr.Conflict("effort-wide descendant allowance is exhausted"))
	case errors.Is(err, identity.ErrHumanAcceptanceRequired):
		apierr.MapError(w, action, apierr.BadRequest("authenticated human product acceptance is required"))
	case errors.Is(err, identity.ErrUnauthorizedAction):
		apierr.MapError(w, action, apierr.BadRequest("delegated action is not authorized"))
	case errors.Is(err, ErrEffortPolicyUnavailable):
		apierr.MapError(w, action, apierr.Conflict("approved effort policy is unavailable"))
	default:
		apierr.MapError(w, action, apierr.BadRequest("%s", err.Error()))
	}
}
