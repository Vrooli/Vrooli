package scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

var (
	errProtectedScenarioDelete = errors.New("cannot delete swarm-manager scenario")
	errArchiveTargetExists     = errors.New("archive target already exists")
)

// SpecSyncArchiveContext mirrors the execution package's ArchiveContext.
type SpecSyncArchiveContext struct {
	ScenarioName   string
	ScenarioPath   string
	PresetOrCustom string
	PreservePaths  []string
	PreservePreset string
}

// SpecSyncArchiveRecord is the result of queueing a spec-sync-archive.
type SpecSyncArchiveRecord struct {
	ExecutionID string
	Status      string
}

// ExecutionQueuer queues spec-sync-archive executions.
type ExecutionQueuer interface {
	QueueSpecSyncArchive(ctx context.Context, ac SpecSyncArchiveContext) (SpecSyncArchiveRecord, error)
}

func normalizePreserveFilesRequest(req *apipb.PreserveFilesRequest) {
	if req == nil {
		return
	}
	if req.Preset != nil {
		normalized := strings.ToLower(strings.TrimSpace(*req.Preset))
		if normalized == "" {
			req.Preset = nil
		} else {
			req.Preset = &normalized
		}
	}
	if len(req.Paths) > 0 {
		trimmed := make([]string, 0, len(req.Paths))
		for _, path := range req.Paths {
			candidate := strings.TrimSpace(path)
			if candidate != "" {
				trimmed = append(trimmed, candidate)
			}
		}
		req.Paths = trimmed
	}
}

// Delete removes a scenario from the catalog with safeguards.
// [REQ:REQ-P0-008] DELETE /api/v1/scenarios/{name} endpoint with archive option
//
// Query parameters:
//   - archive: If true, archives the scenario to the backlog (idea kind) instead of permanent deletion
//
// Request body (optional, JSON):
//
//	{
//	  "preserveFiles": {
//	    "paths": ["PRD.md", "docs/**"],  // Explicit paths/globs to preserve
//	    "preset": "documentation"         // Or use a preset: documentation, requirements, planning, all-planning
//	  }
//	}
//
// The archive option creates a backlog idea entry from the scenario's metadata, preserving
// important information for potential future revival. This provides a safety net
// for accidental deletions while keeping the scenarios directory clean.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	source, scenarioPath, ok := h.resolveDeletableScenario(w, r, name)
	if !ok {
		return
	}

	// Parse archive option from query parameter
	archive := r.URL.Query().Get("archive") == "true"

	// Parse optional request body for preserve_files
	preserveFiles, ok := parseDeletePreserveFiles(w, r)
	if !ok {
		return
	}

	// Load scenario data before deletion (for archive or logging)
	scenario, err := h.loadScenarioFromSource(source)
	if err != nil {
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to load scenario data"))
		return
	}

	var backlogIdeaName string
	var preservedFiles []string
	var archivedIdeaPath string

	// If archiving, create a backlog idea entry first
	if archive {
		ideaName, ideaPath, preserved, err := h.archiveToBacklogIdea(scenario, scenarioPath, preserveFiles)
		if err != nil {
			if errors.Is(err, errArchiveTargetExists) {
				apierr.MapError(w, "[scenarios] delete", apierr.Conflict("%s", err.Error()))
				return
			}
			apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to archive scenario"))
			return
		}
		backlogIdeaName = ideaName
		archivedIdeaPath = ideaPath
		preservedFiles = preserved
		slog.Info("archived scenario to backlog", "scenario", name, "idea", ideaName, "preserved_files", len(preserved))
	}

	// Delete the scenario directory
	if err := os.RemoveAll(scenarioPath); err != nil {
		if archivedIdeaPath != "" {
			if rollbackErr := os.RemoveAll(archivedIdeaPath); rollbackErr != nil {
				slog.Error("archive rollback failed", "scenario", name, "path", archivedIdeaPath, "error", rollbackErr)
				apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to delete scenario directory; archive rollback failed"))
				return
			}
			slog.Warn("rolled back archive due to deletion failure", "scenario", name)
		}
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to delete scenario directory"))
		return
	}

	slog.Info("scenario deleted", "scenario", name, "archived", archive)
	h.invalidateCatalog()

	response := &apipb.DeleteScenarioResponse{
		Name:           name,
		Archived:       archive,
		Message:        deleteResponseMessage(archive, preservedFiles),
		PreservedFiles: preservedFiles,
	}
	if backlogIdeaName != "" {
		response.BacklogIdeaName = &backlogIdeaName
	}
	if err := httputil.ProtoJSON(w, response); err != nil {
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to encode response"))
	}
}

// resolveDeletableScenario locates the named scenario and validates it may be
// deleted (exists on disk, has a path, is not the protected swarm-manager).
// On any failure it writes the appropriate error response and returns ok=false.
func (h *Handler) resolveDeletableScenario(w http.ResponseWriter, r *http.Request, name string) (source ScenarioSource, scenarioPath string, ok bool) {
	trimmedName := strings.TrimSpace(name)

	source, found, err := h.findScenarioSource(r.Context(), name)
	if err != nil {
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to load scenarios from CLI"))
		return source, "", false
	}
	if !found {
		apierr.MapError(w, "", apierr.NotFound("scenario not found"))
		return source, "", false
	}
	scenarioPath = strings.TrimSpace(source.Path)
	if scenarioPath == "" {
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("scenario path missing from CLI output"))
		return source, "", false
	}
	if strings.EqualFold(trimmedName, "swarm-manager") {
		apierr.MapError(w, "[scenarios] delete", apierr.BadRequest("%s", errProtectedScenarioDelete.Error()))
		return source, "", false
	}
	if _, err := os.Stat(scenarioPath); err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "", apierr.NotFound("scenario not found"))
			return source, "", false
		}
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to access scenario directory"))
		return source, "", false
	}
	return source, scenarioPath, true
}

// parseDeletePreserveFiles decodes and validates the optional request body.
// A nil body is valid (returns nil, true). On a malformed/invalid body it
// writes the error response and returns ok=false.
func parseDeletePreserveFiles(w http.ResponseWriter, r *http.Request) (preserveFiles *apipb.PreserveFilesRequest, ok bool) {
	if r.Body == nil || r.ContentLength <= 0 {
		return nil, true
	}
	var req apipb.DeleteScenarioRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		apierr.MapError(w, "[scenarios] delete", apierr.BadRequest("invalid request body"))
		return nil, false
	}
	if req.PreserveFiles != nil {
		normalizePreserveFilesRequest(req.PreserveFiles)
	}
	if !httputil.ValidateProtoRequest(w, "[scenarios] delete", "invalid request body", &req) {
		return nil, false
	}
	return req.PreserveFiles, true
}

// deleteResponseMessage builds the human-readable result message.
func deleteResponseMessage(archive bool, preservedFiles []string) string {
	if !archive {
		return "Scenario permanently deleted"
	}
	if len(preservedFiles) > 0 {
		return fmt.Sprintf("Scenario archived to backlog (idea) with %d preserved files and deleted", len(preservedFiles))
	}
	return "Scenario archived to backlog (idea) and deleted"
}

// SpecSyncArchive triggers a spec-sync agent, then archives on completion.
// POST /api/v1/scenarios/{name}/spec-sync-archive
func (h *Handler) SpecSyncArchive(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	trimmedName := strings.TrimSpace(name)

	if h.executionQueuer == nil {
		apierr.MapError(w, "[scenarios] spec-sync-archive", apierr.Unavailable("spec-sync-archive is not available"))
		return
	}

	source, found, err := h.findScenarioSource(r.Context(), name)
	if err != nil {
		apierr.MapError(w, "[scenarios] spec-sync-archive", apierr.Internal("failed to load scenarios from CLI"))
		return
	}
	if !found {
		apierr.MapError(w, "", apierr.NotFound("scenario not found"))
		return
	}
	scenarioPath := strings.TrimSpace(source.Path)
	if scenarioPath == "" {
		apierr.MapError(w, "[scenarios] spec-sync-archive", apierr.Internal("scenario path missing from CLI output"))
		return
	}
	if strings.EqualFold(trimmedName, "swarm-manager") {
		apierr.MapError(w, "[scenarios] spec-sync-archive", apierr.BadRequest("%s", errProtectedScenarioDelete.Error()))
		return
	}

	// Parse optional request body for preserve_files
	var preserveFiles *apipb.PreserveFilesRequest
	if r.Body != nil && r.ContentLength > 0 {
		var req apipb.SpecSyncArchiveRequest
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			apierr.MapError(w, "[scenarios] spec-sync-archive", apierr.BadRequest("invalid request body"))
			return
		}
		if req.PreserveFiles != nil {
			normalizePreserveFilesRequest(req.PreserveFiles)
		}
		preserveFiles = req.PreserveFiles
	}

	// Build archive context
	ac := SpecSyncArchiveContext{
		ScenarioName:   trimmedName,
		ScenarioPath:   scenarioPath,
		PresetOrCustom: preservePresetOrCustom(preserveFiles),
	}
	if preserveFiles != nil {
		ac.PreservePaths = preserveFiles.Paths
		if preserveFiles.Preset != nil {
			ac.PreservePreset = *preserveFiles.Preset
		}
	}

	record, err := h.executionQueuer.QueueSpecSyncArchive(r.Context(), ac)
	if err != nil {
		if strings.Contains(err.Error(), "not available") {
			apierr.MapError(w, "[scenarios] spec-sync-archive", apierr.Unavailable("agent-manager is not available"))
			return
		}
		apierr.MapError(w, "[scenarios] spec-sync-archive", apierr.Internal("failed to queue spec-sync-archive: %s", httputil.TruncateErrorMessage(err, 240)))
		return
	}

	slog.Info("spec-sync-archive queued", "scenario", name, "execution_id", record.ExecutionID)
	resp := &apipb.SpecSyncArchiveResponse{
		ExecutionId: record.ExecutionID,
		Status:      record.Status,
		Message:     "Spec sync started. Poll execution status for progress.",
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusAccepted, resp); err != nil {
		apierr.MapError(w, "[scenarios] spec-sync-archive", apierr.Internal("failed to encode response"))
	}
}

// archiveToBacklogIdea creates a backlog idea entry from a scenario's metadata.
// [REQ:REQ-P0-008] Archive functionality for scenario preservation
// Returns the idea name and list of preserved files.
func (h *Handler) archiveToBacklogIdea(scenario Scenario, scenarioPath string, preserveFiles *apipb.PreserveFilesRequest) (string, string, []string, error) {
	ideaRoot, err := h.deriveBacklogIdeasRoot(scenarioPath)
	if err != nil {
		return "", "", nil, err
	}

	// Stage archive content outside the target scenario directory first.
	ideaName := scenario.Name + "-archived"
	ideaDir := filepath.Join(ideaRoot, ideaName)
	if _, err := os.Stat(ideaDir); err == nil {
		return "", "", nil, fmt.Errorf("%w: %s", errArchiveTargetExists, ideaName)
	}
	stagingRoot := filepath.Join(filepath.Dir(strings.TrimSpace(scenarioPath)), ".swarm-manager-archive-staging")
	if err := os.MkdirAll(stagingRoot, 0o750); err != nil {
		return "", "", nil, err
	}
	stagingDir, err := os.MkdirTemp(stagingRoot, ideaName+"-")
	if err != nil {
		return "", "", nil, err
	}
	defer func() {
		if rmErr := os.RemoveAll(stagingDir); rmErr != nil && !os.IsNotExist(rmErr) {
			slog.Debug("scenarios: cleanup staging dir failed", "err", rmErr, "dir", stagingDir)
		}
	}()

	if err := os.MkdirAll(stagingDir, 0o750); err != nil {
		return "", "", nil, err
	}

	// Copy preserved files into archive/ subdirectory to separate scenario
	// artifacts from backlog-specific data (spec.json, clarify/, suggest/, enhance/).
	preservedFiles := []string{}
	if preserveFiles != nil {
		archiveSubdir := filepath.Join(stagingDir, "archive")
		preserved, err := copyPreservedFiles(scenarioPath, archiveSubdir, preserveFiles)
		if err != nil {
			slog.Warn("failed to copy some preserved files", "error", err)
			// Continue with what we have, don't fail the entire archive
		}
		preservedFiles = preserved
	}

	// Create spec.json with scenario metadata
	now := time.Now().UTC().Format(time.RFC3339)
	spec := map[string]interface{}{
		"name":                   ideaName,
		"title":                  "[Archived] " + scenario.DisplayName,
		"description":            scenario.Description,
		"status":                 "completed",
		"priority":               scenario.Priority,
		"tags":                   append(scenario.Tags, "archived", "from-scenario"),
		"created":                now,
		"updated":                now,
		"archived_at":            now,
		"sourceScenarioName":     scenario.Name,
		"sourceScenarioPath":     filepath.Clean(scenarioPath),
		"archivedBy":             archiveActor(),
		"archiveReason":          "scenario deleted with archive=true",
		"preservedFiles":         preservedFiles,
		"preservePresetOrCustom": preservePresetOrCustom(preserveFiles),
	}

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", "", nil, err
	}

	specPath := filepath.Join(stagingDir, "spec.json")
	if err := os.WriteFile(specPath, data, 0o600); err != nil {
		return "", "", nil, err
	}

	if err := os.MkdirAll(ideaRoot, 0o750); err != nil {
		return "", "", nil, err
	}
	if err := os.Rename(stagingDir, ideaDir); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", "", nil, fmt.Errorf("%w: %s", errArchiveTargetExists, ideaName)
		}
		return "", "", nil, err
	}

	return ideaName, ideaDir, preservedFiles, nil
}
