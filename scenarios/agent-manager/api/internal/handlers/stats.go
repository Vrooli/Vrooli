// Package handlers provides HTTP handlers for stats endpoints.
package handlers

import (
	"net/http"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// StatsHandler provides HTTP handlers for stats endpoints.
type StatsHandler struct {
	svc orchestration.StatsService
}

// NewStatsHandler creates a new stats handler.
func NewStatsHandler(svc orchestration.StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

// RegisterRoutes registers stats API routes on the given router.
func (h *StatsHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/stats/summary", h.GetSummary).Methods("GET")
	r.HandleFunc("/api/v1/stats/status-distribution", h.GetStatusDistribution).Methods("GET")
	r.HandleFunc("/api/v1/stats/success-rate", h.GetSuccessRate).Methods("GET")
	r.HandleFunc("/api/v1/stats/duration", h.GetDurationStats).Methods("GET")
	r.HandleFunc("/api/v1/stats/cost", h.GetCostStats).Methods("GET")
	r.HandleFunc("/api/v1/stats/runners", h.GetRunnerBreakdown).Methods("GET")
	r.HandleFunc("/api/v1/stats/profiles", h.GetProfileBreakdown).Methods("GET")
	r.HandleFunc("/api/v1/stats/models", h.GetModelBreakdown).Methods("GET")
	r.HandleFunc("/api/v1/stats/tools", h.GetToolUsageStats).Methods("GET")
	r.HandleFunc("/api/v1/stats/errors", h.GetErrorPatterns).Methods("GET")
	r.HandleFunc("/api/v1/stats/time-series", h.GetTimeSeries).Methods("GET")
}

// =============================================================================
// Response Types
// =============================================================================

// SummaryResponse is the HTTP response for the summary endpoint.
type SummaryResponse struct {
	Summary *orchestration.StatsSummary `json:"summary"`
}

// StatusDistributionResponse is the HTTP response for status distribution.
type StatusDistributionResponse struct {
	StatusCounts *repository.RunStatusCounts `json:"statusCounts"`
}

// SuccessRateResponse is the HTTP response for success rate.
type SuccessRateResponse struct {
	SuccessRate float64 `json:"successRate"`
}

// DurationResponse is the HTTP response for duration stats.
type DurationResponse struct {
	Duration *repository.DurationStats `json:"duration"`
}

// CostResponse is the HTTP response for cost stats.
type CostResponse struct {
	Cost *repository.CostStats `json:"cost"`
}

// RunnerBreakdownResponse is the HTTP response for runner breakdown.
type RunnerBreakdownResponse struct {
	Runners []*repository.RunnerBreakdown `json:"runners"`
}

// ProfileBreakdownResponse is the HTTP response for profile breakdown.
type ProfileBreakdownResponse struct {
	Profiles []*repository.ProfileBreakdown `json:"profiles"`
}

// ModelBreakdownResponse is the HTTP response for model breakdown.
type ModelBreakdownResponse struct {
	Models []*repository.ModelBreakdown `json:"models"`
}

// ToolUsageResponse is the HTTP response for tool usage stats.
type ToolUsageResponse struct {
	Tools []*repository.ToolUsageStats `json:"tools"`
}

// ErrorPatternsResponse is the HTTP response for error patterns.
type ErrorPatternsResponse struct {
	Errors []*repository.ErrorPattern `json:"errors"`
}

// TimeSeriesResponse is the HTTP response for time series data.
type TimeSeriesResponse struct {
	Buckets        []*repository.TimeSeriesBucket `json:"buckets"`
	BucketDuration string                         `json:"bucketDuration"`
}

// =============================================================================
// Filter Parsing
// =============================================================================

// parseStatsFilter extracts a StatsFilter from query parameters.
func (h *StatsHandler) parseStatsFilter(r *http.Request) (repository.StatsFilter, error) {
	var filter repository.StatsFilter

	// Parse time window from preset or explicit start/end
	if preset := queryFirst(r, "preset"); preset != "" {
		filter = orchestration.FilterFromPreset(orchestration.TimePreset(preset))
	} else {
		// Parse explicit time range
		startStr := queryFirst(r, "start")
		endStr := queryFirst(r, "end")

		if startStr != "" {
			start, err := time.Parse(time.RFC3339, startStr)
			if err != nil {
				return filter, domain.NewValidationError("start", "invalid RFC3339 timestamp")
			}
			filter.Window.Start = start
		}

		if endStr != "" {
			end, err := time.Parse(time.RFC3339, endStr)
			if err != nil {
				return filter, domain.NewValidationError("end", "invalid RFC3339 timestamp")
			}
			filter.Window.End = end
		}

		// Default to 24h if no time window specified
		if filter.Window.Start.IsZero() && filter.Window.End.IsZero() {
			filter = orchestration.FilterFromPreset(orchestration.TimePreset24H)
		}
	}

	// Parse runner types filter
	if runnerTypesStr := queryFirst(r, "runner_type", "runnerType"); runnerTypesStr != "" {
		runnerType := domain.RunnerType(runnerTypesStr)
		filter.RunnerTypes = []domain.RunnerType{runnerType}
	}

	// Parse profile IDs filter
	if profileIDStr := queryFirst(r, "profile_id", "profileId"); profileIDStr != "" {
		profileID, err := uuid.Parse(profileIDStr)
		if err != nil {
			return filter, domain.NewValidationError("profile_id", "invalid UUID format")
		}
		filter.ProfileIDs = []uuid.UUID{profileID}
	}

	// Parse model filter
	if model := queryFirst(r, "model"); model != "" {
		filter.Models = []string{model}
	}

	// Parse tag prefix filter
	if tagPrefix := queryFirst(r, "tag_prefix", "tagPrefix"); tagPrefix != "" {
		filter.TagPrefix = tagPrefix
	}

	return filter, nil
}

// =============================================================================
// Handlers
// =============================================================================

// GetSummary handles GET /api/v1/stats/summary
func (h *StatsHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	summary, err := h.svc.GetSummary(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, SummaryResponse{Summary: summary})
}

// GetStatusDistribution handles GET /api/v1/stats/status-distribution
func (h *StatsHandler) GetStatusDistribution(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	counts, err := h.svc.GetStatusCounts(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, StatusDistributionResponse{StatusCounts: counts})
}

// GetSuccessRate handles GET /api/v1/stats/success-rate
func (h *StatsHandler) GetSuccessRate(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	rate, err := h.svc.GetSuccessRate(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, SuccessRateResponse{SuccessRate: rate})
}

// GetDurationStats handles GET /api/v1/stats/duration
func (h *StatsHandler) GetDurationStats(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	duration, err := h.svc.GetDurationStats(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, DurationResponse{Duration: duration})
}

// GetCostStats handles GET /api/v1/stats/cost
func (h *StatsHandler) GetCostStats(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	cost, err := h.svc.GetCostStats(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, CostResponse{Cost: cost})
}

// GetRunnerBreakdown handles GET /api/v1/stats/runners
func (h *StatsHandler) GetRunnerBreakdown(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	runners, err := h.svc.GetRunnerBreakdown(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, RunnerBreakdownResponse{Runners: runners})
}

// GetProfileBreakdown handles GET /api/v1/stats/profiles
func (h *StatsHandler) GetProfileBreakdown(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	limit := 10
	if l, ok := parseQueryInt(r, "limit"); ok && l > 0 {
		limit = l
	}

	profiles, err := h.svc.GetProfileBreakdown(r.Context(), filter, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, ProfileBreakdownResponse{Profiles: profiles})
}

// GetModelBreakdown handles GET /api/v1/stats/models
func (h *StatsHandler) GetModelBreakdown(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	limit := 10
	if l, ok := parseQueryInt(r, "limit"); ok && l > 0 {
		limit = l
	}

	models, err := h.svc.GetModelBreakdown(r.Context(), filter, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, ModelBreakdownResponse{Models: models})
}

// GetToolUsageStats handles GET /api/v1/stats/tools
func (h *StatsHandler) GetToolUsageStats(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	limit := 20
	if l, ok := parseQueryInt(r, "limit"); ok && l > 0 {
		limit = l
	}

	tools, err := h.svc.GetToolUsageStats(r.Context(), filter, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, ToolUsageResponse{Tools: tools})
}

// GetErrorPatterns handles GET /api/v1/stats/errors
func (h *StatsHandler) GetErrorPatterns(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	limit := 10
	if l, ok := parseQueryInt(r, "limit"); ok && l > 0 {
		limit = l
	}

	errors, err := h.svc.GetErrorPatterns(r.Context(), filter, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, ErrorPatternsResponse{Errors: errors})
}

// GetTimeSeries handles GET /api/v1/stats/time-series
func (h *StatsHandler) GetTimeSeries(w http.ResponseWriter, r *http.Request) {
	filter, err := h.parseStatsFilter(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Determine bucket duration based on time window
	var bucketDuration time.Duration
	if bucketStr := queryFirst(r, "bucket", "bucketDuration"); bucketStr != "" {
		switch bucketStr {
		case "1h":
			bucketDuration = time.Hour
		case "6h":
			bucketDuration = 6 * time.Hour
		case "1d", "24h":
			bucketDuration = 24 * time.Hour
		default:
			// Try parsing as duration
			parsed, err := time.ParseDuration(bucketStr)
			if err != nil {
				writeSimpleError(w, r, "bucket", "invalid bucket duration")
				return
			}
			bucketDuration = parsed
		}
	} else {
		// Auto-determine bucket size based on time window
		windowDuration := filter.Window.End.Sub(filter.Window.Start)
		switch {
		case windowDuration <= 6*time.Hour:
			bucketDuration = 15 * time.Minute
		case windowDuration <= 24*time.Hour:
			bucketDuration = time.Hour
		case windowDuration <= 7*24*time.Hour:
			bucketDuration = 6 * time.Hour
		default:
			bucketDuration = 24 * time.Hour
		}
	}

	buckets, err := h.svc.GetTimeSeries(r.Context(), filter, bucketDuration)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, TimeSeriesResponse{
		Buckets:        buckets,
		BucketDuration: bucketDuration.String(),
	})
}
