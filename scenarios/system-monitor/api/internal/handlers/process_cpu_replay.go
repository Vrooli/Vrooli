package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services"
)

// ProcessCPUReplayHandler exposes a bounded, read-only historical replay of
// the owner-attributed sustained CPU alert rule.
type ProcessCPUReplayHandler struct {
	monitor *services.MonitorService
}

func NewProcessCPUReplayHandler(monitor *services.MonitorService) *ProcessCPUReplayHandler {
	return &ProcessCPUReplayHandler{monitor: monitor}
}

func (h *ProcessCPUReplayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start, err := time.Parse(time.RFC3339Nano, r.URL.Query().Get("start_utc"))
	if err != nil {
		http.Error(w, "start_utc must be an RFC3339 timestamp", http.StatusBadRequest)
		return
	}
	end, err := time.Parse(time.RFC3339Nano, r.URL.Query().Get("end_utc"))
	if err != nil {
		http.Error(w, "end_utc must be an RFC3339 timestamp", http.StatusBadRequest)
		return
	}
	result, err := services.ReplayProcessCPUAlert(r.Context(), h.monitor, start, end)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, services.ErrInvalidProcessCPUReplayWindow) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(result)
}
