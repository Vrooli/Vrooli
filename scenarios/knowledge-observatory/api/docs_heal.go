package main

// DOC: docs/reference/api-endpoints.md#documentation-healing
import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"knowledge-observatory/internal/services/dochealing"
)

type DocHealRequest struct {
	ScenarioName string   `json:"scenario_name,omitempty"`
	Issues       []string `json:"issues,omitempty"`
	AutoApprove  bool     `json:"auto_approve,omitempty"`
	DryRun       bool     `json:"dry_run,omitempty"`
}

type DocHealApproveRequest struct {
	Actor string `json:"actor,omitempty"`
}

type DocHealRejectRequest struct {
	Actor  string `json:"actor,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type DocHealDiff struct {
	Files   []DocHealFileDiff `json:"files"`
	Summary string            `json:"summary"`
}

type DocHealFileDiff struct {
	Path      string `json:"path"`
	Operation string `json:"operation"`
	OldPath   string `json:"old_path,omitempty"`
	Diff      string `json:"diff"`
}

type DocHealJobResponse struct {
	JobID        string       `json:"job_id"`
	ScenarioName string       `json:"scenario_name"`
	Status       string       `json:"status"`
	Progress     string       `json:"progress,omitempty"`
	StartedAt    *string      `json:"started_at,omitempty"`
	CompletedAt  *string      `json:"completed_at,omitempty"`
	Diff         *DocHealDiff `json:"diff,omitempty"`
	HealthBefore *float64     `json:"health_before,omitempty"`
	HealthAfter  *float64     `json:"health_after,omitempty"`
	Error        string       `json:"error,omitempty"`
}

func (s *Server) handleDocsHeal(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docHealingService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation healing service unavailable")
		return
	}
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	if scenarioName == "" {
		s.respondError(w, http.StatusBadRequest, "Scenario name is required")
		return
	}

	var req DocHealRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
			s.respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}
	}
	if req.ScenarioName != "" && req.ScenarioName != scenarioName {
		s.respondError(w, http.StatusBadRequest, "Scenario name mismatch")
		return
	}

	job, err := s.docHealingService.StartHealing(r.Context(), dochealing.HealRequest{
		ScenarioName: scenarioName,
		Issues:       req.Issues,
		AutoApprove:  req.AutoApprove,
		DryRun:       req.DryRun,
	})
	if err != nil {
		respondDocHealError(w, err)
		return
	}
	writeDocHealJob(w, job)
}

func (s *Server) handleDocsHealStatus(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docHealingService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation healing service unavailable")
		return
	}
	vars := mux.Vars(r)
	jobID := vars["job_id"]
	job, err := s.docHealingService.GetJob(r.Context(), jobID)
	if err != nil {
		respondDocHealError(w, err)
		return
	}
	writeDocHealJob(w, job)
}

func (s *Server) handleDocsHealApprove(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docHealingService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation healing service unavailable")
		return
	}
	vars := mux.Vars(r)
	jobID := vars["job_id"]
	var req DocHealApproveRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	job, err := s.docHealingService.ApproveJob(r.Context(), jobID, req.Actor)
	if err != nil {
		respondDocHealError(w, err)
		return
	}
	writeDocHealJob(w, job)
}

func (s *Server) handleDocsHealReject(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.docHealingService == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Documentation healing service unavailable")
		return
	}
	vars := mux.Vars(r)
	jobID := vars["job_id"]
	var req DocHealRejectRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	job, err := s.docHealingService.RejectJob(r.Context(), jobID, req.Actor, req.Reason)
	if err != nil {
		respondDocHealError(w, err)
		return
	}
	writeDocHealJob(w, job)
}

func writeDocHealJob(w http.ResponseWriter, job *dochealing.HealJob) {
	response := DocHealJobResponse{
		JobID:        job.JobID,
		ScenarioName: job.ScenarioName,
		Status:       job.Status,
		Progress:     job.Progress,
		HealthBefore: job.HealthBefore,
		HealthAfter:  job.HealthAfter,
		Error:        job.Error,
	}
	if job.StartedAt != nil {
		value := job.StartedAt.UTC().Format(time.RFC3339)
		response.StartedAt = &value
	}
	if job.CompletedAt != nil {
		value := job.CompletedAt.UTC().Format(time.RFC3339)
		response.CompletedAt = &value
	}
	if job.Diff != nil {
		files := make([]DocHealFileDiff, 0, len(job.Diff.Files))
		for _, file := range job.Diff.Files {
			files = append(files, DocHealFileDiff{
				Path:      file.Path,
				Operation: file.Operation,
				OldPath:   file.OldPath,
				Diff:      file.Diff,
			})
		}
		response.Diff = &DocHealDiff{
			Files:   files,
			Summary: job.Diff.Summary,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func respondDocHealError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, dochealing.ErrScenarioRequired),
		errors.Is(err, dochealing.ErrScenarioNotFound),
		errors.Is(err, dochealing.ErrJobIDRequired),
		errors.Is(err, dochealing.ErrJobNotReady),
		errors.Is(err, dochealing.ErrJobNotApprovable),
		errors.Is(err, dochealing.ErrJobNotRejectable):
		respondWithError(w, http.StatusBadRequest, err)
	case errors.Is(err, dochealing.ErrJobNotFound):
		respondWithError(w, http.StatusNotFound, err)
	case errors.Is(err, dochealing.ErrScenarioRootEmpty),
		errors.Is(err, dochealing.ErrAgentUnavailable),
		errors.Is(err, dochealing.ErrJobStoreUnavailable),
		errors.Is(err, dochealing.ErrHealthUnavailable):
		respondWithError(w, http.StatusServiceUnavailable, err)
	default:
		respondWithError(w, http.StatusInternalServerError, err)
	}
}
