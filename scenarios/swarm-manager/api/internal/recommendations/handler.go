// Package recommendations provides filesystem-backed recommendations and a basic
// suggestion engine for Swarm Manager.
package recommendations

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/domain"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/storage"
)

// RecommendationStatus represents the current state of a recommendation.
type RecommendationStatus string

const (
	StatusPending  RecommendationStatus = "pending"
	StatusApproved RecommendationStatus = "approved"
	StatusRejected RecommendationStatus = "rejected"
)

// RecommendationType represents the type of recommendation.
type RecommendationType string

const (
	TypeTest     RecommendationType = "test"
	TypeFeature  RecommendationType = "feature"
	TypeRefactor RecommendationType = "refactor"
	TypeDocs     RecommendationType = "docs"
)

// Recommendation represents a suggestion for improving a scenario.
type Recommendation struct {
	ID           string               `json:"id"`
	Scenario     string               `json:"scenarioName"`
	Type         RecommendationType   `json:"type"`
	Description  string               `json:"description"`
	Status       RecommendationStatus `json:"status"`
	Priority     int                  `json:"priority"`
	Created      string               `json:"created"`
	Source       string               `json:"source,omitempty"` // generated|manual
	TaskID       string               `json:"taskId,omitempty"`
	RunID        string               `json:"runId,omitempty"`
	StartedAt    string               `json:"startedAt,omitempty"`
	StartedBy    string               `json:"startedBy,omitempty"`
	AutoApproved bool                 `json:"autoApproved,omitempty"`
}

// Store persists recommendations in JSON.
type Store struct {
	path string
}

// NewStore creates a recommendations store. If path is empty, uses scenario default.
func NewStore(path string) *Store {
	if strings.TrimSpace(path) == "" {
		path = filepath.Join("scenarios", "swarm-manager", ".vrooli", "recommendations.json")
	}
	return &Store{path: path}
}

// Load returns all recommendations.
func (s *Store) Load() ([]Recommendation, error) {
	var items []Recommendation
	exists, err := storage.ReadJSON(s.path, &items)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []Recommendation{}, nil
	}
	return normalizeList(items), nil
}

// Save persists recommendations.
func (s *Store) Save(items []Recommendation) error {
	return storage.WriteJSONAtomic(s.path, items)
}

// Engine generates recommendations based on filesystem data.
type Engine struct {
	scenariosDir string
	source       scenarios.Source
	completeness scenarios.CompletenessSource
}

// NewEngine creates a recommendation engine.
func NewEngine(scenariosDir string) *Engine {
	if strings.TrimSpace(scenariosDir) == "" {
		scenariosDir = "scenarios"
	}
	return &Engine{
		scenariosDir: scenariosDir,
		source:       scenarios.NewCLIProvider(20 * time.Second),
		completeness: scenarios.NewCLICompletenessSource(30 * time.Second),
	}
}

// NewEngineWithSource creates a recommendation engine with a custom scenario source.
func NewEngineWithSource(scenariosDir string, source scenarios.Source) *Engine {
	if strings.TrimSpace(scenariosDir) == "" {
		scenariosDir = "scenarios"
	}
	if source == nil {
		source = scenarios.NewCLIProvider(20 * time.Second)
	}
	return &Engine{
		scenariosDir: scenariosDir,
		source:       source,
		completeness: scenarios.NewCLICompletenessSource(30 * time.Second),
	}
}

// NewEngineWithDeps creates a recommendation engine with custom dependencies.
func NewEngineWithDeps(scenariosDir string, source scenarios.Source, completeness scenarios.CompletenessSource) *Engine {
	if strings.TrimSpace(scenariosDir) == "" {
		scenariosDir = "scenarios"
	}
	if source == nil {
		source = scenarios.NewCLIProvider(20 * time.Second)
	}
	if completeness == nil {
		completeness = scenarios.NewCLICompletenessSource(30 * time.Second)
	}
	return &Engine{
		scenariosDir: scenariosDir,
		source:       source,
		completeness: completeness,
	}
}

// Generate produces recommendations based on settings and filesystem signals.
func (e *Engine) Generate(cfg settings.Settings) ([]Recommendation, error) {
	handler := scenarios.NewHandlerWithDeps(e.scenariosDir, e.source, nil, e.completeness)
	catalog, err := handler.LoadAll()
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	recs := []Recommendation{}

	for _, scenario := range catalog {
		if !scenario.RecommendationsEnabled {
			continue
		}

		scenarioPath := filepath.Join(e.scenariosDir, scenario.Name)
		basePriority := normalizeScenarioPriority(scenario.Priority)

		if cfg.RecommendationSources.Problems {
			if count := countProblems(filepath.Join(scenarioPath, "docs", "PROBLEMS.md")); count > 0 {
				priority := clampPriority(basePriority - 1)
				recs = append(recs, Recommendation{
					ID:          stableID(scenario.Name, TypeRefactor, fmt.Sprintf("problems:%d", count)),
					Scenario:    scenario.Name,
					Type:        TypeRefactor,
					Description: fmt.Sprintf("Resolve %d open items in docs/PROBLEMS.md", count),
					Status:      StatusPending,
					Priority:    priority,
					Created:     now,
					Source:      "generated",
				})
			}
		}

		if cfg.RecommendationSources.Completeness || cfg.RecommendationSources.Coverage {
			completion, passRate := loadRequirementsSummary(filepath.Join(scenarioPath, "coverage", "requirements-sync", "latest.json"))
			if cfg.RecommendationSources.Completeness && completion > 0 && completion < 70 {
				recs = append(recs, Recommendation{
					ID:          stableID(scenario.Name, TypeFeature, fmt.Sprintf("completeness:%d", completion)),
					Scenario:    scenario.Name,
					Type:        TypeFeature,
					Description: fmt.Sprintf("Increase requirements completion (currently %d%%)", completion),
					Status:      StatusPending,
					Priority:    clampPriority(basePriority - 1),
					Created:     now,
					Source:      "generated",
				})
			}
			if cfg.RecommendationSources.Coverage && passRate > 0 && passRate < 100 {
				recs = append(recs, Recommendation{
					ID:          stableID(scenario.Name, TypeTest, fmt.Sprintf("coverage:%d", passRate)),
					Scenario:    scenario.Name,
					Type:        TypeTest,
					Description: fmt.Sprintf("Improve validation pass rate (currently %d%%)", passRate),
					Status:      StatusPending,
					Priority:    clampPriority(basePriority),
					Created:     now,
					Source:      "generated",
				})
			}
		}

		if cfg.RecommendationSources.Tests {
			testingPath := filepath.Join(scenarioPath, ".vrooli", "testing.json")
			if _, err := os.Stat(testingPath); err != nil {
				recs = append(recs, Recommendation{
					ID:          stableID(scenario.Name, TypeTest, "testing-config"),
					Scenario:    scenario.Name,
					Type:        TypeTest,
					Description: "Add or update .vrooli/testing.json to document test coverage",
					Status:      StatusPending,
					Priority:    clampPriority(basePriority + 1),
					Created:     now,
					Source:      "generated",
				})
			}
		}

		if cfg.RecommendationSources.ScenarioNotes {
			notesPath := filepath.Join(scenarioPath, "docs", "PROGRESS.md")
			if _, err := os.Stat(notesPath); err != nil {
				recs = append(recs, Recommendation{
					ID:          stableID(scenario.Name, TypeDocs, "progress-log"),
					Scenario:    scenario.Name,
					Type:        TypeDocs,
					Description: "Create docs/PROGRESS.md to track scenario evolution",
					Status:      StatusPending,
					Priority:    clampPriority(basePriority + 1),
					Created:     now,
					Source:      "generated",
				})
			}
		}

		if cfg.RecommendationSources.CustomFocus {
			focus := strings.TrimSpace(cfg.CustomFocus)
			if focus != "" {
				recs = append(recs, Recommendation{
					ID:          stableID(scenario.Name, TypeFeature, "focus:"+focus),
					Scenario:    scenario.Name,
					Type:        TypeFeature,
					Description: fmt.Sprintf("Focus on: %s", focus),
					Status:      StatusPending,
					Priority:    clampPriority(basePriority + 1),
					Created:     now,
					Source:      "generated",
				})
			}
		}
	}

	return recs, nil
}

// Handler exposes HTTP endpoints for recommendations.
// [REQ:REQ-P1-002-API] Recommendation management API
// [REQ:REQ-P1-001-ENGINE] Recommendation engine core
// [REQ:REQ-P1-001-CONFIG] Recommendation data source configuration
type Handler struct {
	store         *Store
	engine        *Engine
	settingsStore *settings.Store
	agentService  agentmanager.Service
}

// NewHandler creates a new recommendations handler.
func NewHandler(path string) *Handler {
	return &Handler{
		store:         NewStore(path),
		engine:        NewEngine(""),
		settingsStore: settings.NewStore(""),
		agentService:  nil,
	}
}

// NewHandlerWithServices creates a new recommendations handler with injected dependencies.
func NewHandlerWithServices(store *Store, engine *Engine, settingsStore *settings.Store, agentService agentmanager.Service) *Handler {
	return &Handler{
		store:         store,
		engine:        engine,
		settingsStore: settingsStore,
		agentService:  agentService,
	}
}

// RegisterRoutes registers recommendation endpoints.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/recommendations", h.List).Methods("GET")
	r.HandleFunc("/api/v1/recommendations", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/recommendations/refresh", h.Refresh).Methods("POST")
	r.HandleFunc("/api/v1/recommendations/{id}", h.Update).Methods("PATCH")
	r.HandleFunc("/api/v1/recommendations/{id}/start", h.Start).Methods("POST")
}

// List returns recommendations, generating them if needed.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.settingsStore.Load()
	if err != nil {
		httputil.InternalError(w, "[recommendations] list", "failed to load settings")
		return
	}

	if cfg.RecommendationMode == "off" {
		resp := &apipb.ListRecommendationsResponse{Recommendations: nil}
		if err := httputil.ProtoJSON(w, resp); err != nil {
			httputil.InternalError(w, "[recommendations] list", "failed to encode response")
		}
		return
	}

	items, err := h.store.Load()
	if err != nil {
		httputil.InternalError(w, "[recommendations] list", "failed to load recommendations")
		return
	}

	refresh := r.URL.Query().Get("refresh") == "true"
	if refresh || len(items) == 0 {
		generated, err := h.engine.Generate(cfg)
		if err != nil {
			httputil.InternalError(w, "[recommendations] list", "failed to generate recommendations")
			return
		}
		items = mergeRecommendations(items, generated)
		items = applyYoloMode(items, cfg)
		if err := h.store.Save(items); err != nil {
			httputil.InternalError(w, "[recommendations] list", "failed to persist recommendations")
			return
		}
	}

	items = filterRecommendations(items, r)
	resp := &apipb.ListRecommendationsResponse{Recommendations: recommendationsToProto(items)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[recommendations] list", "failed to encode response")
	}
}

// Refresh forces a regeneration of recommendations.
func (h *Handler) Refresh(w http.ResponseWriter, _ *http.Request) {
	cfg, err := h.settingsStore.Load()
	if err != nil {
		httputil.InternalError(w, "[recommendations] refresh", "failed to load settings")
		return
	}

	if cfg.RecommendationMode == "off" {
		resp := &apipb.ListRecommendationsResponse{Recommendations: nil}
		if err := httputil.ProtoJSON(w, resp); err != nil {
			httputil.InternalError(w, "[recommendations] refresh", "failed to encode response")
		}
		return
	}

	existing, err := h.store.Load()
	if err != nil {
		httputil.InternalError(w, "[recommendations] refresh", "failed to load recommendations")
		return
	}

	generated, err := h.engine.Generate(cfg)
	if err != nil {
		httputil.InternalError(w, "[recommendations] refresh", "failed to generate recommendations")
		return
	}

	items := mergeRecommendations(existing, generated)
	items = applyYoloMode(items, cfg)
	if err := h.store.Save(items); err != nil {
		httputil.InternalError(w, "[recommendations] refresh", "failed to persist recommendations")
		return
	}

	resp := &apipb.ListRecommendationsResponse{Recommendations: recommendationsToProto(items)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[recommendations] refresh", "failed to encode response")
	}
}

// Create creates a manual recommendation.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req apipb.CreateRecommendationRequest
	if err := httputil.DecodeProtoJSON(r, &req); err != nil {
		httputil.BadRequest(w, "[recommendations] create", "invalid request body")
		return
	}
	if !httputil.ValidateProtoRequest(w, "[recommendations] create", "invalid request body", &req) {
		return
	}

	scenarioName := strings.TrimSpace(req.ScenarioName)
	description := strings.TrimSpace(req.Description)
	if scenarioName == "" || description == "" {
		httputil.BadRequest(w, "[recommendations] create", "scenarioName and description are required")
		return
	}

	items, err := h.store.Load()
	if err != nil {
		httputil.InternalError(w, "[recommendations] create", "failed to load recommendations")
		return
	}

	rec := Recommendation{
		ID:          idgen.Generate(),
		Scenario:    scenarioName,
		Type:        RecommendationType(req.Type),
		Description: description,
		Status:      StatusPending,
		Priority:    clampPriority(int(req.Priority)),
		Created:     time.Now().UTC().Format(time.RFC3339),
		Source:      "manual",
	}

	items = append(items, rec)
	if err := h.store.Save(items); err != nil {
		httputil.InternalError(w, "[recommendations] create", "failed to persist recommendations")
		return
	}

	resp := &apipb.RecommendationResponse{Recommendation: recommendationToProto(rec)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[recommendations] create", "failed to encode response")
	}
}

// Update updates recommendation status.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		httputil.BadRequest(w, "[recommendations] update", "id is required")
		return
	}

	var patch apipb.UpdateRecommendationRequest
	if err := httputil.DecodeProtoJSON(r, &patch); err != nil {
		httputil.BadRequest(w, "[recommendations] update", "invalid request body")
		return
	}

	if patch.Status == nil {
		httputil.BadRequest(w, "[recommendations] update", "status is required")
		return
	}
	if !httputil.ValidateProtoRequest(w, "[recommendations] update", "invalid request body", &patch) {
		return
	}

	items, err := h.store.Load()
	if err != nil {
		httputil.InternalError(w, "[recommendations] update", "failed to load recommendations")
		return
	}

	updated, found := updateStatus(items, id, RecommendationStatus(*patch.Status))
	if !found {
		httputil.NotFound(w, "[recommendations] update", "recommendation not found")
		return
	}

	if err := h.store.Save(updated); err != nil {
		httputil.InternalError(w, "[recommendations] update", "failed to persist recommendations")
		return
	}

	rec := findByID(updated, id)
	resp := &apipb.RecommendationResponse{Recommendation: recommendationToProto(rec)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[recommendations] update", "failed to encode response")
	}
}

// Start spawns an agent run for a recommendation.
func (h *Handler) Start(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		httputil.BadRequest(w, "[recommendations] start", "id is required")
		return
	}

	var req apipb.StartRecommendationRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := httputil.DecodeProtoJSON(r, &req); err != nil {
			httputil.BadRequest(w, "[recommendations] start", "invalid request body")
			return
		}
		if !httputil.ValidateProtoRequest(w, "[recommendations] start", "invalid request body", &req) {
			return
		}
	}

	cfg, err := h.settingsStore.Load()
	if err != nil {
		httputil.InternalError(w, "[recommendations] start", "failed to load settings")
		return
	}

	if cfg.RecommendationMode == "off" {
		httputil.Conflict(w, "[recommendations] start", "recommendation mode is off")
		return
	}

	items, err := h.store.Load()
	if err != nil {
		httputil.InternalError(w, "[recommendations] start", "failed to load recommendations")
		return
	}

	rec := findByID(items, id)
	if rec.ID == "" {
		httputil.NotFound(w, "[recommendations] start", "recommendation not found")
		return
	}

	if strings.TrimSpace(rec.TaskID) != "" || strings.TrimSpace(rec.RunID) != "" {
		httputil.Conflict(w, "[recommendations] start", "recommendation already started")
		return
	}

	service := h.agentService
	if service == nil {
		service = agentmanager.NewAgentService(agentmanager.DefaultServiceConfig())
		h.agentService = service
	}

	scopePath := strings.TrimSpace(readOptionalString(req.ScopePath))
	if scopePath == "" {
		if strings.TrimSpace(rec.Scenario) != "" {
			scopePath = filepath.Join("scenarios", rec.Scenario)
		} else {
			scopePath = "."
		}
	}

	projectRoot := strings.TrimSpace(readOptionalString(req.ProjectRoot))
	if projectRoot == "" {
		projectRoot = "."
	}

	createdBy := strings.TrimSpace(readOptionalString(req.CreatedBy))
	if createdBy == "" {
		createdBy = "swarm-manager"
	}

	result, err := service.SpawnRecommendation(r.Context(), agentmanager.RecommendationSpawnRequest{
		RecommendationID: rec.ID,
		Scenario:         rec.Scenario,
		Type:             string(rec.Type),
		Description:      rec.Description,
		Prompt:           strings.TrimSpace(readOptionalString(req.Prompt)),
		ScopePath:        scopePath,
		ProjectRoot:      projectRoot,
		CreatedBy:        createdBy,
	})
	if err != nil {
		if errors.Is(err, agentmanager.ErrNotAvailable) {
			httputil.ServiceUnavailable(w, "[recommendations] start", "agent-manager is not available")
			return
		}
		httputil.InternalError(w, "[recommendations] start", "failed to start recommendation")
		return
	}

	startedAt := result.CreatedAt
	if strings.TrimSpace(startedAt) == "" {
		startedAt = time.Now().UTC().Format(time.RFC3339)
	}

	rec.TaskID = result.TaskID
	rec.RunID = result.RunID
	rec.StartedAt = startedAt

	startedBy := "user"
	if cfg.RecommendationMode == "yolo" {
		if rec.Status == StatusPending {
			rec.Status = StatusApproved
		}
		rec.AutoApproved = true
		startedBy = "yolo"
	}
	rec.StartedBy = startedBy

	for i := range items {
		if items[i].ID == rec.ID {
			items[i] = rec
			break
		}
	}

	if err := h.store.Save(items); err != nil {
		httputil.InternalError(w, "[recommendations] start", "failed to persist recommendations")
		return
	}

	resp := &apipb.RecommendationResponse{Recommendation: recommendationToProto(rec)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[recommendations] start", "failed to encode response")
	}
}

func recommendationsToProto(items []Recommendation) []*domainpb.Recommendation {
	if len(items) == 0 {
		return nil
	}
	result := make([]*domainpb.Recommendation, 0, len(items))
	for _, item := range items {
		result = append(result, recommendationToProto(item))
	}
	return result
}

func recommendationToProto(rec Recommendation) *domainpb.Recommendation {
	protoRec := &domainpb.Recommendation{
		Id:           rec.ID,
		ScenarioName: rec.Scenario,
		Type:         string(rec.Type),
		Description:  rec.Description,
		Status:       string(rec.Status),
		Priority:     int32(rec.Priority),
		Created:      rec.Created,
	}
	if value := optionalString(rec.Source); value != nil {
		protoRec.Source = value
	}
	if value := optionalString(rec.TaskID); value != nil {
		protoRec.TaskId = value
	}
	if value := optionalString(rec.RunID); value != nil {
		protoRec.RunId = value
	}
	if value := optionalString(rec.StartedAt); value != nil {
		protoRec.StartedAt = value
	}
	if value := optionalString(rec.StartedBy); value != nil {
		protoRec.StartedBy = value
	}
	if rec.AutoApproved {
		protoRec.AutoApproved = optionalBool(rec.AutoApproved)
	}
	return protoRec
}

func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalBool(value bool) *bool {
	result := value
	return &result
}

func readOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizeList(items []Recommendation) []Recommendation {
	for i := range items {
		items[i].Scenario = strings.TrimSpace(items[i].Scenario)
		items[i].Description = strings.TrimSpace(items[i].Description)
		items[i].TaskID = strings.TrimSpace(items[i].TaskID)
		items[i].RunID = strings.TrimSpace(items[i].RunID)
		items[i].StartedAt = strings.TrimSpace(items[i].StartedAt)
		items[i].StartedBy = strings.TrimSpace(items[i].StartedBy)
		if items[i].Status == "" {
			items[i].Status = StatusPending
		}
		items[i].Priority = clampPriority(items[i].Priority)
		if items[i].Created == "" {
			items[i].Created = time.Now().UTC().Format(time.RFC3339)
		}
		if items[i].Source == "" {
			items[i].Source = "generated"
		}
	}
	return items
}

func isValidType(t RecommendationType) bool {
	switch t {
	case TypeTest, TypeFeature, TypeRefactor, TypeDocs:
		return true
	default:
		return false
	}
}

func isValidStatus(status RecommendationStatus) bool {
	switch status {
	case StatusPending, StatusApproved, StatusRejected:
		return true
	default:
		return false
	}
}

func filterRecommendations(items []Recommendation, r *http.Request) []Recommendation {
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))
	scenarioFilter := strings.TrimSpace(r.URL.Query().Get("scenario"))
	typeFilter := strings.TrimSpace(r.URL.Query().Get("type"))

	filtered := items
	if statusFilter != "" {
		filtered = filterByStatus(filtered, RecommendationStatus(statusFilter))
	}
	if scenarioFilter != "" {
		filtered = filterByScenario(filtered, scenarioFilter)
	}
	if typeFilter != "" {
		filtered = filterByType(filtered, RecommendationType(typeFilter))
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].Priority != filtered[j].Priority {
			return filtered[i].Priority < filtered[j].Priority
		}
		return filtered[i].Created > filtered[j].Created
	})
	return filtered
}

func filterByStatus(items []Recommendation, status RecommendationStatus) []Recommendation {
	if !isValidStatus(status) {
		return items
	}
	result := make([]Recommendation, 0, len(items))
	for _, rec := range items {
		if rec.Status == status {
			result = append(result, rec)
		}
	}
	return result
}

func filterByScenario(items []Recommendation, scenario string) []Recommendation {
	result := make([]Recommendation, 0, len(items))
	for _, rec := range items {
		if strings.EqualFold(rec.Scenario, scenario) {
			result = append(result, rec)
		}
	}
	return result
}

func filterByType(items []Recommendation, recType RecommendationType) []Recommendation {
	if !isValidType(recType) {
		return items
	}
	result := make([]Recommendation, 0, len(items))
	for _, rec := range items {
		if rec.Type == recType {
			result = append(result, rec)
		}
	}
	return result
}

func updateStatus(items []Recommendation, id string, status RecommendationStatus) ([]Recommendation, bool) {
	found := false
	for i := range items {
		if items[i].ID == id {
			items[i].Status = status
			found = true
			break
		}
	}
	return items, found
}

func findByID(items []Recommendation, id string) Recommendation {
	for _, item := range items {
		if item.ID == id {
			return item
		}
	}
	return Recommendation{}
}

func mergeRecommendations(existing, generated []Recommendation) []Recommendation {
	existingByID := make(map[string]Recommendation, len(existing))
	for _, rec := range existing {
		existingByID[rec.ID] = rec
	}

	merged := []Recommendation{}
	for _, rec := range generated {
		if prior, ok := existingByID[rec.ID]; ok {
			rec.Status = prior.Status
			rec.Created = prior.Created
			if prior.Source != "" {
				rec.Source = prior.Source
			}
			if prior.TaskID != "" {
				rec.TaskID = prior.TaskID
			}
			if prior.RunID != "" {
				rec.RunID = prior.RunID
			}
			if prior.StartedAt != "" {
				rec.StartedAt = prior.StartedAt
			}
			if prior.StartedBy != "" {
				rec.StartedBy = prior.StartedBy
			}
			if prior.AutoApproved {
				rec.AutoApproved = true
			}
		}
		merged = append(merged, rec)
	}

	for _, rec := range existing {
		if rec.Source == "manual" || rec.Status != StatusPending {
			if !containsID(merged, rec.ID) {
				merged = append(merged, rec)
			}
		}
	}

	return merged
}

func containsID(items []Recommendation, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return true
		}
	}
	return false
}

func applyYoloMode(items []Recommendation, cfg settings.Settings) []Recommendation {
	if cfg.RecommendationMode != "yolo" {
		return items
	}
	for i := range items {
		if items[i].Status == StatusPending && items[i].Priority >= 3 {
			items[i].Status = StatusApproved
		}
	}
	return items
}

func normalizeScenarioPriority(priority int) int {
	if priority <= 0 {
		return 3
	}
	if priority > 10 {
		return 5
	}
	return int((priority + 1) / 2)
}

func clampPriority(priority int) int {
	if priority < 1 {
		return 1
	}
	if priority > 5 {
		return 5
	}
	return priority
}

func stableID(scenario string, recType RecommendationType, token string) string {
	input := fmt.Sprintf("%s|%s|%s", strings.ToLower(scenario), recType, token)
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])
}

func countProblems(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	lines := strings.Split(string(data), "\n")
	count := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "|") || strings.HasPrefix(trimmed, "-") {
			if strings.Contains(trimmed, "TD-") || strings.Contains(trimmed, "BUG") || strings.Contains(trimmed, "TODO") {
				count++
			}
		}
	}
	return count
}

func loadRequirementsSummary(path string) (int, int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0
	}
	var payload struct {
		Summary struct {
			CompletionRate int `json:"completion_rate"`
			PassRate       int `json:"pass_rate"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0, 0
	}
	return payload.Summary.CompletionRate, payload.Summary.PassRate
}
