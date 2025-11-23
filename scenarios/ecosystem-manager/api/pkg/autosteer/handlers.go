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
	profileService  *ProfileService
	executionEngine *ExecutionEngine
	historyService  *HistoryService
}

// NewAutoSteerHandlers creates new Auto Steer handlers
func NewAutoSteerHandlers(
	profileService *ProfileService,
	executionEngine *ExecutionEngine,
	historyService *HistoryService,
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusNotFound)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.profileService.UpdateProfile(id, &updates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	profile, err := h.profileService.GetProfile(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

// GetExecutionState handles GET /api/auto-steer/execution/:taskId
func (h *AutoSteerHandlers) GetExecutionState(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	state, err := h.executionEngine.GetExecutionState(taskID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if state == nil {
		http.Error(w, "No execution state found", http.StatusNotFound)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if state == nil {
		http.Error(w, "No execution state found", http.StatusNotFound)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusNotFound)
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
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if feedback.Rating < 1 || feedback.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	if err := h.historyService.SubmitFeedback(executionID, feedback.Rating, feedback.Comments); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}
