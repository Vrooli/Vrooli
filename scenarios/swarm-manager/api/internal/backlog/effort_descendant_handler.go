package backlog

import (
	"errors"
	"net/http"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/descendantbudget"
	"swarm-manager/internal/httputil"
)

type effortDescendantAdmitRequest struct {
	AttemptID       string `json:"attempt_id"`
	ParentAttemptID string `json:"parent_attempt_id,omitempty"`
	Depth           int    `json:"depth,omitempty"`
	Premium         bool   `json:"premium,omitempty"`
}

type effortDescendantReservationResponse struct {
	EffortID    string                       `json:"effort_id"`
	Reservation descendantbudget.Reservation `json:"reservation"`
}

type effortDescendantListResponse struct {
	EffortID     string                         `json:"effort_id"`
	Reservations []descendantbudget.Reservation `json:"reservations"`
	Totals       descendantbudget.Totals        `json:"totals"`
}

func (h *Handler) effortDescendantTarget(w http.ResponseWriter, r *http.Request, action string) (string, bool) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, action, apierr.BadRequest("effort_id is required"))
		return "", false
	}
	if h.effortControl == nil {
		apierr.MapError(w, action, apierr.Internal("effort control service is not configured"))
		return "", false
	}
	return effortID, true
}

// AdmitEffortDescendant charges one descendant slot against the effort's shared
// aggregate grant before the descendant is dispatched.
func (h *Handler) AdmitEffortDescendant(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] admit-effort-descendant"
	effortID, ok := h.effortDescendantTarget(w, r, action)
	if !ok {
		return
	}
	request := effortDescendantAdmitRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	reservation, err := h.effortControl.AdmitDescendant(effortID, descendantbudget.Reservation{
		AttemptID:       request.AttemptID,
		ParentAttemptID: request.ParentAttemptID,
		Depth:           request.Depth,
		Premium:         request.Premium,
	})
	if err != nil {
		mapEffortDescendantError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortDescendantReservationResponse{EffortID: effortID, Reservation: reservation})
}

// ReleaseEffortDescendant ends one reservation and returns its slot.
func (h *Handler) ReleaseEffortDescendant(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] release-effort-descendant"
	effortID, ok := h.effortDescendantTarget(w, r, action)
	if !ok {
		return
	}
	attemptID := attemptIDFromRequest(r)
	if attemptID == "" {
		apierr.MapError(w, action, apierr.BadRequest("attempt_id is required"))
		return
	}
	reservation, err := h.effortControl.ReleaseDescendant(effortID, attemptID)
	if err != nil {
		mapEffortDescendantError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortDescendantReservationResponse{EffortID: effortID, Reservation: reservation})
}

// ListEffortDescendants returns reservations and totals, so one read reconciles
// active descendant capacity against the applied cap.
func (h *Handler) ListEffortDescendants(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] list-effort-descendants"
	effortID, ok := h.effortDescendantTarget(w, r, action)
	if !ok {
		return
	}
	reservations, err := h.effortControl.DescendantReservations(effortID)
	if err != nil {
		mapEffortDescendantError(w, action, err)
		return
	}
	totals, err := h.effortControl.DescendantTotals(effortID)
	if err != nil {
		mapEffortDescendantError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortDescendantListResponse{EffortID: effortID, Reservations: reservations, Totals: totals})
}

func mapEffortDescendantError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierr.MapError(w, action, apierr.NotFound("effort control revision not found"))
	case errors.Is(err, ErrDescendantAccountingUnavailable):
		apierr.MapError(w, action, apierr.Unavailable("descendant accounting is not configured"))
	case errors.Is(err, descendantbudget.ErrCapacityExhausted):
		apierr.MapError(w, action, apierr.Conflict("effort-wide descendant allowance is exhausted"))
	case errors.Is(err, descendantbudget.ErrDepthExceeded):
		apierr.MapError(w, action, apierr.Conflict("descendant depth exceeds the approved maximum"))
	case errors.Is(err, descendantbudget.ErrReservationConflict):
		apierr.MapError(w, action, apierr.Conflict("descendant reservation identity is reused"))
	case errors.Is(err, descendantbudget.ErrParentNotActive):
		apierr.MapError(w, action, apierr.Conflict("descendant parent is not active"))
	case errors.Is(err, descendantbudget.ErrUnknownReservation):
		apierr.MapError(w, action, apierr.NotFound("descendant reservation is not reserved"))
	default:
		apierr.MapError(w, action, apierr.BadRequest("%s", err.Error()))
	}
}
