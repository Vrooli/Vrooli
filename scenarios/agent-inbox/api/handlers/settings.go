// Package handlers provides HTTP handlers for the Agent Inbox API.
// This file contains settings-related handlers.
package handlers

import (
	"encoding/json"
	"net/http"
)

// GetYoloMode returns the current YOLO mode setting.
// GET /api/v1/settings/yolo-mode
func (h *Handlers) GetYoloMode(w http.ResponseWriter, r *http.Request) {
	enabled, err := h.Repo.GetYoloMode(r.Context())
	if err != nil {
		h.JSONError(w, "Failed to get YOLO mode", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"enabled": enabled,
	}, http.StatusOK)
}

// SetYoloMode updates the YOLO mode setting.
// POST /api/v1/settings/yolo-mode
// Body: { "enabled": true }
func (h *Handlers) SetYoloMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.JSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.Repo.SetYoloMode(r.Context(), req.Enabled); err != nil {
		h.JSONError(w, "Failed to set YOLO mode", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, map[string]interface{}{
		"enabled": req.Enabled,
		"success": true,
	}, http.StatusOK)
}
