// DOC: docs/concepts/ARCHITECTURE.md
// DOC: docs/reference/api-endpoints.md
package autosteer

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ecosystem-manager/api/pkg/effectiveness"
	"github.com/gorilla/mux"
	"github.com/vrooli/maturity-go/dimensions"
)

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// ProfileServiceAPI defines the profile operations used by HTTP handlers.
type ProfileServiceAPI interface {
	ProfileRepository
	GetTemplates() []*AutoSteerProfile
}

// ExecutionEngineAPI defines controller operations used by HTTP handlers.
type ExecutionEngineAPI interface {
	StartExecution(taskID, profileID, scenarioName string) (*ProfileExecutionState, error)
	EvaluateIteration(taskID, scenarioName string) (*IterationEvaluation, error)
	DeleteExecutionState(taskID string) error
	GetExecutionState(taskID string) (*ProfileExecutionState, error)
	GetCurrentSet(taskID string) ([]string, error)
	GetDecisionTrace(taskID string) ([]DecisionTraceEntry, error)
	Effectiveness(skillID string, dim dimensions.Dimension) ([]effectiveness.Stat, error)
	EffectivenessPrior() (prior, k float64)
}

// HistoryServiceAPI defines history operations used by HTTP handlers.
type HistoryServiceAPI interface {
	GetHistory(filters HistoryFilters) ([]ProfilePerformance, error)
	GetExecution(executionID string) (*ProfilePerformance, error)
	SubmitFeedback(executionID string, rating int, comments string) error
	SubmitFeedbackEntry(executionID string, req ExecutionFeedbackRequest) (*ExecutionFeedbackEntry, error)
	GetProfileAnalytics(profileID string) (*ProfileAnalytics, error)
}

// writeError writes a structured JSON error response
func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    statusCode,
	}); err != nil {
		_ = err // Response already has headers; best-effort only.
	}
}

// writeJSON writes a successful JSON response
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		_ = err // Response already has headers; best-effort only.
	}
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

	writeJSON(w, http.StatusOK, map[string]any{
		"profiles": profiles,
		"count":    len(profiles),
	})
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

	writeJSON(w, http.StatusOK, profile)
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

	writeJSON(w, http.StatusOK, profile)
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

	writeJSON(w, http.StatusOK, map[string]any{
		"templates": templates,
		"count":     len(templates),
	})
}

// dimensionInfo is the wire shape for one canonical improvement dimension.
type dimensionInfo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

// GetDimensions handles GET /api/auto-steer/dimensions. It serves the canonical
// dimension vocabulary (SSOT) so the profile editor's weight controls never
// drift from the controller's actual vocabulary.
func (h *AutoSteerHandlers) GetDimensions(w http.ResponseWriter, _ *http.Request) {
	all := dimensions.All()
	out := make([]dimensionInfo, 0, len(all))
	for _, d := range all {
		out = append(out, dimensionInfo{ID: string(d), Description: dimensions.Describe(d)})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"dimensions": out,
		"count":      len(out),
	})
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

// GetDecisionTrace handles GET /api/auto-steer/execution/:taskId/trace.
// Returns the controller's per-iteration decision trace (state → choice →
// rationale → realized delta) for the glass-box UI.
func (h *AutoSteerHandlers) GetDecisionTrace(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskID := vars["taskId"]

	trace, err := h.executionEngine.GetDecisionTrace(taskID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load decision trace: "+err.Error())
		return
	}
	if trace == nil {
		trace = []DecisionTraceEntry{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"task_id": taskID,
		"trace":   trace,
		"count":   len(trace),
	})
}

// effectivenessRow is the API projection of one ledger row, with the derived
// efficacy the selector would use computed alongside the authoritative raw
// counts.
type effectivenessRow struct {
	SkillID                 string    `json:"skill_id"`
	Dimension               string    `json:"dimension"`
	ClosedCount             int64     `json:"closed_count"`
	IntroducedCount         int64     `json:"introduced_count"`
	NetClosed               int64     `json:"net_closed"`
	TotalRuns               int64     `json:"total_runs"`
	TotalTokens             int64     `json:"total_tokens"`
	AvgTokensPerRun         int64     `json:"avg_tokens_per_run"`
	ObservedEfficacyPerKtok float64   `json:"observed_efficacy_per_ktok"`
	ExpectedEfficacyPerKtok float64   `json:"expected_efficacy_per_ktok"`
	LastRunAt               time.Time `json:"last_run_at"`
}

// GetEffectiveness handles GET /api/auto-steer/effectiveness?skill=&dimension=.
// It dumps the per-(skill, dimension) effectiveness ledger — the operator's
// "which skills actually work" view — with the derived efficacy estimate.
func (h *AutoSteerHandlers) GetEffectiveness(w http.ResponseWriter, r *http.Request) {
	skill := r.URL.Query().Get("skill")
	dim := dimensions.Dimension(r.URL.Query().Get("dimension"))

	stats, err := h.executionEngine.Effectiveness(skill, dim)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to load effectiveness ledger: "+err.Error())
		return
	}
	prior, k := h.executionEngine.EffectivenessPrior()

	rows := make([]effectivenessRow, 0, len(stats))
	for _, s := range stats {
		avg := int64(0)
		if s.TotalRuns > 0 {
			avg = s.TotalTokens / s.TotalRuns
		}
		rows = append(rows, effectivenessRow{
			SkillID:                 s.SkillID,
			Dimension:               string(s.Dimension),
			ClosedCount:             s.ClosedCount,
			IntroducedCount:         s.IntroducedCount,
			NetClosed:               s.NetClosed(),
			TotalRuns:               s.TotalRuns,
			TotalTokens:             s.TotalTokens,
			AvgTokensPerRun:         avg,
			ObservedEfficacyPerKtok: s.ObservedEfficacyPerToken(),
			ExpectedEfficacyPerKtok: s.ExpectedEfficacyPerToken(prior, k),
			LastRunAt:               s.LastRunAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"effectiveness": rows,
		"count":         len(rows),
		"prior":         prior,
		"shrinkage_k":   k,
	})
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

	writeJSON(w, http.StatusOK, state)
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

	writeJSON(w, http.StatusOK, state.Metrics)
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

	writeJSON(w, http.StatusOK, history)
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

	writeJSON(w, http.StatusOK, execution)
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

// SubmitFeedbackEntry records structured feedback for an execution.
func (h *AutoSteerHandlers) SubmitFeedbackEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	executionID := vars["executionId"]

	var req ExecutionFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid feedback payload: "+err.Error())
		return
	}
	entry, err := h.historyService.SubmitFeedbackEntry(executionID, req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to submit feedback entry: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, entry)
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

	writeJSON(w, http.StatusOK, analytics)
}
