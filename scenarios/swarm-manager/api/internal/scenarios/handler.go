// Package scenarios provides HTTP handlers for scenario catalog management.
//
// Scenarios are sourced from the Vrooli CLI (vrooli scenario list), then enriched
// with local metadata (priority, greenfield toggle).
// This handler provides read and update access to the scenario catalog with optional
// filtering, search, and metadata management.
//
// DOC: docs/concepts/ARCHITECTURE.md#key-flows
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/SEAMS.md
//
// Related PRD targets: OT-P0-005, OT-P0-006
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
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/httputil"
)

// ScenarioStatus represents the runtime state of a scenario.
type ScenarioStatus string

const (
	StatusRunning ScenarioStatus = "running"
	StatusStopped ScenarioStatus = "stopped"
	StatusError   ScenarioStatus = "error"
	StatusUnknown ScenarioStatus = "unknown"
)

var (
	errScenarioNameRequired    = errors.New("scenario name is required")
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

// Scenario represents a deployed application in the Vrooli ecosystem.
// [REQ:REQ-P0-006] Scenario data structure for catalog listing
// [REQ:REQ-P0-007] Includes metadata for greenfield toggle
type Scenario struct {
	Name              string         `json:"name"`
	DisplayName       string         `json:"displayName"`
	Description       string         `json:"description"`
	Status            ScenarioStatus `json:"status"`
	Priority          int            `json:"priority"`
	CompletenessScore *int           `json:"completenessScore,omitempty"`
	IsGreenfield      bool           `json:"isGreenfield"`
	Tags              []string       `json:"tags"`
}

// ScenarioMetadata stores editable scenario settings in a local JSON file.
// [REQ:REQ-P0-007] Persistent metadata for scenario management
type ScenarioMetadata struct {
	IsGreenfield bool `json:"isGreenfield"`
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
}

// Handler provides HTTP handlers for scenario operations.
type Handler struct {
	scenariosDir    string
	source          Source
	lifecycle       Lifecycle
	completeness    CompletenessSource
	executionQueuer ExecutionQueuer
	eventDispatcher EventDispatcher
}

// NewHandler creates a new scenarios handler.
// If scenariosDir is empty, it defaults to the Vrooli scenarios directory.
func NewHandler(scenariosDir string) *Handler {
	if scenariosDir == "" {
		scenariosDir = "scenarios"
	}
	return NewHandlerWithDeps(
		scenariosDir,
		NewCLIProvider(defaultCLITimeout),
		NewCLILifecycle(),
		NewCLICompletenessSource(defaultCompletenessTimeout),
	)
}

// NewHandlerWithSource creates a scenarios handler with a custom source.
func NewHandlerWithSource(scenariosDir string, source Source) *Handler {
	return NewHandlerWithDeps(
		scenariosDir,
		source,
		NewCLILifecycle(),
		NewCLICompletenessSource(defaultCompletenessTimeout),
	)
}

// NewHandlerWithDeps creates a scenarios handler with injected dependencies.
func NewHandlerWithDeps(scenariosDir string, source Source, lifecycle Lifecycle, completeness CompletenessSource) *Handler {
	if scenariosDir == "" {
		scenariosDir = "scenarios"
	}
	if source == nil {
		source = NewCLIProvider(defaultCLITimeout)
	}
	if lifecycle == nil {
		lifecycle = NewCLILifecycle()
	}
	if completeness == nil {
		completeness = NewCLICompletenessSource(defaultCompletenessTimeout)
	}
	return &Handler{
		scenariosDir: scenariosDir,
		source:       source,
		lifecycle:    lifecycle,
		completeness: completeness,
	}
}

// SetExecutionQueuer sets the execution queuer for spec-sync-archive support.
func (h *Handler) SetExecutionQueuer(eq ExecutionQueuer) {
	h.executionQueuer = eq
}

// SetEventDispatcher sets an optional event dispatcher for real-time graph updates.
func (h *Handler) SetEventDispatcher(d EventDispatcher) {
	h.eventDispatcher = d
}

// LoadAll exposes scenario listing for non-HTTP consumers.
// This keeps data access centralized in the scenarios package.
func (h *Handler) LoadAll() ([]Scenario, error) {
	return h.loadAllScenarios(context.Background())
}

// Load exposes scenario retrieval for non-HTTP consumers.
func (h *Handler) Load(name string) (Scenario, error) {
	return h.loadScenario(context.Background(), name)
}

func scenarioToProto(s Scenario) *domainpb.Scenario {
	var completeness *int32
	if s.CompletenessScore != nil {
		value := int32(*s.CompletenessScore)
		completeness = &value
	}
	return &domainpb.Scenario{
		Name:              s.Name,
		DisplayName:       s.DisplayName,
		Description:       s.Description,
		Status:            string(s.Status),
		Priority:          int32(s.Priority),
		CompletenessScore: completeness,
		IsGreenfield:      s.IsGreenfield,
		Tags:              s.Tags,
	}
}

// RegisterRoutes registers the scenarios API routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/scenarios", h.List).Methods("GET")
	r.HandleFunc("/api/v1/scenarios/{name}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/scenarios/{name}", h.UpdateMetadata).Methods("PATCH")
	r.HandleFunc("/api/v1/scenarios/{name}", h.Delete).Methods("DELETE")
	r.HandleFunc("/api/v1/scenarios/{name}/files", h.ListFiles).Methods("GET")
	r.HandleFunc("/api/v1/scenarios/{name}/spec-sync-archive", h.SpecSyncArchive).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{name}/start", h.Start).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{name}/stop", h.Stop).Methods("POST")
	r.HandleFunc("/api/v1/scenarios/{name}/restart", h.Restart).Methods("POST")
}

// List returns all scenarios with optional search and filter parameters.
// [REQ:REQ-P0-006] GET /api/v1/scenarios endpoint
//
// Query parameters:
//   - search: Filter by name or description (case-insensitive)
//   - status: Filter by status (running, stopped, error, unknown)
//   - tags: Filter by tags (comma-separated)
//   - sort: Sort field (priority, name, displayName) - default: priority
//   - order: Sort order (asc, desc) - default: asc for priority, asc for name
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	scenarios, err := h.loadAllScenarios(r.Context())
	if err != nil {
		httputil.InternalError(w, "[scenarios] list", "failed to load scenarios")
		return
	}

	// Extract query params
	query := r.URL.Query()
	search := strings.ToLower(query.Get("search"))
	status := query.Get("status")
	tagsParam := query.Get("tags")
	sortField := query.Get("sort")
	sortOrder := query.Get("order")

	// Apply filters
	scenarios = h.filterScenarios(scenarios, search, status, tagsParam)

	// Sort scenarios
	h.sortScenarios(scenarios, sortField, sortOrder)

	log.Printf("[scenarios] list: returning %d scenarios (search=%q, status=%q, tags=%q)", len(scenarios), search, status, tagsParam)
	protoScenarios := make([]*domainpb.Scenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		protoScenarios = append(protoScenarios, scenarioToProto(scenario))
	}
	resp := &apipb.ListScenariosResponse{Scenarios: protoScenarios}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[scenarios] list", "failed to encode response")
	}
}

// filterScenarios applies search, status, and tag filters to scenarios.
func (h *Handler) filterScenarios(scenarios []Scenario, search, status, tagsParam string) []Scenario {
	// Apply search filter
	if search != "" {
		var filtered []Scenario
		for _, s := range scenarios {
			if matchesSearch(s, search) {
				filtered = append(filtered, s)
			}
		}
		scenarios = filtered
	}

	// Apply status filter
	if status != "" {
		var filtered []Scenario
		for _, s := range scenarios {
			if string(s.Status) == status {
				filtered = append(filtered, s)
			}
		}
		scenarios = filtered
	}

	// Apply tags filter
	if tagsParam != "" {
		filterTags := strings.Split(tagsParam, ",")
		var filtered []Scenario
		for _, s := range scenarios {
			if hasAnyTag(s.Tags, filterTags) {
				filtered = append(filtered, s)
			}
		}
		scenarios = filtered
	}

	return scenarios
}

// matchesSearch checks if a scenario matches a search term.
func matchesSearch(s Scenario, search string) bool {
	return strings.Contains(strings.ToLower(s.Name), search) ||
		strings.Contains(strings.ToLower(s.DisplayName), search) ||
		strings.Contains(strings.ToLower(s.Description), search)
}

// sortScenarios sorts scenarios by the specified field and order.
func (h *Handler) sortScenarios(scenarios []Scenario, sortField, sortOrder string) {
	if sortField == "" {
		sortField = "priority"
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}

	sort.Slice(scenarios, func(i, j int) bool {
		var less bool
		switch sortField {
		case "name":
			less = scenarios[i].Name < scenarios[j].Name
		case "displayName":
			less = scenarios[i].DisplayName < scenarios[j].DisplayName
		default: // priority
			if scenarios[i].Priority != scenarios[j].Priority {
				less = scenarios[i].Priority < scenarios[j].Priority
			} else {
				less = scenarios[i].Name < scenarios[j].Name
			}
		}
		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

// ListFiles returns the file tree for a scenario.
// GET /api/v1/scenarios/{name}/files
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	source, found, err := h.findScenarioSource(r.Context(), name)
	if err != nil {
		httputil.InternalError(w, "[scenarios] list files", "failed to load scenarios from CLI")
		return
	}
	if !found {
		httputil.NotFound(w, "", "scenario not found")
		return
	}

	scenarioPath := strings.TrimSpace(source.Path)
	if scenarioPath == "" {
		httputil.InternalError(w, "[scenarios] list files", "scenario path missing")
		return
	}

	files, err := h.buildScenarioFileTree(scenarioPath, "")
	if err != nil {
		log.Printf("[scenarios] list files: failed to build file tree for %q: %v", name, err)
		httputil.InternalError(w, "[scenarios] list files", "failed to read file tree")
		return
	}

	resp := &apipb.ScenarioFilesResponse{Files: scenarioFilesToProto(files)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[scenarios] list files", "failed to encode response")
	}
}

// ScenarioFile represents a file or directory within a scenario folder.
type ScenarioFile struct {
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	Type     string         `json:"type"` // "file" or "directory"
	Size     int64          `json:"size,omitempty"`
	Children []ScenarioFile `json:"children,omitempty"`
}

func scenarioFilesToProto(files []ScenarioFile) []*apipb.ScenarioFile {
	if len(files) == 0 {
		return nil
	}
	result := make([]*apipb.ScenarioFile, 0, len(files))
	for _, file := range files {
		result = append(result, scenarioFileToProto(file))
	}
	return result
}

func scenarioFileToProto(file ScenarioFile) *apipb.ScenarioFile {
	children := scenarioFilesToProto(file.Children)
	var size *int64
	if file.Type == "file" {
		size = &file.Size
	}
	return &apipb.ScenarioFile{
		Name:     file.Name,
		Path:     file.Path,
		Type:     file.Type,
		Size:     size,
		Children: children,
	}
}

func (h *Handler) buildScenarioFileTree(baseDir, relativePath string) ([]ScenarioFile, error) {
	dirPath := filepath.Join(baseDir, relativePath)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	files := make([]ScenarioFile, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(relativePath, name)
		if relativePath == "" {
			path = name
		}
		file := ScenarioFile{
			Name: name,
			Path: path,
		}

		if entry.IsDir() {
			file.Type = "directory"
			children, err := h.buildScenarioFileTree(baseDir, path)
			if err == nil {
				file.Children = children
			}
		} else {
			file.Type = "file"
			if info, err := entry.Info(); err == nil {
				file.Size = info.Size()
			}
		}

		files = append(files, file)
	}

	// Sort: directories first, then alphabetically
	sort.Slice(files, func(i, j int) bool {
		if files[i].Type != files[j].Type {
			return files[i].Type == "directory"
		}
		return files[i].Name < files[j].Name
	})

	if files == nil {
		files = []ScenarioFile{}
	}
	return files, nil
}

// Get returns a single scenario by name.
// [REQ:REQ-P0-006] GET /api/v1/scenarios/{name} endpoint
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	scenario, err := h.loadScenario(r.Context(), name)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "", "scenario not found")
			return
		}
		httputil.InternalError(w, "[scenarios] get", "failed to load scenario")
		return
	}

	resp := &apipb.ScenarioResponse{Scenario: scenarioToProto(scenario)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[scenarios] get", "failed to encode response")
	}
}

// UpdateMetadata updates editable metadata for a scenario.
// [REQ:REQ-P0-007] PATCH /api/v1/scenarios/{name} endpoint for metadata management
//
// This endpoint allows toggling:
//   - isGreenfield: Whether the scenario is treated as a new project
//
// Metadata is stored in .vrooli/metadata.json within the scenario directory.
func (h *Handler) UpdateMetadata(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	source, found, err := h.findScenarioSource(r.Context(), name)
	if err != nil {
		httputil.InternalError(w, "[scenarios] update", "failed to load scenarios from CLI")
		return
	}
	if !found {
		httputil.NotFound(w, "", "scenario not found")
		return
	}

	// Parse request body
	var req apipb.UpdateScenarioMetadataRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.BadRequest(w, "[scenarios] update", "invalid request body")
		return
	}
	if !httputil.ValidateProtoRequest(w, "[scenarios] update", "invalid request body", &req) {
		return
	}

	// Load existing metadata
	metadata, _, err := h.loadMetadata(source.Path)
	if err != nil {
		httputil.InternalError(w, "[scenarios] update", "failed to load metadata")
		return
	}

	// Apply updates (partial update pattern)
	if req.IsGreenfield != nil {
		metadata.IsGreenfield = *req.IsGreenfield
	}

	// Save updated metadata
	if err := h.saveMetadata(source.Path, metadata); err != nil {
		httputil.InternalError(w, "[scenarios] update", "failed to save metadata")
		return
	}

	// Return updated scenario
	scenario, err := h.loadScenarioFromSource(source)
	if err != nil {
		httputil.InternalError(w, "[scenarios] update", "failed to load scenario")
		return
	}
	applyCompletenessScore(&scenario, h.getCompletenessScores(r.Context()))

	log.Printf("[scenarios] updated: %q (isGreenfield=%v)", name, scenario.IsGreenfield)
	resp := &apipb.ScenarioResponse{Scenario: scenarioToProto(scenario)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[scenarios] update", "failed to encode response")
	}
}

// loadAllScenarios reads all scenarios from the CLI source.
func (h *Handler) loadAllScenarios(ctx context.Context) ([]Scenario, error) {
	sources, err := h.source.List(ctx)
	if err != nil {
		return nil, err
	}

	var scores map[string]int
	if len(sources) > 0 {
		scores = h.getCompletenessScores(ctx)
	}
	scenarios := make([]Scenario, 0, len(sources))
	for _, source := range sources {
		scenario, err := h.loadScenarioFromSource(source)
		if err != nil {
			log.Printf("[scenarios] load: skipping %q due to error: %v", source.Name, err)
			continue
		}
		applyCompletenessScore(&scenario, scores)
		scenarios = append(scenarios, scenario)
	}
	return scenarios, nil
}

// loadScenario reads a single scenario by name.
func (h *Handler) loadScenario(ctx context.Context, name string) (Scenario, error) {
	source, found, err := h.findScenarioSource(ctx, name)
	if err != nil {
		return Scenario{}, err
	}
	if !found {
		return Scenario{}, os.ErrNotExist
	}
	scenario, err := h.loadScenarioFromSource(source)
	if err != nil {
		return Scenario{}, err
	}
	applyCompletenessScore(&scenario, h.getCompletenessScores(ctx))
	return scenario, nil
}

func (h *Handler) findScenarioSource(ctx context.Context, name string) (ScenarioSource, bool, error) {
	sources, err := h.source.List(ctx)
	if err != nil {
		return ScenarioSource{}, false, err
	}
	trimmed := strings.TrimSpace(name)
	for _, source := range sources {
		if source.Name == trimmed {
			return source, true, nil
		}
	}
	return ScenarioSource{}, false, nil
}

func (h *Handler) getCompletenessScores(ctx context.Context) map[string]int {
	if h.completeness == nil {
		return nil
	}
	scores, err := h.completeness.Scores(ctx)
	if err != nil {
		log.Printf("[scenarios] completeness: failed to load scores: %v", err)
		return nil
	}
	return scores
}

// metadataPath returns the path to the metadata file for a scenario.
func metadataPath(scenarioPath string) string {
	return filepath.Join(scenarioPath, ".vrooli", "metadata.json")
}

// loadMetadata reads the metadata for a scenario.
// [REQ:REQ-P0-007] Load editable metadata from .vrooli/metadata.json
func (h *Handler) loadMetadata(scenarioPath string) (ScenarioMetadata, bool, error) {
	metaPath := metadataPath(scenarioPath)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if no metadata file exists
			return ScenarioMetadata{
				IsGreenfield: false,
			}, false, nil
		}
		return ScenarioMetadata{}, false, err
	}

	var metadata ScenarioMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return ScenarioMetadata{}, true, err
	}
	return metadata, true, nil
}

// saveMetadata writes the metadata for a scenario.
// [REQ:REQ-P0-007] Persist editable metadata to .vrooli/metadata.json
func (h *Handler) saveMetadata(scenarioPath string, metadata ScenarioMetadata) error {
	metaPath := metadataPath(scenarioPath)

	// Ensure .vrooli directory exists
	dir := filepath.Dir(metaPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0o644)
}

// loadScenarioFromSource maps CLI metadata into a Scenario enriched with local data.
// [REQ:REQ-P0-007] Includes metadata for greenfield settings
func (h *Handler) loadScenarioFromSource(source ScenarioSource) (Scenario, error) {
	name := strings.TrimSpace(source.Name)
	if name == "" {
		return Scenario{}, errors.New("scenario name missing")
	}
	scenarioPath := strings.TrimSpace(source.Path)
	if scenarioPath == "" {
		return Scenario{}, errors.New("scenario path missing")
	}

	if _, err := os.Stat(scenarioPath); err != nil {
		return Scenario{}, err
	}

	displayName := name
	description := strings.TrimSpace(source.Description)

	status := normalizeScenarioStatus(source.Status)

	// Read priority from lighthouse.json if available
	priority := loadPriorityFromLighthouse(scenarioPath)

	tags := source.Tags
	if tags == nil {
		tags = []string{}
	}

	// Determine default greenfield status (check for PRD.md)
	defaultGreenfield := true
	prdPath := filepath.Join(scenarioPath, "PRD.md")
	if _, err := os.Stat(prdPath); err == nil {
		defaultGreenfield = false
	}

	// Load metadata for editable fields
	metadata, metaExists, err := h.loadMetadata(scenarioPath)
	if err != nil {
		log.Printf("[scenarios] metadata: failed to load for %q: %v", name, err)
		metadata = ScenarioMetadata{
			IsGreenfield: false,
		}
		metaExists = false
	}

	// Use metadata values; metadata file stores explicit user choices
	isGreenfield := metadata.IsGreenfield
	if !metaExists {
		isGreenfield = defaultGreenfield
	}

	return Scenario{
		Name:         name,
		DisplayName:  displayName,
		Description:  description,
		Status:       status,
		Priority:     priority,
		IsGreenfield: isGreenfield,
		Tags:         tags,
	}, nil
}

// loadScenarioByPath builds a Scenario from a name and filesystem path.
// Used by the Archiver when the scenario is already located.
func (h *Handler) loadScenarioByPath(name, scenarioPath string) (Scenario, error) {
	return h.loadScenarioFromSource(ScenarioSource{
		Name: name,
		Path: scenarioPath,
	})
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
		httputil.InternalError(w, "[scenarios] delete", "failed to load scenarios from CLI")
		return
	}
	if !found {
		httputil.NotFound(w, "", "scenario not found")
		return
	}
	scenarioPath := strings.TrimSpace(source.Path)
	if scenarioPath == "" {
		httputil.InternalError(w, "[scenarios] delete", "scenario path missing from CLI output")
		return
	}
	if strings.EqualFold(trimmedName, "swarm-manager") {
		httputil.BadRequest(w, "[scenarios] delete", errProtectedScenarioDelete.Error())
		return
	}
	if _, err := os.Stat(scenarioPath); err != nil {
		if os.IsNotExist(err) {
			httputil.NotFound(w, "", "scenario not found")
			return
		}
		httputil.InternalError(w, "[scenarios] delete", "failed to access scenario directory")
		return
	}

	// Parse archive option from query parameter
	archive := r.URL.Query().Get("archive") == "true"

	// Parse optional request body for preserve_files
	var preserveFiles *apipb.PreserveFilesRequest
	if r.Body != nil && r.ContentLength > 0 {
		var req apipb.DeleteScenarioRequest
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			httputil.BadRequest(w, "[scenarios] delete", "invalid request body")
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
		httputil.InternalError(w, "[scenarios] delete", "failed to load scenario data")
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
				httputil.Conflict(w, "[scenarios] delete", err.Error())
				return
			}
			httputil.InternalError(w, "[scenarios] delete", "failed to archive scenario")
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
				httputil.InternalError(w, "[scenarios] delete", "failed to delete scenario directory; archive rollback failed")
				return
			}
			log.Printf("[scenarios] delete: rolled back archive for %q due to deletion failure", name)
		}
		httputil.InternalError(w, "[scenarios] delete", "failed to delete scenario directory")
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
		httputil.InternalError(w, "[scenarios] delete", "failed to encode response")
	}
}

// SpecSyncArchive triggers a spec-sync agent, then archives on completion.
// POST /api/v1/scenarios/{name}/spec-sync-archive
func (h *Handler) SpecSyncArchive(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	trimmedName := strings.TrimSpace(name)

	if h.executionQueuer == nil {
		httputil.ServiceUnavailable(w, "[scenarios] spec-sync-archive", "spec-sync-archive is not available")
		return
	}

	source, found, err := h.findScenarioSource(r.Context(), name)
	if err != nil {
		httputil.InternalError(w, "[scenarios] spec-sync-archive", "failed to load scenarios from CLI")
		return
	}
	if !found {
		httputil.NotFound(w, "", "scenario not found")
		return
	}
	scenarioPath := strings.TrimSpace(source.Path)
	if scenarioPath == "" {
		httputil.InternalError(w, "[scenarios] spec-sync-archive", "scenario path missing from CLI output")
		return
	}
	if strings.EqualFold(trimmedName, "swarm-manager") {
		httputil.BadRequest(w, "[scenarios] spec-sync-archive", errProtectedScenarioDelete.Error())
		return
	}

	// Parse optional request body for preserve_files
	var preserveFiles *apipb.PreserveFilesRequest
	if r.Body != nil && r.ContentLength > 0 {
		var req apipb.SpecSyncArchiveRequest
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			httputil.BadRequest(w, "[scenarios] spec-sync-archive", "invalid request body")
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
			httputil.ServiceUnavailable(w, "[scenarios] spec-sync-archive", "agent-manager is not available")
			return
		}
		httputil.InternalError(w, "[scenarios] spec-sync-archive", "failed to queue spec-sync-archive: "+httputil.TruncateErrorMessage(err, 240))
		return
	}

	log.Printf("[scenarios] spec-sync-archive queued for %q: execution_id=%s", name, record.ExecutionID)
	resp := &apipb.SpecSyncArchiveResponse{
		ExecutionId: record.ExecutionID,
		Status:      record.Status,
		Message:     "Spec sync started. Poll execution status for progress.",
	}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusAccepted, resp); err != nil {
		httputil.InternalError(w, "[scenarios] spec-sync-archive", "failed to encode response")
	}
}

// Start starts a scenario via the Vrooli CLI.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	h.handleLifecycleAction(w, r, "start")
}

// Stop stops a scenario via the Vrooli CLI.
func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	h.handleLifecycleAction(w, r, "stop")
}

// Restart restarts a scenario via the Vrooli CLI.
func (h *Handler) Restart(w http.ResponseWriter, r *http.Request) {
	h.handleLifecycleAction(w, r, "restart")
}

func (h *Handler) handleLifecycleAction(w http.ResponseWriter, r *http.Request, action string) {
	vars := mux.Vars(r)
	name := strings.TrimSpace(vars["name"])
	if name == "" {
		httputil.BadRequest(w, "[scenarios] "+action, "name is required")
		return
	}
	if h.lifecycle == nil {
		httputil.InternalError(w, "[scenarios] "+action, "scenario lifecycle is unavailable")
		return
	}

	_, found, err := h.findScenarioSource(r.Context(), name)
	if err != nil {
		httputil.InternalError(w, "[scenarios] "+action, "failed to load scenarios from CLI")
		return
	}
	if !found {
		httputil.NotFound(w, "", "scenario not found")
		return
	}

	var actionErr error
	switch action {
	case "start":
		actionErr = h.lifecycle.Start(r.Context(), name)
	case "stop":
		actionErr = h.lifecycle.Stop(r.Context(), name)
	case "restart":
		actionErr = h.lifecycle.Restart(r.Context(), name)
	default:
		httputil.BadRequest(w, "[scenarios] "+action, "unsupported action")
		return
	}
	if actionErr != nil {
		if errors.Is(actionErr, errScenarioNameRequired) {
			httputil.BadRequest(w, "[scenarios] "+action, "name is required")
			return
		}
		httputil.InternalError(w, "[scenarios] "+action, "failed to "+action+" scenario")
		return
	}

	scenario, err := h.loadScenario(r.Context(), name)
	if err != nil {
		httputil.InternalError(w, "[scenarios] "+action, "failed to load scenario")
		return
	}

	resp := &apipb.ScenarioResponse{Scenario: scenarioToProto(scenario)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[scenarios] "+action, "failed to encode response")
	}

	// Dispatch graph event for scenario status change.
	if h.eventDispatcher != nil {
		h.eventDispatcher.DispatchNodeUpdate("Scenario", "scenario/"+name, map[string]any{
			"name":   scenario.Name,
			"status": string(scenario.Status),
		})
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

func normalizeScenarioStatus(status string) ScenarioStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return StatusRunning
	case "stopped":
		return StatusStopped
	case "error":
		return StatusError
	default:
		return StatusUnknown
	}
}

func loadPriorityFromLighthouse(scenarioPath string) int {
	priority := 5 // Default priority
	lighthousePath := filepath.Join(scenarioPath, ".vrooli", "lighthouse.json")
	if lighthouseData, err := os.ReadFile(lighthousePath); err == nil {
		var lighthouse struct {
			Priority int `json:"priority"`
		}
		if err := json.Unmarshal(lighthouseData, &lighthouse); err == nil && lighthouse.Priority > 0 {
			priority = lighthouse.Priority
		}
	}
	return priority
}

func applyCompletenessScore(scenario *Scenario, scores map[string]int) {
	if scenario == nil || scores == nil {
		return
	}
	score, ok := scores[scenario.Name]
	if !ok {
		return
	}
	scenario.CompletenessScore = &score
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

// hasAnyTag checks if the scenario has any of the filter tags.
func hasAnyTag(scenarioTags, filterTags []string) bool {
	for _, ft := range filterTags {
		ft = strings.TrimSpace(strings.ToLower(ft))
		for _, st := range scenarioTags {
			if strings.ToLower(st) == ft {
				return true
			}
		}
	}
	return false
}
