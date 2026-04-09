package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"scenario-to-desktop-api/shared/errors"
	httputil "scenario-to-desktop-api/shared/http"
)

type bundleCleanRequest struct {
	Framework    string `json:"framework,omitempty"`
	LocationMode string `json:"location_mode,omitempty"`
	PipelineID   string `json:"pipeline_id,omitempty"`
}

type bundleCleanResponse struct {
	ScenarioName string `json:"scenario_name"`
	Framework    string `json:"framework"`
	LocationMode string `json:"location_mode"`
	PipelineID   string `json:"pipeline_id,omitempty"`
	Path         string `json:"path"`
	Removed      bool   `json:"removed"`
}

func (h *Handler) handleBundleClean(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioName := vars["name"]
	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	var req bundleCleanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		httputil.WriteError(w, errors.ErrBadRequest("invalid request body").
			WithCause(err).
			WithRecovery(errors.RecoveryFixInput, "Ensure the request body is valid JSON"))
		return
	}

	framework := req.Framework
	if framework == "" {
		framework = FrameworkElectron
	}
	locationMode := req.LocationMode
	if locationMode == "" {
		locationMode = "proper"
	}
	if isStagingLocation(locationMode) && strings.TrimSpace(req.PipelineID) == "" {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline_id is required for staging/temp location_mode"))
		return
	}

	// Compute scenario path from conventional repo layout.
	home, _ := os.UserHomeDir()
	scenarioPath := filepath.Join(home, "Vrooli", "scenarios", scenarioName)

	cfg := &Config{
		ScenarioName:   scenarioName,
		LocationMode:   locationMode,
		Framework:      framework,
		Platforms:      nil,
		DeploymentMode: "",
	}
	_, desktopPath := resolvePipelineOutputPaths(cfg, scenarioPath, req.PipelineID, framework)
	if desktopPath == "" || !strings.Contains(desktopPath, filepath.Join("platforms", framework)) {
		httputil.WriteError(w, errors.ErrBadRequest("refusing to clean: computed output path is unsafe"))
		return
	}
	bundlePath := filepath.Join(desktopPath, "bundle")

	removed := false
	if _, statErr := os.Stat(bundlePath); statErr == nil {
		if err := removeAllRobust(bundlePath); err != nil {
			httputil.WriteError(w, errors.ErrBadRequest("failed to clean bundle directory").WithCause(err))
			return
		}
		removed = true
	} else if !os.IsNotExist(statErr) {
		httputil.WriteError(w, errors.ErrBadRequest("failed to stat bundle directory").WithCause(statErr))
		return
	}

	httputil.WriteJSON(w, http.StatusOK, bundleCleanResponse{
		ScenarioName: scenarioName,
		Framework:    framework,
		LocationMode: locationMode,
		PipelineID:   req.PipelineID,
		Path:         bundlePath,
		Removed:      removed,
	})
}

// handleGetActivePipeline handles GET /api/v1/scenarios/{name}/pipeline/active
// Returns the active pipeline for a scenario, optionally auto-creating one if none exists.
// Query params:
//   - auto_create: if "true", creates a new pipeline if none exists (default: true)
func (h *Handler) handleGetActivePipeline(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline manager not configured"))
		return
	}

	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	// Check auto_create query param (default: true)
	autoCreate := r.URL.Query().Get("auto_create") != "false"

	if !autoCreate {
		// Just return the current active pipeline, don't create
		status, ok := h.manager.GetActivePipelineStatus(scenarioName)
		if !ok {
			httputil.WriteJSONOK(w, ActivePipelineResponse{
				Pipeline: nil,
				Created:  false,
			})
			return
		}
		httputil.WriteJSONOK(w, ActivePipelineResponse{
			Pipeline: status,
			Created:  false,
		})
		return
	}

	// Get or create active pipeline
	status, created, err := h.manager.GetOrCreateActivePipeline(context.Background(), scenarioName, nil)
	if err != nil {
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to get or create active pipeline").
			InDomain("pipeline"))
		return
	}

	httputil.WriteJSONOK(w, ActivePipelineResponse{
		Pipeline: status,
		Created:  created,
	})
}

// handleCreatePipeline handles POST /api/v1/scenarios/{name}/pipeline
// Creates a new active pipeline, archiving the current one if it exists.
func (h *Handler) handleCreatePipeline(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline manager not configured"))
		return
	}

	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	// Parse optional request body for config
	var config *Config
	if r.ContentLength > 0 {
		var c Config
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			httputil.WriteError(w, errors.ErrBadRequest("invalid request body").
				WithCause(err).
				WithRecovery(errors.RecoveryFixInput, "Ensure the request body is valid JSON"))
			return
		}
		config = &c
	}

	status, archivedID, err := h.manager.CreateNewPipeline(context.Background(), scenarioName, config)
	if err != nil {
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to create pipeline").
			InDomain("pipeline"))
		return
	}

	httputil.WriteJSONAccepted(w, CreatePipelineResponse{
		Pipeline:   status,
		ArchivedID: archivedID,
	})
}

// handleResetPipeline handles POST /api/v1/scenarios/{name}/pipeline/reset
// Archives the current active pipeline and clears the active slot.
func (h *Handler) handleResetPipeline(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline manager not configured"))
		return
	}

	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	archivedID, err := h.manager.ResetActivePipeline(scenarioName)
	if err != nil {
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to reset pipeline").
			InDomain("pipeline"))
		return
	}

	httputil.WriteJSONOK(w, ResetPipelineResponse{
		ArchivedID: archivedID,
		Cleared:    true,
	})
}

// handleGetPipelineHistory handles GET /api/v1/scenarios/{name}/pipeline/history
// Returns the history of pipelines for a scenario.
// Query params:
//   - limit: maximum number of pipelines to return (default: 10)
func (h *Handler) handleGetPipelineHistory(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline manager not configured"))
		return
	}

	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	// Parse limit query param
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	pipelines, total, err := h.manager.GetPipelineHistory(scenarioName, limit)
	if err != nil {
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to get pipeline history").
			InDomain("pipeline"))
		return
	}

	httputil.WriteJSONOK(w, PipelineHistoryResponse{
		Pipelines: pipelines,
		Total:     total,
	})
}

// handleStartActivePipeline handles POST /api/v1/scenarios/{name}/pipeline/start
// Starts the active pipeline for a scenario with optional config overrides.
// This is the correct way to run stages - it uses the existing active pipeline
// rather than creating orphaned new ones.
// Query params:
//   - block: if "true", wait for pipeline completion before returning (default: false)
//   - timeout: max wait time in seconds when block=true (default: 600)
func (h *Handler) handleStartActivePipeline(w http.ResponseWriter, r *http.Request) {
	if h.manager == nil {
		httputil.WriteError(w, errors.ErrBadRequest("pipeline manager not configured"))
		return
	}

	vars := mux.Vars(r)
	scenarioName := vars["name"]

	if scenarioName == "" {
		httputil.WriteError(w, errors.ErrBadRequest("scenario name is required"))
		return
	}

	// Parse optional request body for config overrides
	config, ok := h.parseStartConfig(w, r, scenarioName)
	if !ok {
		return // error already written
	}

	// Check for blocking mode
	block, timeoutSecs := parseBlockingParams(r)

	if block {
		h.startActivePipelineBlocking(w, r, scenarioName, config, timeoutSecs)
		return
	}

	h.startActivePipelineAsync(w, scenarioName, config)
}

// parseStartConfig parses the optional request body for config overrides, including version updates.
// Returns (nil, true) when there is no body. Returns (nil, false) on error (error already written to w).
func (h *Handler) parseStartConfig(w http.ResponseWriter, r *http.Request, scenarioName string) (*Config, bool) {
	if r.ContentLength <= 0 {
		return nil, true
	}

	var req RunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, errors.ErrBadRequest("invalid request body").
			WithCause(err).
			WithRecovery(errors.RecoveryFixInput, "Ensure the request body is valid JSON"))
		return nil, false
	}

	c := req.Config
	c.ScenarioName = scenarioName

	if req.VersionUpdate != nil {
		if c.Version != "" {
			httputil.WriteError(w, errors.ErrBadRequest("version and version_update are mutually exclusive"))
			return nil, false
		}
		updater := newVersionUpdater("")
		version, rollback, derr := updater.Apply(c.ScenarioName, req.VersionUpdate)
		if derr != nil {
			httputil.WriteError(w, derr)
			return nil, false
		}
		if version != "" {
			c.Version = version
		}
		c.setVersionRollback(rollback)
	}

	return &c, true
}

// parseBlockingParams extracts block and timeout query parameters from the request.
func parseBlockingParams(r *http.Request) (block bool, timeoutSecs int) {
	block = r.URL.Query().Get("block") == "true"
	timeoutSecs = 600 // default 10 minutes
	if ts := r.URL.Query().Get("timeout"); ts != "" {
		if parsed, err := strconv.Atoi(ts); err == nil && parsed > 0 {
			timeoutSecs = parsed
		}
	}
	return
}

// startActivePipelineBlocking handles the blocking mode of starting an active pipeline.
func (h *Handler) startActivePipelineBlocking(w http.ResponseWriter, r *http.Request, scenarioName string, config *Config, timeoutSecs int) {
	extendWriteDeadline(w, time.Duration(timeoutSecs)*time.Second)

	status, err := h.manager.StartActivePipelineBlocking(r.Context(), scenarioName, config, timeoutSecs)
	if err != nil {
		if status == nil || (status.IsComplete() && status.Status != StatusCompleted) {
			if config != nil {
				if rollback := config.takeVersionRollback(); rollback != nil {
					_ = rollback.Restore()
				}
			}
		}
		if status != nil {
			httputil.WriteError(w, errors.Wrap(errors.CodePipelineTimeout, err, "pipeline timed out").
				InDomain("pipeline").
				WithDetails(map[string]interface{}{
					"pipeline_id": status.PipelineID,
					"status":      status.Status,
				}).
				WithRecovery(errors.RecoveryRetry, "Increase the timeout or check pipeline progress"))
			return
		}
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to run pipeline").
			InDomain("pipeline"))
		return
	}

	if status.Status == StatusFailed {
		httputil.WriteJSON(w, http.StatusInternalServerError, status)
	} else {
		httputil.WriteJSONOK(w, status)
	}
}

// startActivePipelineAsync handles the async mode of starting an active pipeline.
func (h *Handler) startActivePipelineAsync(w http.ResponseWriter, scenarioName string, config *Config) {
	status, err := h.manager.StartActivePipeline(context.Background(), scenarioName, config)
	if err != nil {
		if config != nil {
			if rollback := config.takeVersionRollback(); rollback != nil {
				_ = rollback.Restore()
			}
		}
		httputil.WriteError(w, errors.Wrap(errors.CodePipelineFailed, err, "failed to start active pipeline").
			InDomain("pipeline"))
		return
	}

	statusURL := fmt.Sprintf("%s/%s", h.basePath, status.PipelineID)

	httputil.WriteJSONAccepted(w, StartActivePipelineResponse{
		Pipeline:  status,
		StatusURL: statusURL,
		Message:   "Pipeline started successfully",
	})
}
