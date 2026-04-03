package scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

var (
	errProtectedScenarioDelete = errors.New("cannot delete swarm-manager scenario")
	errArchiveTargetExists     = errors.New("archive target already exists")
)

// archivePresets defines named file patterns for archive preservation.
var archivePresets = map[string][]string{
	"documentation": {"PRD.md", "README.md", "docs/**", "*.md"},
	"requirements":  {"PRD.md", "requirements/**", "specs/**", "REQUIREMENTS.md"},
	"planning":      {"PRD.md", ".vrooli/**", "planning/**", "design/**"},
	"all-planning":  {"PRD.md", "README.md", "docs/**", "requirements/**", "specs/**", "planning/**", "design/**", ".vrooli/**", "*.md"},
}

var archiveIgnoredDirs = map[string]struct{}{
	"node_modules": {},
	".git":         {},
	"dist":         {},
	"build":        {},
	"coverage":     {},
	".next":        {},
	".turbo":       {},
	"target":       {},
	"vendor":       {},
}

func isIgnoredArchivePath(path string) bool {
	parts := strings.Split(path, string(filepath.Separator))
	for _, part := range parts {
		if _, ignored := archiveIgnoredDirs[part]; ignored {
			return true
		}
	}
	return false
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

// EventDispatcher emits graph change events for real-time WebSocket updates.
type EventDispatcher interface {
	DispatchNodeUpdate(nodeType, nodeID string, data any)
	DispatchInvalidate(lenses ...string)
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
	trimmedName := strings.TrimSpace(name)

	source, found, err := h.findScenarioSource(r.Context(), name)
	if err != nil {
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to load scenarios from CLI"))
		return
	}
	if !found {
		apierr.MapError(w, "", apierr.NotFound("scenario not found"))
		return
	}
	scenarioPath := strings.TrimSpace(source.Path)
	if scenarioPath == "" {
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("scenario path missing from CLI output"))
		return
	}
	if strings.EqualFold(trimmedName, "swarm-manager") {
		apierr.MapError(w, "[scenarios] delete", apierr.BadRequest("%s", errProtectedScenarioDelete.Error()))
		return
	}
	if _, err := os.Stat(scenarioPath); err != nil {
		if os.IsNotExist(err) {
			apierr.MapError(w, "", apierr.NotFound("scenario not found"))
			return
		}
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to access scenario directory"))
		return
	}

	// Parse archive option from query parameter
	archive := r.URL.Query().Get("archive") == "true"

	// Parse optional request body for preserve_files
	var preserveFiles *apipb.PreserveFilesRequest
	if r.Body != nil && r.ContentLength > 0 {
		var req apipb.DeleteScenarioRequest
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			apierr.MapError(w, "[scenarios] delete", apierr.BadRequest("invalid request body"))
			return
		}
		if req.PreserveFiles != nil {
			normalizePreserveFilesRequest(req.PreserveFiles)
		}
		if !httputil.ValidateProtoRequest(w, "[scenarios] delete", "invalid request body", &req) {
			return
		}
		preserveFiles = req.PreserveFiles
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
		log.Printf("[scenarios] archived: %q to backlog (idea=%s, preserved=%d files)", name, ideaName, len(preserved))
	}

	// Delete the scenario directory
	if err := os.RemoveAll(scenarioPath); err != nil {
		if archivedIdeaPath != "" {
			if rollbackErr := os.RemoveAll(archivedIdeaPath); rollbackErr != nil {
				log.Printf("[scenarios] delete: archive rollback failed for %q at %q: %v", name, archivedIdeaPath, rollbackErr)
				apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to delete scenario directory; archive rollback failed"))
				return
			}
			log.Printf("[scenarios] delete: rolled back archive for %q due to deletion failure", name)
		}
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to delete scenario directory"))
		return
	}

	log.Printf("[scenarios] deleted: %q (archived=%v)", name, archive)

	message := "Scenario permanently deleted"
	if archive {
		message = "Scenario archived to backlog (idea) and deleted"
		if len(preservedFiles) > 0 {
			message = fmt.Sprintf("Scenario archived to backlog (idea) with %d preserved files and deleted", len(preservedFiles))
		}
	}
	response := &apipb.DeleteScenarioResponse{
		Name:           name,
		Archived:       archive,
		Message:        message,
		PreservedFiles: preservedFiles,
	}
	if backlogIdeaName != "" {
		response.BacklogIdeaName = &backlogIdeaName
	}
	if err := httputil.ProtoJSON(w, response); err != nil {
		apierr.MapError(w, "[scenarios] delete", apierr.Internal("failed to encode response"))
	}
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

	log.Printf("[scenarios] spec-sync-archive queued for %q: execution_id=%s", name, record.ExecutionID)
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
	if err := os.MkdirAll(stagingRoot, 0o755); err != nil {
		return "", "", nil, err
	}
	stagingDir, err := os.MkdirTemp(stagingRoot, ideaName+"-")
	if err != nil {
		return "", "", nil, err
	}
	defer func() {
		_ = os.RemoveAll(stagingDir)
	}()

	if err := os.MkdirAll(stagingDir, 0o755); err != nil {
		return "", "", nil, err
	}

	// Copy preserved files into archive/ subdirectory to separate scenario
	// artifacts from backlog-specific data (spec.json, clarify/, suggest/, enhance/).
	preservedFiles := []string{}
	if preserveFiles != nil {
		archiveSubdir := filepath.Join(stagingDir, "archive")
		preserved, err := copyPreservedFiles(scenarioPath, archiveSubdir, preserveFiles)
		if err != nil {
			log.Printf("[scenarios] archive: warning: failed to copy some preserved files: %v", err)
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
		"status":                 "archived",
		"priority":               scenario.Priority,
		"tags":                   append(scenario.Tags, "archived", "from-scenario"),
		"created":                now,
		"updated":                now,
		"sourceScenarioName":     scenario.Name,
		"sourceScenarioPath":     filepath.Clean(scenarioPath),
		"archivedAt":             now,
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
	if err := os.WriteFile(specPath, data, 0o644); err != nil {
		return "", "", nil, err
	}

	if err := os.MkdirAll(ideaRoot, 0o755); err != nil {
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

// copyPreservedFiles copies files matching the specified patterns from scenario to idea directory.
func copyPreservedFiles(scenarioPath, ideaDir string, preserveFiles *apipb.PreserveFilesRequest) ([]string, error) {
	explicitPatterns := append([]string{}, preserveFiles.Paths...)
	presetPatterns := []string{}
	if preserveFiles.Preset != nil && *preserveFiles.Preset != "" {
		presetMatches, ok := archivePresets[*preserveFiles.Preset]
		if ok {
			presetPatterns = append(presetPatterns, presetMatches...)
		}
	}

	patterns := append([]string{}, explicitPatterns...)
	patterns = append(patterns, presetPatterns...)
	if len(patterns) == 0 {
		return nil, nil
	}

	// Deduplicate patterns
	seen := make(map[string]bool)
	uniquePatterns := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		normalized, err := normalizeArchiveRelativePath(pattern)
		if err != nil {
			log.Printf("[scenarios] archive: warning: skipping invalid preserve path %q: %v", pattern, err)
			continue
		}
		if !seen[normalized] {
			seen[normalized] = true
			uniquePatterns = append(uniquePatterns, normalized)
		}
	}

	// Collect files matching patterns. Preset matches exclude generated/vendor dirs.
	matchedFiles := make(map[string]bool)
	for _, pattern := range uniquePatterns {
		matches, err := resolveGlobPattern(scenarioPath, pattern)
		if err != nil {
			log.Printf("[scenarios] archive: warning: failed to resolve pattern %q: %v", pattern, err)
			continue
		}
		isPresetPattern := false
		for _, presetPattern := range presetPatterns {
			if presetPattern == pattern {
				isPresetPattern = true
				break
			}
		}
		for _, match := range matches {
			if isPresetPattern && isIgnoredArchivePath(match) {
				continue
			}
			matchedFiles[match] = true
		}
	}

	// Copy matched files
	var preserved []string
	for relPath := range matchedFiles {
		srcPath := filepath.Join(scenarioPath, relPath)
		dstPath := filepath.Join(ideaDir, relPath)

		if err := copyFile(srcPath, dstPath); err != nil {
			log.Printf("[scenarios] archive: warning: failed to copy %q: %v", relPath, err)
			continue
		}
		preserved = append(preserved, relPath)
	}

	sort.Strings(preserved)
	return preserved, nil
}

// resolveGlobPattern expands a glob pattern relative to a base directory.
func resolveGlobPattern(baseDir, pattern string) ([]string, error) {
	normalizedPattern, err := normalizeArchiveRelativePath(pattern)
	if err != nil {
		return nil, err
	}

	// Handle exact file matches first
	exactPath := filepath.Join(baseDir, normalizedPattern)
	if info, err := os.Stat(exactPath); err == nil && !info.IsDir() {
		return []string{normalizedPattern}, nil
	}

	// Use doublestar for ** glob support
	fullPattern := filepath.Join(baseDir, normalizedPattern)
	matches, err := doublestar.FilepathGlob(fullPattern)
	if err != nil {
		return nil, err
	}

	// Convert to relative paths and filter directories
	var result []string
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		relPath, err := filepath.Rel(baseDir, match)
		if err != nil {
			continue
		}
		normalizedRelPath, err := normalizeArchiveRelativePath(relPath)
		if err != nil {
			continue
		}
		result = append(result, normalizedRelPath)
	}

	return result, nil
}

func normalizeArchiveRelativePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path is required")
	}
	normalized := filepath.Clean(filepath.FromSlash(trimmed))
	if normalized == "." {
		return "", errors.New("path must reference a file")
	}
	if filepath.IsAbs(normalized) {
		return "", errors.New("path must be relative")
	}
	if normalized == ".." || strings.HasPrefix(normalized, ".."+string(filepath.Separator)) {
		return "", errors.New("path traversal is not allowed")
	}
	return normalized, nil
}

// copyFile copies a file from src to dst, creating parent directories as needed.
func copyFile(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if srcInfo.IsDir() {
		return fmt.Errorf("cannot copy directory: %s", src)
	}

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Chmod(srcInfo.Mode())
}

func (h *Handler) deriveBacklogIdeasRoot(scenarioPath string) (string, error) {
	trimmedScenarioPath := strings.TrimSpace(scenarioPath)
	if trimmedScenarioPath != "" {
		cleanScenarioPath := filepath.Clean(trimmedScenarioPath)
		if strings.EqualFold(filepath.Base(cleanScenarioPath), "swarm-manager") {
			return "", errProtectedScenarioDelete
		}
	}

	baseDir := strings.TrimSpace(h.scenariosDir)
	if baseDir == "" {
		baseDir = "scenarios"
	}
	if !filepath.IsAbs(baseDir) {
		if absBaseDir, err := filepath.Abs(baseDir); err == nil {
			baseDir = absBaseDir
		}
	}
	return filepath.Join(baseDir, "swarm-manager", "ideas"), nil
}

func preservePresetOrCustom(preserveFiles *apipb.PreserveFilesRequest) string {
	if preserveFiles == nil {
		return "none"
	}
	if len(preserveFiles.Paths) > 0 {
		return "custom"
	}
	if preserveFiles.Preset != nil && strings.TrimSpace(*preserveFiles.Preset) != "" {
		return "preset:" + strings.ToLower(strings.TrimSpace(*preserveFiles.Preset))
	}
	return "none"
}

func archiveActor() string {
	actor := strings.TrimSpace(os.Getenv("USER"))
	if actor == "" {
		return "swarm-manager-api"
	}
	return actor
}
