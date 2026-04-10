package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// OnboardingProgress represents a row in the onboarding_progress table.
type OnboardingProgress struct {
	ID             int             `json:"id"`
	UserID         string          `json:"user_id"`
	CurrentStep    int             `json:"current_step"`
	CompletedSteps json.RawMessage `json:"completed_steps"`
	ConfigData     json.RawMessage `json:"config_data"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (s *Server) handleGetProgress(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "default"
	}

	var p OnboardingProgress
	err := s.db.QueryRow(`
		SELECT id, user_id, current_step, completed_steps, config_data, updated_at
		FROM onboarding_progress
		WHERE user_id = $1
	`, userID).Scan(&p.ID, &p.UserID, &p.CurrentStep, &p.CompletedSteps, &p.ConfigData, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "no progress found for user: " + userID,
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "database error: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, p)
}

// progressUpdateRequest is the expected JSON body for PUT /api/v1/progress.
type progressUpdateRequest struct {
	UserID         string          `json:"user_id"`
	CurrentStep    *int            `json:"current_step"`
	CompletedSteps json.RawMessage `json:"completed_steps"`
	ConfigData     json.RawMessage `json:"config_data"`
}

func (s *Server) handleUpdateProgress(w http.ResponseWriter, r *http.Request) {
	var req progressUpdateRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}

	if req.UserID == "" {
		req.UserID = "default"
	}

	// Default to step 0 if not provided
	step := 0
	if req.CurrentStep != nil {
		step = *req.CurrentStep
	}

	// Default empty JSON values
	completedSteps := json.RawMessage("[]")
	if len(req.CompletedSteps) > 0 {
		completedSteps = req.CompletedSteps
	}
	configData := json.RawMessage("{}")
	if len(req.ConfigData) > 0 {
		configData = req.ConfigData
	}

	var p OnboardingProgress
	err := s.db.QueryRow(`
		INSERT INTO onboarding_progress (user_id, current_step, completed_steps, config_data, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			current_step    = EXCLUDED.current_step,
			completed_steps = EXCLUDED.completed_steps,
			config_data     = EXCLUDED.config_data,
			updated_at      = NOW()
		RETURNING id, user_id, current_step, completed_steps, config_data, updated_at
	`, req.UserID, step, completedSteps, configData).Scan(
		&p.ID, &p.UserID, &p.CurrentStep, &p.CompletedSteps, &p.ConfigData, &p.UpdatedAt,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "database error: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, p)
}
