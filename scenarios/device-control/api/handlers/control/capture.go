package control

import (
	"encoding/json"
	"net/http"

	internal "device-control/internal/control"
	"github.com/gorilla/mux"
)

func (h *handler) captureDevice(w http.ResponseWriter, r *http.Request) {
	var in internal.CaptureRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "capture request is invalid")
		return
	}
	in.Device = mux.Vars(r)["id"]
	result, err := h.service.Capture(r.Context(), in)
	if err != nil {
		write(w, http.StatusConflict, map[string]any{"result": result, "error": map[string]any{"code": "capture_failed", "message": err.Error()}})
		return
	}
	write(w, http.StatusOK, result)
}

func (h *handler) launchApp(w http.ResponseWriter, r *http.Request) {
	var in internal.LaunchAppRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "launch request is invalid")
		return
	}
	in.Device = mux.Vars(r)["id"]
	result, err := h.service.LaunchApp(r.Context(), in)
	if err != nil {
		write(w, http.StatusConflict, map[string]any{"result": result, "error": map[string]any{"code": "app_launch_failed", "message": err.Error()}})
		return
	}
	write(w, http.StatusOK, result)
}

func (h *handler) screenDevice(w http.ResponseWriter, r *http.Request) {
	var in internal.ScreenRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "screen request is invalid")
		return
	}
	in.Device = mux.Vars(r)["id"]
	result, err := h.service.Screen(r.Context(), in)
	if err != nil {
		write(w, http.StatusConflict, map[string]any{"result": result, "error": map[string]any{"code": "screen_operation_failed", "message": err.Error()}})
		return
	}
	write(w, http.StatusOK, result)
}

func (h *handler) installApp(w http.ResponseWriter, r *http.Request) {
	var in internal.InstallAppRequest
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "install request is invalid")
		return
	}
	in.Device = mux.Vars(r)["id"]
	result, err := h.service.InstallApp(r.Context(), in)
	if err != nil {
		write(w, http.StatusConflict, map[string]any{"result": result, "error": map[string]any{"code": "app_install_failed", "message": err.Error()}})
		return
	}
	write(w, http.StatusOK, result)
}
