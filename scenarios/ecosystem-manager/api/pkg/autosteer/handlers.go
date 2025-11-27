package autosteer

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// ProfileServiceAPI defines the profile operations used by HTTP handlers.
type ProfileServiceAPI interface {
	CreateProfile(profile *AutoSteerProfile) error
	ListProfiles(tags []string) ([]*AutoSteerProfile, error)
	GetProfile(id string) (*AutoSteerProfile, error)
	UpdateProfile(id string, updates *AutoSteerProfile) error
	DeleteProfile(id string) error
	GetTemplates() []*AutoSteerProfile
}

// ExecutionEngineAPI defines execution control operations used by HTTP handlers.
type ExecutionEngineAPI interface {
	StartExecution(taskID, profileID, scenarioName string) (*ProfileExecutionState, error)
	EvaluateIteration(taskID, scenarioName string) (*IterationEvaluation, error)
	DeleteExecutionState(taskID string) error
	SeekExecution(taskID, profileID, scenarioName string, phaseIndex, phaseIteration int) (*ProfileExecutionState, error)
	AdvancePhase(taskID, scenarioName string) (*PhaseAdvanceResult, error)
	GetExecutionState(taskID string) (*ProfileExecutionState, error)
	GetCurrentMode(taskID string) (SteerMode, error)
}

// HistoryServiceAPI defines history operations used by HTTP handlers.
type HistoryServiceAPI interface {
	GetHistory(filters HistoryFilters) ([]ProfilePerformance, error)
	GetExecution(executionID string) (*ProfilePerformance, error)
	SubmitFeedback(executionID string, rating int, comments string) error
	GetProfileAnalytics(profileID string) (*ProfileAnalytics, error)
}

// writeError writes a structured JSON error response
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    statusCode,
	})
}

// writeJSON writes a successful JSON response
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// AutoSteerHandlers handles HTTP requests for Auto Steer functionality
type AutoSteerHandlers struct {
	profileService  ProfileServiceAPI
	executionEngine ExecutionEngineAPI
	historyService  HistoryServiceAPI
}

// NewAutoSteerHandlers creates new Auto Steer handlers
func NewAutoSteerHandlers(
	profileService ProfileServiceAPI,
	executionEngine ExecutionEngineAPI,
	historyService HistoryServiceAPI,
) *AutoSteerHandlers {
	return &AutoSteerHandlers{
		profileService:  profileService,
		executionEngine: executionEngine,
		historyService:  historyService,
	}
}

// CreateProfile handles POST /api/auto-steer/profiles
func (h *AutoSteerHandlers) CreateProfile(w http.ResponseWriter, r *http.Request) {
	var profile AutoSteerProfile

	if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.profileService.CreateProfile(&profile); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create profile: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, profile)
}

// ListProfiles handles GET /api/auto-steer/profiles
func (h *AutoSteerHandlers) ListProfiles(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	tags := r.URL.Query()["tag"]

	profiles, err := h.profileService.ListProfiles(tags)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list profiles: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profiles)
}

// GetProfile handles GET /api/auto-steer/profiles/:id
func (h *AutoSteerHandlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	profile, err := h.profileService.GetProfile(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// UpdateProfile handles PUT /api/auto-steer/profiles/:id
func (h *AutoSteerHandlers) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var updates AutoSteerProfile
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if err := h.profileService.UpdateProfile(id, &updates); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update profile: "+err.Error())
		return
	}

	profile, err := h.profileService.GetProfile(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load updated profile: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profile)
}

// DeleteProfile handles DELETE /api/auto-steer/profiles/:id
func (h *AutoSteerHandlers) DeleteProfile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := h.profileService.DeleteProfile(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete profile: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTemplates handles GET /api/auto-steer/templates
func (h *AutoSteerHandlers) GetTemplates(w http.ResponseWriter, r *http.Request) {
	templates := h.profileService.GetTemplates()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// StartExecution handles POST /api/auto-steer/execution/start
func (h *AutoSteerHandlers) StartExecution(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID       string `json:"task_id"`
		ProfileID    string `json:"profile_id"`
		ScenarioName string `json:"scenario_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.TaskID == "" || req.ProfileID == "" || req.ScenarioName == "" {
		writeError(w, http.StatusBadRequest, "task_id, profile_id, and scenario_name are required")
		return
	}

	state, err := h.executionEngine.StartExecution(req.TaskID, req.ProfileID, req.ScenarioName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to start execution: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, state)
}

// EvaluateIteration handles POST /api/auto-steer/execution/evaluate
func (h *AutoSteerHandlers) EvaluateIteration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID       string `json:"task_id"`
		ScenarioName string `json:"scenario_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.TaskID == "" || req.ScenarioName == "" {
		writeError(w, http.StatusBadRequest, "task_id and scenario_name are required")
		return
	}

	result, err := h.executionEngine.EvaluateIteration(req.TaskID, req.ScenarioName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to evaluate iteration: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// ResetExecution handles POST /api/auto-steer/execution/reset
func (h *AutoSteerHandlers) ResetExecution(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID string `json:"task_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.TaskID == "" {
		writeError(w, http.StatusBadRequest, "task_id is required")
		return
	}

	if h.executionEngine == nil {
		writeError(w, http.StatusServiceUnavailable, "Auto Steer execution engine unavailable")
		return
	}

	if err := h.executionEngine.DeleteExecutionState(req.TaskID); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reset Auto Steer state: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Auto Steer state reset",
	})
}

// SeekExecution handles POST /api/auto-steer/execution/seek
// Allows operators to jump to a specific phase/iteration without restarting.
// If no execution state exists, profile_id and scenario_name can be provided to initialize it first.
func (h *AutoSteerHandlers) SeekExecution(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID         string `json:"task_id"`
		ProfileID      string `json:"profile_id,omitempty"`       // Optional: for initialization if no state exists
		ScenarioName   string `json:"scenario_name,omitempty"`    // Optional: for initialization if no state exists
		PhaseIndex     int    `json:"phase_index"`
		PhaseIteration int    `json:"phase_iteration"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.TaskID == "" {
		writeError(w, http.StatusBadRequest, "task_id is required")
		return
	}

	state, err := h.executionEngine.SeekExecution(req.TaskID, req.ProfileID, req.ScenarioName, req.PhaseIndex, req.PhaseIteration)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to seek execution: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, state)
}

// AdvancePhase handles POST /api/auto-steer/execution/advance
func (h *AutoSteerHandlers) AdvancePhase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TaskID       string `json:"task_id"`
		ScenarioName string `json:"scenario_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.TaskID == "" || req.ScenarioName == "" {
		writeError(w, http.StatusBadRequest, "task_id and scenario_name are required")
		return
	}

	result, err := h.executionEngine.AdvancePhase(req.TaskID, req.ScenarioName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to advance phase: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// GetExecutionState handles GET /api/auto-steer/execution/:taskId
func (h *AutoSteerHandlers) GetExecutionState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	state, err := h.executionEngine.GetExecutionState(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load execution state: "+err.Error())
		return
	}

	if state == nil {
		writeError(w, http.StatusNotFound, "No execution state found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

// GetMetrics handles GET /api/auto-steer/metrics/:taskId
func (h *AutoSteerHandlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	state, err := h.executionEngine.GetExecutionState(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load execution state: "+err.Error())
		return
	}

	if state == nil {
		writeError(w, http.StatusNotFound, "No execution state found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state.Metrics)
}

// GetHistory handles GET /api/auto-steer/history
func (h *AutoSteerHandlers) GetHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	profileID := r.URL.Query().Get("profile_id")
	scenario := r.URL.Query().Get("scenario")

	filters := HistoryFilters{
		ProfileID:    profileID,
		ScenarioName: scenario,
	}

	history, err := h.historyService.GetHistory(filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load history: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// GetExecution handles GET /api/auto-steer/history/:executionId
func (h *AutoSteerHandlers) GetExecution(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["executionId"]

	execution, err := h.historyService.GetExecution(executionID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// SubmitFeedback handles POST /api/auto-steer/history/:executionId/feedback
func (h *AutoSteerHandlers) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["executionId"]

	var feedback struct {
		Rating   int    `json:"rating"`
		Comments string `json:"comments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if feedback.Rating < 1 || feedback.Rating > 5 {
		writeError(w, http.StatusBadRequest, "Rating must be between 1 and 5")
		return
	}

	if err := h.historyService.SubmitFeedback(executionID, feedback.Rating, feedback.Comments); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to submit feedback: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetProfileAnalytics handles GET /api/auto-steer/analytics/:profileId
func (h *AutoSteerHandlers) GetProfileAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	profileID := vars["profileId"]

	analytics, err := h.historyService.GetProfileAnalytics(profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load profile analytics: "+err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}
