package handlers

import (
	"net/http"
	"strings"
)

type seedCleanupScheduleRequest struct {
	CleanupToken string `json:"cleanup_token"`
	SeedScenario string `json:"seed_scenario,omitempty"`
}

// ScheduleExecutionSeedCleanup handles POST /api/v1/executions/{id}/seed-cleanup.
func (h *Handler) ScheduleExecutionSeedCleanup(w http.ResponseWriter, r *http.Request) {
	executionID, ok := h.parseUUIDParam(w, r, "id", ErrInvalidExecutionID)
	if !ok {
		return
	}

	var req seedCleanupScheduleRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		h.respondError(w, ErrInvalidRequest.WithDetails(map[string]string{"error": err.Error()}))
		return
	}

	cleanupToken := strings.TrimSpace(req.CleanupToken)
	if cleanupToken == "" {
		h.respondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "cleanup_token"}))
		return
	}

	seedScenario := strings.TrimSpace(req.SeedScenario)
	if seedScenario == "" {
		seedScenario = "browser-automation-studio"
	}

	if h.seedCleanupManager == nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":         "seed cleanup manager unavailable",
			"execution_id":  executionID.String(),
			"seed_scenario": seedScenario,
		}))
		return
	}

	if err := h.seedCleanupManager.Schedule(executionID.String(), seedScenario, cleanupToken); err != nil {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error":         "seed cleanup scheduling failed",
			"details":       err.Error(),
			"execution_id":  executionID.String(),
			"seed_scenario": seedScenario,
		}))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"scheduled"}`))
}
