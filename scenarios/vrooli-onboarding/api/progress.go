package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	userHomeDirFn   = os.UserHomeDir
	userConfigDirFn = os.UserConfigDir
	completeNowFn   = time.Now
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

type onboardingConfigState struct {
	Onboarding onboardingLifecycleState `json:"onboarding"`
}

type onboardingLifecycleState struct {
	AutoOpen   *bool  `json:"auto_open,omitempty"`
	PromptedAt string `json:"prompted_at,omitempty"`
	Completed  bool   `json:"completed,omitempty"`
	Skipped    bool   `json:"skipped,omitempty"`
}

type onboardingCompleteRequest struct {
	UserID string `json:"user_id"`
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

func (s *Server) handleCompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	var req onboardingCompleteRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	if req.UserID == "" {
		req.UserID = "default"
	}

	completedAt := completeNowFn().UTC().Format(time.RFC3339)
	configPath, err := vrooliConfigPath()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to resolve vrooli config path: " + err.Error(),
		})
		return
	}
	if err := markOnboardingCompleted(configPath, completedAt); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to mark onboarding complete: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":       "ok",
		"user_id":      req.UserID,
		"completed_at": completedAt,
		"config_path":  configPath,
	})
}

func vrooliConfigPath() (string, error) {
	configDir, err := userConfigDirFn()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "vrooli", "config.json"), nil
}

func markOnboardingCompleted(path, completedAt string) error {
	doc := map[string]json.RawMessage{}
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &doc); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	var state onboardingLifecycleState
	if raw, ok := doc["onboarding"]; ok && len(raw) > 0 {
		if err := json.Unmarshal(raw, &state); err != nil {
			return err
		}
	}
	if state.PromptedAt == "" {
		state.PromptedAt = completedAt
	}
	state.Completed = true
	state.Skipped = false

	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	doc["onboarding"] = raw

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
