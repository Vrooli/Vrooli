// Package scenarios provides HTTP handlers for scenario catalog management.
//
// Scenarios are sourced from the Vrooli CLI (vrooli scenario list), then enriched
// with local metadata (priority, greenfield toggle, recommendations enablement).
// This handler provides read and update access to the scenario catalog with optional
// filtering, search, and metadata management.
//
// Related PRD targets: OT-P0-005, OT-P0-006
package scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

var errScenarioNameRequired = errors.New("scenario name is required")

// Scenario represents a deployed application in the Vrooli ecosystem.
// [REQ:REQ-P0-006] Scenario data structure for catalog listing
// [REQ:REQ-P0-007] Includes metadata for greenfield toggle and recommendations
type Scenario struct {
	Name                   string         `json:"name"`
	DisplayName            string         `json:"displayName"`
	Description            string         `json:"description"`
	Status                 ScenarioStatus `json:"status"`
	Priority               int            `json:"priority"`
	CompletenessScore      *int           `json:"completenessScore,omitempty"`
	IsGreenfield           bool           `json:"isGreenfield"`
	Tags                   []string       `json:"tags"`
	RecommendationsEnabled bool           `json:"recommendationsEnabled"`
}

// ScenarioMetadata stores editable scenario settings in a local JSON file.
// [REQ:REQ-P0-007] Persistent metadata for scenario management
type ScenarioMetadata struct {
	IsGreenfield           bool `json:"isGreenfield"`
	RecommendationsEnabled bool `json:"recommendationsEnabled"`
}

// Handler provides HTTP handlers for scenario operations.
type Handler struct {
	scenariosDir string
	source       Source
	lifecycle    Lifecycle
	completeness CompletenessSource
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

// LoadAll exposes scenario listing for non-HTTP consumers (recommendations engine).
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
		Name:                   s.Name,
		DisplayName:            s.DisplayName,
		Description:            s.Description,
		Status:                 string(s.Status),
		Priority:               int32(s.Priority),
		CompletenessScore:      completeness,
		IsGreenfield:           s.IsGreenfield,
		Tags:                   s.Tags,
		RecommendationsEnabled: s.RecommendationsEnabled,
	}
}

// RegisterRoutes registers the scenarios API routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/scenarios", h.List).Methods("GET")
	r.HandleFunc("/api/v1/scenarios/{name}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/scenarios/{name}", h.UpdateMetadata).Methods("PATCH")
	r.HandleFunc("/api/v1/scenarios/{name}", h.Delete).Methods("DELETE")
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
//   - recommendationsEnabled: Whether the recommendation engine can suggest improvements
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
	if req.RecommendationsEnabled != nil {
		metadata.RecommendationsEnabled = *req.RecommendationsEnabled
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

	log.Printf("[scenarios] updated: %q (isGreenfield=%v, recommendationsEnabled=%v)", name, scenario.IsGreenfield, scenario.RecommendationsEnabled)
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
				IsGreenfield:           false,
				RecommendationsEnabled: true, // Enabled by default
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
// [REQ:REQ-P0-007] Includes metadata for greenfield and recommendations settings
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
			IsGreenfield:           false,
			RecommendationsEnabled: true,
		}
		metaExists = false
	}

	// Use metadata values; metadata file stores explicit user choices
	isGreenfield := metadata.IsGreenfield
	if !metaExists {
		isGreenfield = defaultGreenfield
	}

	return Scenario{
		Name:                   name,
		DisplayName:            displayName,
		Description:            description,
		Status:                 status,
		Priority:               priority,
		IsGreenfield:           isGreenfield,
		RecommendationsEnabled: metadata.RecommendationsEnabled,
		Tags:                   tags,
	}, nil
}

// Delete removes a scenario from the catalog with safeguards.
// [REQ:REQ-P0-008] DELETE /api/v1/scenarios/{name} endpoint with archive option
//
// Query parameters:
//   - archive: If true, archives the scenario to the backlog (idea kind) instead of permanent deletion
//
// The archive option creates a backlog idea entry from the scenario's metadata, preserving
// important information for potential future revival. This provides a safety net
// for accidental deletions while keeping the scenarios directory clean.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

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

	// Load scenario data before deletion (for archive or logging)
	scenario, err := h.loadScenarioFromSource(source)
	if err != nil {
		httputil.InternalError(w, "[scenarios] delete", "failed to load scenario data")
		return
	}

	// If archiving, create a backlog idea entry first
	if archive {
		if err := h.archiveToBacklogIdea(scenario, scenarioPath); err != nil {
			httputil.InternalError(w, "[scenarios] delete", "failed to archive scenario")
			return
		}
		log.Printf("[scenarios] archived: %q to backlog (idea)", name)
	}

	// Delete the scenario directory
	if err := os.RemoveAll(scenarioPath); err != nil {
		httputil.InternalError(w, "[scenarios] delete", "failed to delete scenario directory")
		return
	}

	log.Printf("[scenarios] deleted: %q (archived=%v)", name, archive)

	message := "Scenario permanently deleted"
	if archive {
		message = "Scenario archived to backlog (idea) and deleted"
	}
	response := &apipb.DeleteScenarioResponse{
		Name:     name,
		Archived: archive,
		Message:  message,
	}
	if err := httputil.ProtoJSON(w, response); err != nil {
		httputil.InternalError(w, "[scenarios] delete", "failed to encode response")
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
}

// archiveToBacklogIdea creates a backlog idea entry from a scenario's metadata.
// [REQ:REQ-P0-008] Archive functionality for scenario preservation
func (h *Handler) archiveToBacklogIdea(scenario Scenario, scenarioPath string) error {
	ideaRoot, err := deriveBacklogIdeasRoot(scenarioPath)
	if err != nil {
		return err
	}

	// Create backlog idea directory structure
	ideaName := scenario.Name + "-archived"
	ideaDir := filepath.Join(ideaRoot, ideaName)

	// Ensure parent directory exists
	if err := os.MkdirAll(ideaDir, 0o755); err != nil {
		return err
	}

	// Create spec.json with scenario metadata
	spec := map[string]interface{}{
		"name":        ideaName,
		"title":       "[Archived] " + scenario.DisplayName,
		"description": scenario.Description,
		"status":      "archived",
		"priority":    scenario.Priority,
		"tags":        append(scenario.Tags, "archived", "from-scenario"),
		"created":     "auto-generated",
		"updated":     "auto-generated",
	}

	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}

	specPath := filepath.Join(ideaDir, "spec.json")
	return os.WriteFile(specPath, data, 0o644)
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

func deriveBacklogIdeasRoot(scenarioPath string) (string, error) {
	cleaned := filepath.Clean(scenarioPath)
	parts := strings.Split(filepath.ToSlash(cleaned), "/scenarios/")
	if len(parts) < 2 {
		return "", fmt.Errorf("unable to derive backlog (idea) root from scenario path %q", scenarioPath)
	}
	root := parts[0]
	if root == "" {
		root = string(filepath.Separator)
	}
	return filepath.Join(root, "scenarios", "swarm-manager", "ideas"), nil
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
