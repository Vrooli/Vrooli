package agentmanager

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

// DOC: docs/concepts/ARCHITECTURE.md#api-boundaries
// DOC: docs/internal/SEAMS.md

// Handler exposes agent-manager status endpoints.
type Handler struct {
	service Service
}

// NewHandler creates a new status handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers agent-manager endpoints.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/agent-manager/status", h.Status).Methods("GET")
	r.HandleFunc("/api/v1/agent-manager/runs/{runID}", h.GetRun).Methods("GET")
	r.HandleFunc("/api/v1/agent-manager/runs/{runID}/stop", h.StopRun).Methods("POST")
}

// Status returns agent-manager availability.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	response := &apipb.AgentManagerStatusResponse{}
	if h.service != nil {
		response.Enabled = h.service.IsEnabled()
		if response.Enabled {
			response.Available = h.service.IsAvailable(r.Context())
			if url, err := h.service.ResolveURL(r.Context()); err == nil && url != "" {
				response.Url = &url
			}
		}
		if profileID := h.service.GetProfileID(); profileID != "" {
			response.ProfileId = &profileID
		}
	}

	if err := httputil.ProtoJSON(w, response); err != nil {
		apierr.MapError(w, "[agent-manager] status", apierr.Internal("failed to encode response"))
	}
}

type runStatusResponse struct {
	RunID           string  `json:"run_id"`
	TaskID          string  `json:"task_id,omitempty"`
	Status          string  `json:"status"`
	StartedAt       string  `json:"started_at,omitempty"`
	FinishedAt      string  `json:"finished_at,omitempty"`
	ErrorMessage    string  `json:"error_message,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	Active          bool    `json:"active"`
}

type stopRunResponse struct {
	RunID   string `json:"run_id"`
	Stopped bool   `json:"stopped"`
	Status  string `json:"status"`
}

// GetRun returns agent-manager lifecycle state for a run.
func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	if h.service == nil || !h.service.IsEnabled() {
		apierr.MapError(w, "[agent-manager] run", apierr.Unavailable("agent-manager is not available"))
		return
	}

	runID := strings.TrimSpace(mux.Vars(r)["runID"])
	if runID == "" {
		apierr.MapError(w, "[agent-manager] run", apierr.BadRequest("run_id is required"))
		return
	}

	state, err := h.service.GetRunState(r.Context(), runID)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotAvailable):
			apierr.MapError(w, "[agent-manager] run", apierr.Unavailable("agent-manager is not available"))
		case strings.Contains(err.Error(), "status 404"):
			apierr.MapError(w, "[agent-manager] run", apierr.NotFound("run not found"))
		case errors.Is(err, ErrRequestFailed):
			apierr.MapError(w, "[agent-manager] run", apierr.BadGateway("failed to load run status"))
		default:
			apierr.MapError(w, "[agent-manager] run", apierr.Internal("failed to load run status"))
		}
		return
	}

	response := runStatusResponse{
		RunID:        state.RunID,
		TaskID:       state.TaskID,
		Status:       state.Status,
		StartedAt:    state.StartedAt,
		FinishedAt:   state.FinishedAt,
		ErrorMessage: state.ErrorMsg,
		Active:       isActiveStatus(state.Status),
	}
	if duration := durationSeconds(state.StartedAt, state.FinishedAt); duration > 0 {
		response.DurationSeconds = duration
	}
	if err := httputil.JSON(w, response); err != nil {
		apierr.MapError(w, "[agent-manager] run", apierr.Internal("failed to encode response"))
	}
}

// StopRun requests cancellation for a run.
func (h *Handler) StopRun(w http.ResponseWriter, r *http.Request) {
	if h.service == nil || !h.service.IsEnabled() {
		apierr.MapError(w, "[agent-manager] run stop", apierr.Unavailable("agent-manager is not available"))
		return
	}

	runID := strings.TrimSpace(mux.Vars(r)["runID"])
	if runID == "" {
		apierr.MapError(w, "[agent-manager] run stop", apierr.BadRequest("run_id is required"))
		return
	}
	if err := h.service.StopRun(r.Context(), runID); err != nil {
		switch {
		case errors.Is(err, ErrNotAvailable):
			apierr.MapError(w, "[agent-manager] run stop", apierr.Unavailable("agent-manager is not available"))
		case strings.Contains(err.Error(), "status 404"):
			apierr.MapError(w, "[agent-manager] run stop", apierr.NotFound("run not found"))
		case errors.Is(err, ErrRequestFailed):
			apierr.MapError(w, "[agent-manager] run stop", apierr.BadGateway("failed to stop run"))
		default:
			apierr.MapError(w, "[agent-manager] run stop", apierr.Internal("failed to stop run"))
		}
		return
	}

	if err := httputil.JSON(w, stopRunResponse{
		RunID:   runID,
		Stopped: true,
		Status:  "stop_requested",
	}); err != nil {
		apierr.MapError(w, "[agent-manager] run stop", apierr.Internal("failed to encode response"))
	}
}

func isActiveStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "starting", "running", "needs_review":
		return true
	default:
		return false
	}
}

func durationSeconds(startedAt, finishedAt string) float64 {
	started, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return 0
	}
	end := time.Now().UTC()
	if strings.TrimSpace(finishedAt) != "" {
		if parsedEnd, parseErr := time.Parse(time.RFC3339, finishedAt); parseErr == nil {
			end = parsedEnd
		}
	}
	if end.Before(started) {
		return 0
	}
	seconds := end.Sub(started).Seconds()
	if seconds <= 0 {
		return 0
	}
	return seconds
}
