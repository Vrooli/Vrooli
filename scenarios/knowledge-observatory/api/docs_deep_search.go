package main

// DOC: docs/reference/api-endpoints.md#documentation-deep-search
import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/services/deepsearch"
)

// DeepSearchRequest defines the input schema for deep search.
type DeepSearchRequest struct {
	Query          string `json:"query"`
	Scope          string `json:"scope,omitempty"`
	Scenario       string `json:"scenario,omitempty"`
	BasePath       string `json:"base_path,omitempty"`
	MaxResults     int    `json:"max_results,omitempty"`
	FollowRefs     *bool  `json:"follow_refs,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

// DeepSearchJobResponse represents a deep search job status response.
type DeepSearchJobResponse struct {
	JobID       string                        `json:"job_id"`
	Status      string                        `json:"status"`
	Progress    string                        `json:"progress,omitempty"`
	StartedAt   *string                       `json:"started_at,omitempty"`
	CompletedAt *string                       `json:"completed_at,omitempty"`
	Results     []deepsearch.DeepSearchResult `json:"results,omitempty"`
	Error       string                        `json:"error,omitempty"`
}

func (s *Server) handleDocsSearchDeep(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docDeepSearchService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Deep search service unavailable")
		return
	}
	var req DeepSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	followRefs := true
	if req.FollowRefs != nil {
		followRefs = *req.FollowRefs
	}
	job, err := s.docDeepSearchService.StartSearch(r.Context(), deepsearch.DeepSearchRequest{
		Query:          req.Query,
		Scope:          req.Scope,
		Scenario:       req.Scenario,
		BasePath:       req.BasePath,
		MaxResults:     req.MaxResults,
		FollowRefs:     followRefs,
		TimeoutSeconds: req.TimeoutSeconds,
	})
	if err != nil {
		respondDeepSearchError(w, err)
		return
	}
	writeDeepSearchJob(w, job)
}

func (s *Server) handleDocsSearchDeepStatus(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docDeepSearchService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Deep search service unavailable")
		return
	}
	vars := mux.Vars(r)
	jobID := vars["job_id"]
	job, err := s.docDeepSearchService.GetJob(r.Context(), jobID)
	if err != nil {
		respondDeepSearchError(w, err)
		return
	}
	writeDeepSearchJob(w, job)
}

func writeDeepSearchJob(w http.ResponseWriter, job *deepsearch.DeepSearchJob) {
	response := DeepSearchJobResponse{
		JobID:    job.JobID,
		Status:   job.Status,
		Progress: job.Progress,
		Results:  job.Results,
		Error:    job.Error,
	}
	if job.StartedAt != nil {
		value := job.StartedAt.UTC().Format(time.RFC3339)
		response.StartedAt = &value
	}
	if job.CompletedAt != nil {
		value := job.CompletedAt.UTC().Format(time.RFC3339)
		response.CompletedAt = &value
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func respondDeepSearchError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, deepsearch.ErrQueryRequired),
		errors.Is(err, deepsearch.ErrScopeInvalid),
		errors.Is(err, deepsearch.ErrScenarioRequired),
		errors.Is(err, deepsearch.ErrBasePathRequired),
		errors.Is(err, deepsearch.ErrBasePathInvalid),
		errors.Is(err, deepsearch.ErrJobIDRequired):
		respondWithError(w, http.StatusBadRequest, err)
	case errors.Is(err, deepsearch.ErrJobNotFound):
		respondWithError(w, http.StatusNotFound, err)
	case errors.Is(err, deepsearch.ErrScenarioRootEmpty),
		errors.Is(err, deepsearch.ErrAgentUnavailable),
		errors.Is(err, deepsearch.ErrJobStoreUnavailable):
		respondWithError(w, http.StatusServiceUnavailable, err)
	default:
		respondWithError(w, http.StatusInternalServerError, err)
	}
}
