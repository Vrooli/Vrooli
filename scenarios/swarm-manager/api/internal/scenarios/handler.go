// Package scenarios provides HTTP handlers for scenario catalog management.
//
// Scenarios are read from the Vrooli ecosystem's scenarios directory and their
// service.json files. This handler provides read and update access to the scenario
// catalog with optional filtering, search, and metadata management.
//
// Related PRD targets: OT-P0-005, OT-P0-006
package scenarios

import (
	"encoding/json"
	"io/fs"
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

// ServiceJSON represents the structure of a scenario's service.json file.
type ServiceJSON struct {
	Profile ServiceProfile `json:"profile"`
}

// ServiceProfile contains the profile section of service.json.
type ServiceProfile struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// Handler provides HTTP handlers for scenario operations.
type Handler struct {
	scenariosDir string
}

// NewHandler creates a new scenarios handler.
// If scenariosDir is empty, it defaults to the Vrooli scenarios directory.
func NewHandler(scenariosDir string) *Handler {
	if scenariosDir == "" {
		scenariosDir = "scenarios"
	}
	return &Handler{scenariosDir: scenariosDir}
}

// LoadAll exposes scenario listing for non-HTTP consumers (recommendations engine).
// This keeps data access centralized in the scenarios package.
func (h *Handler) LoadAll() ([]Scenario, error) {
	return h.loadAllScenarios()
}

// Load exposes scenario retrieval for non-HTTP consumers.
func (h *Handler) Load(name string) (Scenario, error) {
	return h.loadScenario(name)
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
	scenarios, err := h.loadAllScenarios()
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

	scenario, err := h.loadScenario(name)
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

	// Verify scenario exists
	scenarioPath := filepath.Join(h.scenariosDir, name)
	serviceJSONPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	if _, err := os.Stat(serviceJSONPath); os.IsNotExist(err) {
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
	metadata, err := h.loadMetadata(name)
	if err != nil && !os.IsNotExist(err) {
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
	if err := h.saveMetadata(name, metadata); err != nil {
		httputil.InternalError(w, "[scenarios] update", "failed to save metadata")
		return
	}

	// Return updated scenario
	scenario, err := h.loadScenario(name)
	if err != nil {
		httputil.InternalError(w, "[scenarios] update", "failed to load scenario")
		return
	}

	log.Printf("[scenarios] updated: %q (isGreenfield=%v, recommendationsEnabled=%v)", name, scenario.IsGreenfield, scenario.RecommendationsEnabled)
	resp := &apipb.ScenarioResponse{Scenario: scenarioToProto(scenario)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[scenarios] update", "failed to encode response")
	}
}

// loadAllScenarios reads all scenarios from the scenarios directory.
func (h *Handler) loadAllScenarios() ([]Scenario, error) {
	var scenarios []Scenario

	err := filepath.WalkDir(h.scenariosDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) && path == h.scenariosDir {
				return nil
			}
			return err
		}

		// Only look at .vrooli/service.json files in immediate subdirectories
		if d.IsDir() && path != h.scenariosDir {
			serviceJSONPath := filepath.Join(path, ".vrooli", "service.json")
			if _, err := os.Stat(serviceJSONPath); err == nil {
				scenario, err := h.loadScenarioFromPath(path)
				if err == nil {
					scenarios = append(scenarios, scenario)
				} else {
					log.Printf("[scenarios] load: skipping %q due to error: %v", path, err)
				}
			}
			return fs.SkipDir // Don't descend further
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if scenarios == nil {
		scenarios = []Scenario{}
	}
	return scenarios, nil
}

// loadScenario reads a single scenario by name.
func (h *Handler) loadScenario(name string) (Scenario, error) {
	scenarioPath := filepath.Join(h.scenariosDir, name)
	return h.loadScenarioFromPath(scenarioPath)
}

// metadataPath returns the path to the metadata file for a scenario.
func (h *Handler) metadataPath(name string) string {
	return filepath.Join(h.scenariosDir, name, ".vrooli", "metadata.json")
}

// loadMetadata reads the metadata for a scenario.
// [REQ:REQ-P0-007] Load editable metadata from .vrooli/metadata.json
func (h *Handler) loadMetadata(name string) (ScenarioMetadata, error) {
	metaPath := h.metadataPath(name)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if no metadata file exists
			return ScenarioMetadata{
				IsGreenfield:           false,
				RecommendationsEnabled: true, // Enabled by default
			}, nil
		}
		return ScenarioMetadata{}, err
	}

	var metadata ScenarioMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return ScenarioMetadata{}, err
	}
	return metadata, nil
}

// saveMetadata writes the metadata for a scenario.
// [REQ:REQ-P0-007] Persist editable metadata to .vrooli/metadata.json
func (h *Handler) saveMetadata(name string, metadata ScenarioMetadata) error {
	metaPath := h.metadataPath(name)

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

// loadScenarioFromPath reads a scenario from its directory.
// [REQ:REQ-P0-007] Includes metadata for greenfield and recommendations settings
func (h *Handler) loadScenarioFromPath(scenarioPath string) (Scenario, error) {
	name := filepath.Base(scenarioPath)
	serviceJSONPath := filepath.Join(scenarioPath, ".vrooli", "service.json")

	data, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		return Scenario{}, err
	}

	var serviceJSON ServiceJSON
	if err := json.Unmarshal(data, &serviceJSON); err != nil {
		return Scenario{}, err
	}

	// Determine display name
	displayName := serviceJSON.Profile.Name
	if displayName == "" {
		displayName = name
	}

	// Determine default greenfield status (check for PRD.md)
	defaultGreenfield := true
	prdPath := filepath.Join(scenarioPath, "PRD.md")
	if _, err := os.Stat(prdPath); err == nil {
		defaultGreenfield = false
	}

	// Determine status (check for running processes - simplified for now)
	status := StatusUnknown

	// Read priority from lighthouse.json if available
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

	tags := serviceJSON.Profile.Tags
	if tags == nil {
		tags = []string{}
	}

	// Load metadata for editable fields
	// [REQ:REQ-P0-007] Apply metadata overrides for greenfield and recommendations
	metadata, _ := h.loadMetadata(name)

	// Use metadata values; metadata file stores explicit user choices
	// If metadata file doesn't exist, loadMetadata returns defaults
	isGreenfield := metadata.IsGreenfield
	// If no metadata file exists and PRD.md doesn't exist, it's greenfield
	metaPath := h.metadataPath(name)
	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		isGreenfield = defaultGreenfield
	}

	return Scenario{
		Name:                   name,
		DisplayName:            displayName,
		Description:            serviceJSON.Profile.Description,
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
//   - archive: If true, archives the scenario to the ideas backlog instead of permanent deletion
//
// The archive option creates an idea entry from the scenario's metadata, preserving
// important information for potential future revival. This provides a safety net
// for accidental deletions while keeping the scenarios directory clean.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Verify scenario exists
	scenarioPath := filepath.Join(h.scenariosDir, name)
	serviceJSONPath := filepath.Join(scenarioPath, ".vrooli", "service.json")
	if _, err := os.Stat(serviceJSONPath); os.IsNotExist(err) {
		httputil.NotFound(w, "", "scenario not found")
		return
	}

	// Parse archive option from query parameter
	archive := r.URL.Query().Get("archive") == "true"

	// Load scenario data before deletion (for archive or logging)
	scenario, err := h.loadScenario(name)
	if err != nil {
		httputil.InternalError(w, "[scenarios] delete", "failed to load scenario data")
		return
	}

	// If archiving, create an idea entry first
	if archive {
		if err := h.archiveToIdeas(scenario); err != nil {
			httputil.InternalError(w, "[scenarios] delete", "failed to archive scenario")
			return
		}
		log.Printf("[scenarios] archived: %q to ideas backlog", name)
	}

	// Delete the scenario directory
	if err := os.RemoveAll(scenarioPath); err != nil {
		httputil.InternalError(w, "[scenarios] delete", "failed to delete scenario directory")
		return
	}

	log.Printf("[scenarios] deleted: %q (archived=%v)", name, archive)

	message := "Scenario permanently deleted"
	if archive {
		message = "Scenario archived to ideas backlog and deleted"
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

// archiveToIdeas creates an idea entry from a scenario's metadata.
// [REQ:REQ-P0-008] Archive functionality for scenario preservation
func (h *Handler) archiveToIdeas(scenario Scenario) error {
	// Create idea directory structure
	ideaName := scenario.Name + "-archived"
	ideaDir := filepath.Join(h.scenariosDir, "..", "swarm-manager", "ideas", ideaName)

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
