package backlog

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/providerpool"
)

type effortProviderReserveRequest struct {
	AttemptID string `json:"attempt_id"`
	Units     int64  `json:"units,omitempty"`
	Known     bool   `json:"known,omitempty"`
}

type effortProviderSettleRequest struct {
	UsedUnits int64 `json:"used_units,omitempty"`
	Known     bool  `json:"known,omitempty"`
}

type effortProviderStateResponse struct {
	EffortID string             `json:"effort_id"`
	State    providerpool.State `json:"state"`
}

type effortProviderReservationResponse struct {
	EffortID    string                   `json:"effort_id"`
	Reservation providerpool.Reservation `json:"reservation"`
}

type effortProviderReservationsResponse struct {
	EffortID     string                     `json:"effort_id"`
	Reservations []providerpool.Reservation `json:"reservations"`
}

type effortProviderTotalsResponse struct {
	EffortID string              `json:"effort_id"`
	Totals   providerpool.Totals `json:"totals"`
}

type effortProviderRefreshResponse struct {
	EffortID      string                     `json:"effort_id"`
	ProviderPools []ProviderPoolLimitRefresh `json:"provider_pools"`
}

func poolRefFromRequest(r *http.Request) string {
	return strings.TrimSpace(mux.Vars(r)["pool"])
}

func (h *Handler) effortProviderTarget(w http.ResponseWriter, r *http.Request, action string) (string, string, bool) {
	effortID := effortIDFromRequest(r)
	if effortID == "" {
		apierr.MapError(w, action, apierr.BadRequest("effort_id is required"))
		return "", "", false
	}
	poolRef := poolRefFromRequest(r)
	if poolRef == "" {
		apierr.MapError(w, action, apierr.BadRequest("pool is required"))
		return "", "", false
	}
	if h.effortControl == nil {
		apierr.MapError(w, action, apierr.Internal("effort control service is not configured"))
		return "", "", false
	}
	return effortID, poolRef, true
}

// ObserveEffortProviderLimit records a typed runner-limit or recovery
// observation for one shared provider pool.
func (h *Handler) ObserveEffortProviderLimit(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] observe-effort-provider-limit"
	effortID, poolRef, ok := h.effortProviderTarget(w, r, action)
	if !ok {
		return
	}
	request := providerpool.Observation{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	request.Pool = poolRef
	state, err := h.effortControl.ObserveProviderLimit(effortID, request)
	if err != nil {
		mapEffortProviderError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortProviderStateResponse{EffortID: effortID, State: state})
}

// ObserveEffortProviderFailure records an AI Gateway provider failure as a pool
// observation. The caller supplies the gateway's failure class and observed
// provenance; the pool translates the class vocabulary and carries the recovery
// window. A class with no shared-pool recovery condition is a no-op that
// returns the pool's current state.
func (h *Handler) ObserveEffortProviderFailure(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] observe-effort-provider-failure"
	effortID, poolRef, ok := h.effortProviderTarget(w, r, action)
	if !ok {
		return
	}
	request := providerpool.ProviderFailure{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	request.Pool = poolRef
	state, err := h.effortControl.ObserveProviderFailure(effortID, request)
	if err != nil {
		mapEffortProviderError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortProviderStateResponse{EffortID: effortID, State: state})
}

// GetEffortProviderPool returns the current admission state for one pool.
func (h *Handler) GetEffortProviderPool(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] get-effort-provider-pool"
	effortID, poolRef, ok := h.effortProviderTarget(w, r, action)
	if !ok {
		return
	}
	state, err := h.effortControl.ProviderPoolState(effortID, poolRef)
	if err != nil {
		mapEffortProviderError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortProviderStateResponse{EffortID: effortID, State: state})
}

// ReserveEffortProviderSlot accounts a dispatch before the owner is contacted.
func (h *Handler) ReserveEffortProviderSlot(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] reserve-effort-provider-slot"
	effortID, poolRef, ok := h.effortProviderTarget(w, r, action)
	if !ok {
		return
	}
	request := effortProviderReserveRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	reservation, err := h.effortControl.ReserveProviderSlot(effortID, providerpool.Reservation{
		AttemptID: request.AttemptID,
		Pool:      poolRef,
		Units:     request.Units,
		Known:     request.Known,
	})
	if err != nil {
		mapEffortProviderError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortProviderReservationResponse{EffortID: effortID, Reservation: reservation})
}

// SettleEffortProviderSlot records terminal usage for one reservation.
func (h *Handler) SettleEffortProviderSlot(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] settle-effort-provider-slot"
	effortID, poolRef, ok := h.effortProviderTarget(w, r, action)
	if !ok {
		return
	}
	attemptID := attemptIDFromRequest(r)
	if attemptID == "" {
		apierr.MapError(w, action, apierr.BadRequest("attempt_id is required"))
		return
	}
	request := effortProviderSettleRequest{}
	if err := httputil.DecodeJSONStrict(r, &request); err != nil {
		apierr.MapError(w, action, apierr.BadRequest("invalid request body"))
		return
	}
	reservation, err := h.effortControl.SettleProviderSlot(effortID, poolRef, attemptID, request.UsedUnits, request.Known)
	if err != nil {
		mapEffortProviderError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortProviderReservationResponse{EffortID: effortID, Reservation: reservation})
}

// ReleaseEffortProviderSlot finalizes a reservation whose effect never started.
func (h *Handler) ReleaseEffortProviderSlot(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] release-effort-provider-slot"
	effortID, poolRef, ok := h.effortProviderTarget(w, r, action)
	if !ok {
		return
	}
	attemptID := attemptIDFromRequest(r)
	if attemptID == "" {
		apierr.MapError(w, action, apierr.BadRequest("attempt_id is required"))
		return
	}
	reservation, err := h.effortControl.ReleaseProviderSlot(effortID, poolRef, attemptID)
	if err != nil {
		mapEffortProviderError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortProviderReservationResponse{EffortID: effortID, Reservation: reservation})
}

// ListEffortProviderReservations lists the pool's reservations. Totals are
// included so one read reconciles held and settled investment.
func (h *Handler) ListEffortProviderReservations(w http.ResponseWriter, r *http.Request) {
	const action = "[backlog] list-effort-provider-reservations"
	effortID, poolRef, ok := h.effortProviderTarget(w, r, action)
	if !ok {
		return
	}
	reservations, err := h.effortControl.ProviderReservations(effortID, poolRef)
	if err != nil {
		mapEffortProviderError(w, action, err)
		return
	}
	_ = httputil.JSON(w, effortProviderReservationsResponse{EffortID: effortID, Reservations: reservations})
}

func mapEffortProviderError(w http.ResponseWriter, action string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		apierr.MapError(w, action, apierr.NotFound("effort control revision not found"))
	case errors.Is(err, ErrProviderAccountingUnavailable):
		apierr.MapError(w, action, apierr.Unavailable("provider pool accounting is not configured"))
	case errors.Is(err, providerpool.ErrPoolBlocked):
		apierr.MapError(w, action, apierr.Wrap(providerpool.ErrPoolBlocked, http.StatusTooManyRequests, err.Error()))
	case errors.Is(err, providerpool.ErrPoolSaturated):
		apierr.MapError(w, action, apierr.Wrap(providerpool.ErrPoolSaturated, http.StatusTooManyRequests, err.Error()))
	case errors.Is(err, providerpool.ErrAllowanceExhausted):
		apierr.MapError(w, action, apierr.Wrap(providerpool.ErrAllowanceExhausted, http.StatusTooManyRequests, err.Error()))
	case errors.Is(err, providerpool.ErrUnknownReservation):
		apierr.MapError(w, action, apierr.NotFound("provider reservation is not reserved"))
	case errors.Is(err, providerpool.ErrReservationConflict):
		apierr.MapError(w, action, apierr.Conflict("provider reservation identity is reused"))
	case errors.Is(err, providerpool.ErrReservationActive):
		apierr.MapError(w, action, apierr.Conflict("provider reservation is still unresolved"))
	case errors.Is(err, providerpool.ErrUnknownClass), errors.Is(err, providerpool.ErrPoolMismatch):
		apierr.MapError(w, action, apierr.BadRequest("%s", err.Error()))
	default:
		apierr.MapError(w, action, apierr.BadRequest("%s", err.Error()))
	}
}
