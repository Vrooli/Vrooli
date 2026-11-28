package agentaccess

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type appMonitorCaptureRequest struct {
	Phase     string `json:"phase"`
	CaptureID string `json:"capture_id,omitempty"`
}

func (h *HTTPHandler) appMonitorCapture(w http.ResponseWriter, r *http.Request) {
	var request appMonitorCaptureRequest
	if !decodeJSON(w, r, &request) {
		return
	}
	request.Phase = strings.TrimSpace(request.Phase)
	request.CaptureID = strings.TrimSpace(request.CaptureID)
	if request.Phase == "begin" {
		if request.CaptureID != "" {
			writeErrorStatus(w, http.StatusBadRequest, errors.New("begin phase cannot select a capture ID"))
			return
		}
	} else if request.Phase != "after_mount" && request.Phase != "after_scroll" && request.Phase != "after_picker" && request.Phase != "finish" {
		writeErrorStatus(w, http.StatusBadRequest, errors.New("unsupported App Monitor capture phase"))
		return
	} else if request.CaptureID == "" {
		writeErrorStatus(w, http.StatusBadRequest, errors.New("capture ID is required after begin"))
		return
	}
	response, err := h.Broker.CaptureAppMonitor(r.Context(), chi.URLParam(r, "sessionID"), request.CaptureID, AppMonitorCapturePhase(request.Phase))
	writeResponse(w, response, err)
}
