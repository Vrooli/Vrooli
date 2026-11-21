package control

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	internal "device-control/internal/control"
	internalflows "device-control/internal/flows"
	"github.com/gorilla/mux"
)

func (h *handler) liveFlowRuns(w http.ResponseWriter, r *http.Request) {
	ids, active := h.service.LiveRunInventory()
	write(w, http.StatusOK, map[string]any{"run_ids": ids, "active_flow_count": active, "quiesced": false})
}

func (h *handler) forgetLiveFlowRun(w http.ResponseWriter, r *http.Request) {
	var request struct {
		AcknowledgeVolatileCacheOnly bool `json:"acknowledge_volatile_cache_only"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil || !request.AcknowledgeVolatileCacheOnly {
		writeError(w, http.StatusBadRequest, "cache_only_acknowledgment_required", "acknowledge_volatile_cache_only must be true")
		return
	}
	id := mux.Vars(r)["id"]
	if err := h.service.ForgetLiveRun(id); err != nil {
		switch {
		case errors.Is(err, internal.ErrLiveRunNotFound):
			writeError(w, http.StatusNotFound, "live_run_not_found", "completed process-local run not found")
		case errors.Is(err, internal.ErrLiveRunInProgress):
			writeError(w, http.StatusConflict, "live_run_in_progress", "run still has an active lease cleanup")
		case errors.Is(err, internal.ErrLiveRunNotEvictable):
			writeError(w, http.StatusConflict, "live_run_not_evictable", "only ordinary completed flow runs can be forgotten")
		default:
			writeError(w, http.StatusConflict, "live_run_cache_inconsistent", "run cache indexes are incomplete; no entries were removed")
		}
		return
	}
	write(w, http.StatusOK, map[string]any{
		"run_id":                           id,
		"scope":                            "volatile_run_cache",
		"cache_deleted":                    true,
		"evidence_preserved":               true,
		"audits_unchanged":                 true,
		"saved_flow_definitions_unchanged": true,
		"retained_store_unchanged":         true,
		"note":                             "This forgets only the process-local run cache. Existing retained or saved copies and evidence may still be available.",
	})
}

func (h *handler) retainedFlowRuns(w http.ResponseWriter, r *http.Request) {
	ids, err := h.service.RetainedRunIDs(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "run_retention_list_failed", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"runs": ids})
}

// POST is an explicit owner decision to retain the full flow and run result.
// The result may contain sensitive targets and arguments; GET is not a summary.
func (h *handler) retainedFlowRun(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	switch r.Method {
	case http.MethodPost:
		var request struct {
			AcknowledgeSensitive bool      `json:"acknowledge_sensitive"`
			ExpiresAt            time.Time `json:"expires_at"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "expected a valid JSON request with an RFC3339 expires_at")
			return
		}
		if !request.AcknowledgeSensitive {
			writeError(w, http.StatusBadRequest, "retention_acknowledgment_required", "retention stores full flow and result, including potentially private content; acknowledge_sensitive must be true")
			return
		}
		if request.ExpiresAt.IsZero() || !request.ExpiresAt.After(time.Now()) {
			writeError(w, http.StatusBadRequest, "retention_expiry_required", "expires_at must be a future RFC3339 timestamp")
			return
		}
		if err := h.service.RetainRun(r.Context(), id, request.ExpiresAt.UTC()); err != nil {
			if errors.Is(err, internal.ErrLiveRunNotFound) {
				writeError(w, http.StatusNotFound, "live_run_not_found", "completed device run not found")
			} else if errors.Is(err, internalflows.ErrInvalidRequest) || errors.Is(err, internalflows.ErrClipboardRetention) {
				writeError(w, http.StatusBadRequest, "run_retention_refused", err.Error())
			} else {
				writeError(w, http.StatusInternalServerError, "run_retention_failed", "run store unavailable or inconsistent")
			}
			return
		}
		write(w, http.StatusOK, map[string]any{"run_id": id, "retained": true, "expires_at": request.ExpiresAt.UTC()})
	case http.MethodDelete:
		if err := h.service.DeleteRetainedRun(r.Context(), id); err != nil {
			writeError(w, http.StatusInternalServerError, "run_retention_delete_failed", err.Error())
			return
		}
		write(w, http.StatusOK, map[string]any{"run_id": id, "retained": false})
	case http.MethodGet:
		run, err := h.service.RetainedRun(r.Context(), id)
		if err != nil {
			if errors.Is(err, internalflows.ErrInvalidRequest) {
				writeError(w, http.StatusNotFound, "retained_run_not_found", "retained run not found")
			} else {
				writeError(w, http.StatusInternalServerError, "retained_run_read_failed", "retained run store unavailable")
			}
			return
		}
		write(w, http.StatusOK, run)
	}
}

func (h *handler) exportFlow(w http.ResponseWriter, r *http.Request) {
	export, err := h.service.ExportFlow(mux.Vars(r)["id"])
	if err != nil {
		if errors.Is(err, internalflows.ErrInvalidRequest) {
			writeError(w, http.StatusNotFound, "run_not_found", "run not found")
		} else {
			writeError(w, http.StatusInternalServerError, "run_export_failed", "run store unavailable")
		}
		return
	}
	write(w, http.StatusOK, export)
}

type flowRequest struct {
	Flow       internal.Flow `json:"flow"`
	StrategyID string        `json:"strategy_id"`
	DeviceID   string        `json:"device_id"`
	Actor      string        `json:"actor"`
	LeaseToken string        `json:"lease_token,omitempty"`
	Confirmed  bool          `json:"confirmed,omitempty"`
}

func (h *handler) validateFlow(w http.ResponseWriter, r *http.Request) {
	var in flowRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	write(w, http.StatusOK, h.service.Validate(r.Context(), in.Flow, in.StrategyID))
}

func (h *handler) runFlow(w http.ResponseWriter, r *http.Request) {
	var in flowRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.service.RunWithLeaseConfirmed(r.Context(), in.Flow, in.DeviceID, in.Actor, in.LeaseToken, in.Confirmed)
	if err != nil {
		if strings.Contains(err.Error(), "requires explicit confirmation") {
			writeError(w, http.StatusPreconditionRequired, "confirmation_required", err.Error())
			return
		}
		if strings.HasPrefix(err.Error(), "unknown device ") {
			writeError(w, http.StatusBadRequest, "unknown_device", err.Error())
			return
		}
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, result)
}
