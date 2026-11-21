package control

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	internal "device-control/internal/control"
	"github.com/gorilla/mux"
)

func (h *handler) listSessions(w http.ResponseWriter, r *http.Request) {
	sessions := h.service.ListLiveSessionsContext(r.Context())
	views := make([]map[string]any, 0, len(sessions))
	for _, session := range sessions {
		views = append(views, publicSession(session))
	}
	write(w, http.StatusOK, map[string]any{"sessions": views})
}

func publicSession(session internal.Session) map[string]any {
	return map[string]any{
		"id": session.ID, "device_id": session.DeviceID, "actor": session.Actor,
		"state": session.State, "kill_reason": session.KillReason,
		"expires_at": session.ExpiresAt, "created_at": session.CreatedAt,
	}
}

type acquireRequest struct {
	DeviceID   string `json:"device_id"`
	Actor      string `json:"actor"`
	TTLSeconds int    `json:"ttl_seconds"`
}

type externalRecordingRequest struct {
	DeviceID   string `json:"device_id"`
	Actor      string `json:"actor"`
	LeaseToken string `json:"lease_token"`
	HandleID   string `json:"handle_id,omitempty"`
	Transport  string `json:"transport,omitempty"`
}

type reviewRecordingRequest struct {
	DeviceID     string   `json:"device_id"`
	Actor        string   `json:"actor"`
	LeaseToken   string   `json:"lease_token"`
	ReferenceIDs []string `json:"reference_ids"`
}

func (h *handler) startExternalRecording(w http.ResponseWriter, r *http.Request) {
	var in externalRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	handle, err := h.service.StartExternalRecording(r.Context(), in.DeviceID, in.Actor, in.LeaseToken, in.Transport)
	if err != nil {
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"handle": handle})
}

func (h *handler) stopExternalRecording(w http.ResponseWriter, r *http.Request) {
	var in externalRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.service.StopExternalRecording(r.Context(), in.DeviceID, in.Actor, in.LeaseToken, in.HandleID)
	if err != nil {
		if result.Reference.ID != "" {
			write(w, http.StatusConflict, map[string]any{"status": "failed", "code": "recording_stop_uncertain", "message": err.Error(), "recovered_recording": result})
		} else {
			writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		}
		return
	}
	write(w, http.StatusOK, result)
}

func (h *handler) inspectExternalRecording(w http.ResponseWriter, r *http.Request) {
	var in externalRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	inspection, err := h.service.InspectExternalRecording(r.Context(), in.DeviceID, in.LeaseToken, in.HandleID)
	if err != nil {
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, inspection)
}

func (h *handler) concatRecordings(w http.ResponseWriter, r *http.Request) {
	var in reviewRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := h.service.FinalizeReviewRecording(r.Context(), in.DeviceID, in.Actor, in.LeaseToken, in.ReferenceIDs)
	if err != nil {
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, result)
}

func (h *handler) acquire(w http.ResponseWriter, r *http.Request) {
	var in acquireRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	s, err := h.service.AcquireContext(r.Context(), in.DeviceID, in.Actor, time.Duration(in.TTLSeconds)*time.Second)
	if err != nil {
		if strings.HasPrefix(err.Error(), "unknown device ") {
			writeError(w, http.StatusBadRequest, "unknown_device", err.Error())
			return
		}
		writeError(w, http.StatusConflict, "lease_refused", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"session": s})
}

func (h *handler) kill(w http.ResponseWriter, r *http.Request) {
	s, err := h.service.KillContext(r.Context(), mux.Vars(r)["id"], "operator requested kill")
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"session": publicSession(s)})
}

func (h *handler) release(w http.ResponseWriter, r *http.Request) {
	s, err := h.service.ReleaseContext(r.Context(), mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"session": publicSession(s)})
}

func (h *handler) validateLease(w http.ResponseWriter, r *http.Request) {
	var in struct {
		DeviceID   string `json:"device_id"`
		LeaseToken string `json:"lease_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if err := h.service.ValidateLease(r.Context(), in.DeviceID, in.LeaseToken); err != nil {
		writeError(w, http.StatusConflict, leaseErrorCode(err), err.Error())
		return
	}
	write(w, http.StatusOK, map[string]any{"valid": true})
}

func leaseErrorCode(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "lease expired"):
		return "lease_expired"
	case strings.Contains(message, "lease token is invalid"), strings.Contains(message, "lease is "):
		return "lease_invalid"
	case strings.Contains(message, "bound to device"):
		return "lease_device_mismatch"
	default:
		return "run_failed"
	}
}
