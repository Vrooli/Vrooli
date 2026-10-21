package creation

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"device-studio/internal/jobs"
	"device-studio/internal/module"
	"device-studio/internal/renderjobs"
	"github.com/gorilla/mux"
)

type coordinator interface {
	SubmitGeometryCreation(context.Context, renderjobs.GeometryCreationRequest) (jobs.Job, bool, error)
	FindGeometryCreationByKey(context.Context, string, string) (jobs.Job, error)
	OpenGeometryCorrection(context.Context, string) (renderjobs.GeometryCorrectionDetails, error)
	Cancel(context.Context, string, int64) (jobs.Job, error)
}
type jobReader interface {
	Get(context.Context, string) (jobs.Job, error)
}
type RouteActivator interface{ ActivateTestRoute(context.Context) error }
type createBody struct {
	CandidateID            string     `json:"candidate_id"`
	IdempotencyKey         string     `json:"idempotency_key"`
	Brief                  string     `json:"brief"`
	EnvelopeMM             [3]float64 `json:"envelope_mm"`
	ReferenceMediaIDs      []string   `json:"reference_media_ids"`
	FundingRoute           string     `json:"funding_route,omitempty"`
	ConfirmLocalAI         bool       `json:"confirm_local_ai"`
	ConfirmRemoteAI        bool       `json:"confirm_remote_ai"`
	CorrectionSourceJobID  string     `json:"correction_source_job_id,omitempty"`
	CorrectionRecipeSHA256 string     `json:"correction_recipe_sha256,omitempty"`
	CorrectionFeedback     string     `json:"correction_feedback,omitempty"`
}
type cancelBody struct {
	Version int64 `json:"version"`
}
type jobProjection struct {
	JobID        string     `json:"job_id"`
	ProjectID    string     `json:"project_id"`
	CandidateID  string     `json:"candidate_id"`
	State        jobs.State `json:"state"`
	Version      int64      `json:"version"`
	ReviewStatus string     `json:"review_status"`
	Message      string     `json:"message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func Module(c coordinator, j jobReader) module.Module {
	return ModuleWithCoordinator(c, j, nil)
}

func ModuleWithCoordinator(c coordinator, j jobReader, activator RouteActivator) module.Module {
	return module.Module{Name: "creation", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/projects/{project_id}/geometry-creations", createHandler(c, activator)).Methods(http.MethodPost)
		r.HandleFunc("/api/v1/projects/{project_id}/geometry-creations/operations/{operation_key}", operationHandler(c)).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/geometry-creations/{job_id}", statusHandler(j)).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/geometry-creations/{job_id}/correction", correctionHandler(c)).Methods(http.MethodGet)
		r.HandleFunc("/api/v1/geometry-creations/{job_id}/cancel", cancelHandler(c, j, activator)).Methods(http.MethodPost)
	}}
}

func correctionHandler(c coordinator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		details, err := c.OpenGeometryCorrection(r.Context(), strings.TrimSpace(mux.Vars(r)["job_id"]))
		if err != nil {
			status := http.StatusConflict
			if errors.Is(err, jobs.ErrNotFound) {
				status = http.StatusNotFound
			}
			writeJSON(w, status, map[string]string{"error": "verified correction details are unavailable for this job state"})
			return
		}
		writeJSON(w, http.StatusOK, details)
	}
}

func operationHandler(c coordinator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, err := c.FindGeometryCreationByKey(r.Context(), strings.TrimSpace(mux.Vars(r)["project_id"]), strings.TrimSpace(mux.Vars(r)["operation_key"]))
		if err != nil || job.Input.SubjectKind != jobs.SubjectGeometryCreate {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "geometry creation operation not found"})
			return
		}
		writeJSON(w, http.StatusOK, project(job, ""))
	}
}

func createHandler(c coordinator, activator RouteActivator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := strings.TrimSpace(mux.Vars(r)["project_id"])
		var body createBody
		if err := decodeBody(w, r, &body, 16<<10); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid creation request"})
			return
		}
		fundingRoute := renderjobs.NormalizeGeometryFundingRoute(body.FundingRoute)
		if fundingRoute == "" {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "select a supported AI funding route"})
			return
		}
		if fundingRoute == renderjobs.GeometryFundingLocal && !body.ConfirmLocalAI {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "confirm the disclosed local-only AI request before starting"})
			return
		}
		if fundingRoute != renderjobs.GeometryFundingLocal && !body.ConfirmRemoteAI {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "confirm that the selected images and brief will be sent to the chosen remote provider"})
			return
		}
		job, created, err := c.SubmitGeometryCreation(r.Context(), renderjobs.GeometryCreationRequest{ProjectID: projectID, CandidateID: body.CandidateID, IdempotencyKey: body.IdempotencyKey, Brief: body.Brief, EnvelopeMM: body.EnvelopeMM, ReferenceMediaIDs: body.ReferenceMediaIDs, FundingRoute: fundingRoute, CorrectionSourceJobID: body.CorrectionSourceJobID, CorrectionRecipeSHA256: body.CorrectionRecipeSHA256, CorrectionFeedback: body.CorrectionFeedback})
		if err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": safeCreationError(err)})
			return
		}
		if activator != nil {
			if err := activator.ActivateTestRoute(r.Context()); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, project(job, "The request was accepted but its routed progress context could not be confirmed. Retry only with the same request identity."))
				return
			}
		}
		status := http.StatusAccepted
		message := "AI request queued. The result remains unreviewed."
		if fundingRoute == renderjobs.GeometryFundingLocal {
			message = "Local-only AI request queued. The result remains unreviewed."
		} else if fundingRoute == renderjobs.GeometryFundingBYOK {
			message = "BYOK-only AI request queued; no other funding route will be used. The result remains unreviewed."
		} else {
			message = "Vrooli-credits-only AI request queued; no other funding route will be used. The result remains unreviewed."
		}
		if !created {
			status = http.StatusOK
			message = "Existing request state: " + string(job.State) + "; no second operation was created."
		}
		writeJSON(w, status, project(job, message))
	}
}
func statusHandler(j jobReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		job, err := j.Get(r.Context(), mux.Vars(r)["job_id"])
		if err != nil || job.Input.SubjectKind != jobs.SubjectGeometryCreate {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "geometry creation not found"})
			return
		}
		writeJSON(w, http.StatusOK, project(job, ""))
	}
}
func cancelHandler(c coordinator, j jobReader, activator RouteActivator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["job_id"]
		job, err := j.Get(r.Context(), id)
		if err != nil || job.Input.SubjectKind != jobs.SubjectGeometryCreate {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "geometry creation not found"})
			return
		}
		var body cancelBody
		if err := decodeBody(w, r, &body, 1024); err != nil || body.Version <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "a current job version is required"})
			return
		}
		job, err = c.Cancel(r.Context(), id, body.Version)
		if err != nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "creation cancellation could not be confirmed"})
			return
		}
		if activator != nil {
			if err := activator.ActivateTestRoute(r.Context()); err != nil {
				writeJSON(w, http.StatusServiceUnavailable, project(job, "Cancellation was recorded but its routed reconciliation context could not be confirmed."))
				return
			}
		}
		writeJSON(w, http.StatusAccepted, project(job, "Cancellation requested; an already submitted local call may finish."))
	}
}
func decodeBody(w http.ResponseWriter, r *http.Request, target any, max int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, max)
	defer r.Body.Close()
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("trailing request data")
	}
	return nil
}
func project(job jobs.Job, message string) jobProjection {
	return jobProjection{JobID: job.ID, ProjectID: job.Input.ProjectID, CandidateID: job.Input.SubjectID, State: job.State, Version: job.Version, ReviewStatus: "unreviewed", Message: message, CreatedAt: job.CreatedAt, UpdatedAt: job.UpdatedAt}
}
func safeCreationError(err error) string {
	if err == nil {
		return "creation request was refused"
	}
	switch {
	case errors.Is(err, jobs.ErrIdempotencyConflict):
		return "this request key already belongs to different immutable inputs"
	case strings.Contains(err.Error(), "reference"):
		return "selected project references are unavailable or outside the supported image limits"
	case strings.Contains(err.Error(), "dimensions"):
		return "locked dimensions are outside the supported range"
	default:
		return "local geometry creation is unavailable for this request"
	}
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
