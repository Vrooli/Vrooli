// Package backlog provides HTTP handlers for backlog management.
//
// Backlog items are stored as git-tracked folders with spec.json files in
// scenarios/swarm-manager/{ideas|research|fix|execute}/. This handler provides
// CRUD operations, file access, agent spawning, and conversion between backlog kinds.
//
// DOC: docs/concepts/ARCHITECTURE.md#key-flows
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/INVARIANTS.md
//
// Related PRD targets: OT-P0-001, OT-P0-002
package backlog

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/workshop"
)

type AgentSpawner interface {
	IsEnabled() bool
	SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
}

// Handler provides HTTP handlers for backlog operations.
type Handler struct {
	rootDir            string
	store              Store
	agentService       AgentSpawner
	promptClient       promptmanager.Client
	initiativeAssigner InitiativeAssigner
	executionQueuer    ExecutionQueuer
	policyProvider     execution.PolicyProvider
	eventDispatcher    EventDispatcher
	eventLogger        EventLogger
}

// EventDispatcher emits graph invalidation events for graph projections.
type EventDispatcher interface {
	DispatchInvalidate(lenses ...string)
}

// EventLogger records state-change events for analytics.
type EventLogger interface {
	EmitBacklogCreated(entityID, kind, status string, priority int, initiative, effort string)
	EmitBacklogStatusChanged(entityID, from, to string)
	EmitBacklogPriorityChanged(entityID string, from, to int)
	EmitBacklogEffortChanged(entityID, from, to string)
	EmitBacklogDependencyAdded(entityID, target string)
	EmitBacklogDependencyRemoved(entityID, target string)
	EmitBacklogInitiativeChanged(entityID, from, to string)
	EmitBacklogArchived(entityID string)
	EmitWorkshopRoundCompleted(entityID string, roundNumber int)
}

// NewHandler creates a new backlog handler.
// If rootDir is empty, it defaults to the scenario root directory.
func NewHandler(rootDir string) *Handler {
	if rootDir == "" {
		rootDir = pathutil.ResolveScenarioRoot("swarm-manager")
	}
	return &Handler{
		rootDir:      rootDir,
		store:        NewFileStore(rootDir),
		agentService: nil,
		promptClient: promptmanager.NewHTTPClient(),
	}
}

// NewHandlerWithClients creates a new backlog handler with custom dependencies.
func NewHandlerWithClients(rootDir string, agentService AgentSpawner, promptClient promptmanager.Client) *Handler {
	if rootDir == "" {
		rootDir = pathutil.ResolveScenarioRoot("swarm-manager")
	}
	h := &Handler{
		rootDir:      rootDir,
		store:        NewFileStore(rootDir),
		agentService: agentService,
		promptClient: promptClient,
	}
	if h.promptClient == nil {
		h.promptClient = promptmanager.NewHTTPClient()
	}
	return h
}

// Store returns the underlying backlog store for cross-package use (e.g.,
// initiative rollup computation).
func (h *Handler) Store() Store {
	return h.store
}

// SetPolicyProvider injects a policy provider for execution service creation.
func (h *Handler) SetPolicyProvider(pp execution.PolicyProvider) {
	h.policyProvider = pp
}

// SetEventDispatcher injects an optional graph invalidation dispatcher.
func (h *Handler) SetEventDispatcher(d EventDispatcher) {
	h.eventDispatcher = d
}

// SetEventLogger injects an optional event logger for analytics tracking.
func (h *Handler) SetEventLogger(l EventLogger) {
	h.eventLogger = l
}

func (h *Handler) emitDependencyChanges(entityID string, oldDeps, newDeps []string) {
	old := make(map[string]bool, len(oldDeps))
	for _, d := range oldDeps {
		old[d] = true
	}
	cur := make(map[string]bool, len(newDeps))
	for _, d := range newDeps {
		cur[d] = true
	}
	for d := range cur {
		if !old[d] {
			h.eventLogger.EmitBacklogDependencyAdded(entityID, d)
		}
	}
	for d := range old {
		if !cur[d] {
			h.eventLogger.EmitBacklogDependencyRemoved(entityID, d)
		}
	}
}

func (h *Handler) invalidateAllGraphLenses() {
	if h.eventDispatcher == nil {
		return
	}
	h.eventDispatcher.DispatchInvalidate("topology", "flow", "operations")
}

func validateCreateBacklogItemRequest(req *apipb.CreateBacklogItemRequest) string {
	if strings.TrimSpace(req.Title) == "" {
		return "title is required"
	}
	if strings.TrimSpace(req.Kind) == "" {
		return "kind is required"
	}
	if req.Priority != nil {
		if *req.Priority < 1 || *req.Priority > 10 {
			return "priority must be between 1 and 10"
		}
	}
	return ""
}

func normalizeCreateBacklogItemRequest(req *apipb.CreateBacklogItemRequest) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		req.Name = strings.TrimSpace(req.Title)
	}
	req.Kind = strings.ToLower(strings.TrimSpace(req.Kind))
	if req.Effort != nil {
		normalized := strings.ToUpper(strings.TrimSpace(*req.Effort))
		if normalized == "" {
			req.Effort = nil
		} else {
			req.Effort = &normalized
		}
	}
}

func (h *Handler) validateInitiativeReference(name string) error {
	if strings.TrimSpace(name) == "" || h.initiativeAssigner == nil {
		return nil
	}
	if _, err := h.initiativeAssigner.Get(strings.TrimSpace(name)); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Errorf("initiative %q does not exist", strings.TrimSpace(name))
		}
		return fmt.Errorf("failed to load initiative %q: %w", strings.TrimSpace(name), err)
	}
	return nil
}

// RegisterRoutes registers the backlog API routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/backlog", h.List).Methods("GET")
	r.HandleFunc("/api/v1/backlog", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/backlog/batch", h.BatchCreate).Methods("POST")
	r.HandleFunc("/api/v1/backlog/batch/queue", h.BatchQueue).Methods("POST")
	r.HandleFunc("/api/v1/backlog/summary", h.BacklogSummary).Methods("GET")
	r.HandleFunc("/api/v1/backlog/feedback-summary", h.FeedbackSummary).Methods("GET")
	r.HandleFunc("/api/v1/backlog/maturity-summary", h.MaturitySummary).Methods("GET")
	r.HandleFunc("/api/v1/backlog/pending-questions", h.PendingQuestions).Methods("GET")
	r.HandleFunc("/api/v1/backlog/validate-globs", h.ValidateGlobs).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}", h.Get).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}", h.Update).Methods("PATCH")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}", h.Delete).Methods("DELETE")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/files", h.ListFiles).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/files", h.UploadFile).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/files", h.OperateFile).Methods("PATCH")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/files/{filepath:.*}", h.GetFileContent).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/process-preflight", h.ProcessPreflight).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/queue", h.Queue).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/research", h.Research).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/save", h.WorkshopSave).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/round", h.WorkshopDeleteRound).Methods("DELETE")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive/targets", h.GetArchiveTargets).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive/targets", h.CreateTargetHandler).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive/targets/{targetId}", h.UpdateTargetHandler).Methods("PUT")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive/targets/{targetId}", h.DeleteTargetHandler).Methods("DELETE")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive/requirements", h.CreateModuleHandler).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive/requirements/{moduleId}", h.UpdateModuleRequirementsHandler).Methods("PUT")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive/requirements/{moduleId}/meta", h.UpdateModuleMetaHandler).Methods("PUT")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive/requirements/{moduleId}", h.DeleteModuleHandler).Methods("DELETE")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive/review", h.BatchReviewHandler).Methods("PUT")
	r.HandleFunc("/api/v1/backlog/export", h.Export).Methods("POST")
	r.HandleFunc("/api/v1/backlog/import", h.Import).Methods("POST")
}

// List returns all backlog items.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	kinds, err := parseKindsQuery(r)
	if err != nil {
		httputil.BadRequest(w, "[backlog] list", err.Error())
		return
	}

	statusFilter, err := parseStatusesQuery(r)
	if err != nil {
		httputil.BadRequest(w, "[backlog] list", err.Error())
		return
	}

	items, err := h.store.LoadAll(kinds)
	if err != nil {
		httputil.InternalError(w, "[backlog] list", err.Error())
		return
	}

	items = filterByStatus(items, statusFilter)

	// Sort by priority (ascending) then by updated (descending)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		return items[i].Updated > items[j].Updated
	})

	protoItems := make([]*domainpb.BacklogItem, 0, len(items))
	for _, item := range items {
		protoItems = append(protoItems, backlogToProto(item))
	}

	resp := &apipb.ListBacklogItemsResponse{Items: protoItems}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] list", "failed to encode response")
	}
}

// Get returns a single backlog item by name.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "get")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.NotFound(w, "", "backlog item not found")
			return
		}
		httputil.InternalError(w, "[backlog] get", err.Error())
		return
	}

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] get", "failed to encode response")
	}
}

// Create creates a new backlog item.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req apipb.CreateBacklogItemRequest
	if err := httputil.DecodeProtoJSONStrict(r, &req); err != nil {
		httputil.BadRequest(w, "[backlog] create", "invalid request body")
		return
	}
	normalizeCreateBacklogItemRequest(&req)
	if !httputil.ValidateProtoRequest(w, "[backlog] create", "invalid request body", &req) {
		return
	}
	if validationErr := validateCreateBacklogItemRequest(&req); validationErr != "" {
		httputil.BadRequest(w, "[backlog] create", validationErr)
		return
	}

	kind, err := ParseBacklogKind(req.Kind)
	if err != nil {
		httputil.BadRequest(w, "[backlog] create", err.Error())
		return
	}

	// Sanitize name (folder-safe). Allow title fallback.
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = req.Title
	}
	name = sanitizeName(name)
	if name == "" {
		httputil.BadRequest(w, "[backlog] create", "name is required")
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	priority := 5
	if req.Priority != nil {
		priority = int(*req.Priority)
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	dependsOn := req.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}
	initiative := ""
	if req.Initiative != nil {
		initiative = strings.TrimSpace(*req.Initiative)
	}
	if err := h.validateInitiativeReference(initiative); err != nil {
		httputil.BadRequest(w, "[backlog] create", err.Error())
		return
	}

	effort := ""
	if req.Effort != nil {
		normalized, err := validateEffort(*req.Effort)
		if err != nil {
			httputil.BadRequest(w, "[backlog] create", err.Error())
			return
		}
		effort = normalized
	}

	if err := validateGlobs(req.AcceptanceAllow); err != nil {
		httputil.BadRequest(w, "[backlog] create", "acceptance_allow: "+err.Error())
		return
	}
	if err := validateGlobs(req.AcceptanceDeny); err != nil {
		httputil.BadRequest(w, "[backlog] create", "acceptance_deny: "+err.Error())
		return
	}

	item := BacklogItem{
		Name:            name,
		Title:           req.Title,
		Description:     description,
		Status:          StatusBacklog,
		Priority:        priority,
		Tags:            tags,
		Created:         now,
		Updated:         now,
		Kind:            kind,
		DependsOn:       dependsOn,
		Initiative:      initiative,
		Effort:          effort,
		AcceptanceAllow: req.AcceptanceAllow,
		AcceptanceDeny:  req.AcceptanceDeny,
	}

	itemDir := h.store.ItemDir(kind, name)
	if err := os.Mkdir(itemDir, 0o755); err != nil {
		if os.IsExist(err) {
			httputil.Conflict(w, "[backlog] create", "backlog item already exists")
			return
		}
		// Parent dir may not exist for the first item of this kind — ensure it, then retry.
		if mkErr := os.MkdirAll(filepath.Dir(itemDir), 0o755); mkErr != nil {
			log.Printf("[backlog] create: failed to create parent directory for %q: %v", name, mkErr)
			httputil.InternalError(w, "[backlog] create", "failed to create backlog directory")
			return
		}
		if retryErr := os.Mkdir(itemDir, 0o755); retryErr != nil {
			if os.IsExist(retryErr) {
				httputil.Conflict(w, "[backlog] create", "backlog item already exists")
				return
			}
			log.Printf("[backlog] create: failed to create directory for %q: %v", name, retryErr)
			httputil.InternalError(w, "[backlog] create", "failed to create backlog directory")
			return
		}
	}

	// Validate dependencies exist and check for cycles.
	if len(item.DependsOn) > 0 {
		if err := h.store.ValidateDependencies(item.DependsOn); err != nil {
			_ = os.RemoveAll(itemDir)
			httputil.BadRequest(w, "[backlog] create", err.Error())
			return
		}
		if err := h.checkDependencyCycles(item); err != nil {
			_ = os.RemoveAll(itemDir)
			httputil.BadRequest(w, "[backlog] create", err.Error())
			return
		}
	}

	if err := h.store.SaveItem(item); err != nil {
		_ = os.RemoveAll(itemDir)
		log.Printf("[backlog] create: failed to save %q: %v", name, err)
		httputil.InternalError(w, "[backlog] create", "failed to save backlog item")
		return
	}

	// Auto-initialize workshop for new items (unless disabled in settings or blocked by deps).
	h.maybeAutoWorkshop(item, false)

	log.Printf("[backlog] created: %q (kind=%s, priority=%d, status=%s)", name, kind, priority, StatusBacklog)
	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogCreated(string(kind)+"/"+name, string(kind), string(StatusBacklog), priority, item.Initiative, item.Effort)
	}
	h.invalidateAllGraphLenses()
	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSONWithStatus(w, http.StatusCreated, resp); err != nil {
		httputil.InternalError(w, "[backlog] create", "failed to encode response")
	}
}

// maybeAutoWorkshop checks the global auto_initialize_workshop setting,
// dependency readiness, and spawns the first workshop round asynchronously
// if appropriate.
func (h *Handler) maybeAutoWorkshop(item BacklogItem, forceOverride bool) {
	cfg, err := settings.NewStore("").Load()
	if err != nil {
		log.Printf("[backlog] auto-workshop: settings load error for %s/%s: %v, using defaults", item.Kind, item.Name, err)
		cfg = settings.DefaultSettings()
	}
	if !workshop.ShouldAutoInitialize(cfg.AutoInitializeWorkshop) {
		return
	}
	if !forceOverride && len(item.DependsOn) > 0 {
		depStatuses, err := h.store.CheckWorkshopDependencies(item.DependsOn)
		if err != nil {
			log.Printf("[backlog] auto-workshop: dep check error for %s/%s: %v, proceeding anyway", item.Kind, item.Name, err)
		} else {
			result := workshop.CheckWorkshopDependencies(depStatuses)
			if result.Blocked {
				log.Printf("[backlog] auto-workshop: blocked for %s/%s by deps: %v", item.Kind, item.Name, result.BlockingDeps)
				return
			}
		}
	}
	go func() {
		_, _, spawnErr := h.spawnWorkshopAsync(item, ResearchModeInitialize)
		if spawnErr != nil {
			log.Printf("[backlog] auto-init: failed for %s/%s: %v", item.Kind, item.Name, spawnErr)
		}
	}()
}

// cascadeWorkshopTrigger finds items that depend on the given item and
// auto-triggers their workshops if all their dependencies are now met.
// Only triggers for items still in "backlog" status with no existing
// workshop rounds.
func (h *Handler) cascadeWorkshopTrigger(readyItem BacklogItem) {
	readyKey := string(readyItem.Kind) + "/" + readyItem.Name

	allItems, err := h.store.LoadAll(nil)
	if err != nil {
		log.Printf("[backlog] cascade: failed to load items: %v", err)
		return
	}

	for _, item := range allItems {
		if item.Status != StatusBacklog {
			continue
		}
		dependsOnReady := false
		for _, dep := range item.DependsOn {
			if dep == readyKey {
				dependsOnReady = true
				break
			}
		}
		if !dependsOnReady {
			continue
		}

		depStatuses, err := h.store.CheckWorkshopDependencies(item.DependsOn)
		if err != nil {
			log.Printf("[backlog] cascade: dep check failed for %s/%s: %v", item.Kind, item.Name, err)
			continue
		}
		result := workshop.CheckWorkshopDependencies(depStatuses)
		if result.Blocked {
			continue
		}

		itemDir := h.store.ItemDir(item.Kind, item.Name)
		_, roundCount, _ := workshop.LoadLatestRound(itemDir)
		if roundCount > 0 {
			continue
		}

		log.Printf("[backlog] cascade: triggering workshop for %s/%s (unblocked by %s)", item.Kind, item.Name, readyKey)
		go func(it BacklogItem) {
			_, _, spawnErr := h.spawnWorkshopAsync(it, ResearchModeInitialize)
			if spawnErr != nil {
				log.Printf("[backlog] cascade: failed for %s/%s: %v", it.Kind, it.Name, spawnErr)
			}
		}(item)
	}
}

// Update updates an existing backlog item.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "update")
	if !ok {
		return
	}

	existing, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.NotFound(w, "[backlog] update", "backlog item not found")
			return
		}
		log.Printf("[backlog] update: failed to load %q: %v", name, err)
		httputil.InternalError(w, "[backlog] update", httputil.TruncateErrorMessage(err, 240))
		return
	}

	update, fields, err := decodeUpdateBacklogPatch(r)
	if err != nil {
		httputil.BadRequest(w, "[backlog] update", err.Error())
		return
	}
	if validationErr := validateUpdateBacklogItemRequest(update, fields, existing.Kind); validationErr != "" {
		httputil.BadRequest(w, "[backlog] update", validationErr)
		return
	}

	oldStatus := existing.Status
	oldPriority := existing.Priority
	oldEffort := existing.Effort
	oldInitiative := existing.Initiative
	oldDependsOn := append([]string(nil), existing.DependsOn...)

	if fields.Has(updateFieldEffort) {
		normalized, err := validateEffort(update.GetEffort())
		if err != nil {
			httputil.BadRequest(w, "[backlog] update", err.Error())
			return
		}
		update.Effort = &normalized
	}

	applyUpdateBacklogPatch(&existing, update, fields)
	existing.Updated = time.Now().UTC().Format(time.RFC3339)

	if fields.Has(updateFieldInitiative) {
		if err := h.validateInitiativeReference(existing.Initiative); err != nil {
			httputil.BadRequest(w, "[backlog] update", err.Error())
			return
		}
	}

	if fields.Has(updateFieldDependsOn) && len(existing.DependsOn) > 0 {
		if err := h.store.ValidateDependencies(existing.DependsOn); err != nil {
			httputil.BadRequest(w, "[backlog] update", err.Error())
			return
		}
		if err := h.checkDependencyCycles(existing); err != nil {
			httputil.BadRequest(w, "[backlog] update", err.Error())
			return
		}
	}

	if err := h.store.SaveItem(existing); err != nil {
		log.Printf("[backlog] update: failed to save %q: %v", name, err)
		httputil.InternalError(w, "[backlog] update", "failed to save backlog item")
		return
	}

	if oldStatus != existing.Status || oldPriority != existing.Priority {
		log.Printf("[backlog] updated: %q (status=%s→%s, priority=%d→%d)", name, oldStatus, existing.Status, oldPriority, existing.Priority)
	} else {
		log.Printf("[backlog] updated: %q", name)
	}

	if h.eventLogger != nil {
		entityID := string(kind) + "/" + name
		if oldStatus != existing.Status {
			h.eventLogger.EmitBacklogStatusChanged(entityID, string(oldStatus), string(existing.Status))
		}
		if oldPriority != existing.Priority {
			h.eventLogger.EmitBacklogPriorityChanged(entityID, oldPriority, existing.Priority)
		}
		if oldEffort != existing.Effort {
			h.eventLogger.EmitBacklogEffortChanged(entityID, oldEffort, existing.Effort)
		}
		if oldInitiative != existing.Initiative {
			h.eventLogger.EmitBacklogInitiativeChanged(entityID, oldInitiative, existing.Initiative)
		}
		h.emitDependencyChanges(entityID, oldDependsOn, existing.DependsOn)
	}

	// Cascade: when status transitions to workshop-ready, trigger
	// workshops for dependents that were previously blocked.
	if oldStatus != existing.Status &&
		workshop.IsWorkshopReady(string(existing.Status)) &&
		!workshop.IsWorkshopReady(string(oldStatus)) {
		cfg, cfgErr := settings.NewStore("").Load()
		if cfgErr != nil {
			log.Printf("[backlog] cascade: settings load error: %v, using defaults", cfgErr)
			cfg = settings.DefaultSettings()
		}
		if workshop.ShouldCascade(cfg.AutoCascadeWorkshop) {
			go h.cascadeWorkshopTrigger(existing)
		}
	}

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(existing)}
	h.invalidateAllGraphLenses()
	if err := httputil.ProtoJSON(w, resp); err != nil {
		httputil.InternalError(w, "[backlog] update", "failed to encode response")
	}
}

// Delete deletes a backlog item by name.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "delete")
	if !ok {
		return
	}

	// Idempotent delete: if the item doesn't exist, return 204 immediately.
	itemDir := h.store.ItemDir(kind, name)
	if _, err := h.store.LoadItem(kind, name); errors.Is(err, ErrNotFound) {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := os.RemoveAll(itemDir); err != nil {
		log.Printf("[backlog] delete: failed to delete %q: %v", name, err)
		httputil.InternalError(w, "[backlog] delete", "failed to delete backlog item")
		return
	}

	log.Printf("[backlog] deleted: %q (kind=%s)", name, kind)
	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogArchived(string(kind) + "/" + name)
	}
	h.invalidateAllGraphLenses()
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) parseKindAndName(w http.ResponseWriter, r *http.Request, action string) (BacklogKind, string, bool) {
	vars := mux.Vars(r)
	kindRaw := vars["kind"]
	name := vars["name"]
	kind, err := ParseBacklogKind(kindRaw)
	if err != nil {
		httputil.BadRequest(w, "[backlog] "+action, "invalid kind")
		return "", "", false
	}
	if strings.TrimSpace(name) == "" {
		httputil.BadRequest(w, "[backlog] "+action, "name is required")
		return "", "", false
	}
	return kind, name, true
}

// checkDependencyCycles builds a dependency graph from all existing items plus
// the given item and checks for cycles.
func (h *Handler) checkDependencyCycles(item BacklogItem) error {
	items, err := h.store.LoadAll(nil)
	if err != nil {
		return fmt.Errorf("failed to load items for cycle check: %w", err)
	}

	g := depgraph.New()
	itemKey := string(item.Kind) + "/" + item.Name
	g.AddNode(itemKey, item.DependsOn)

	for _, existing := range items {
		key := string(existing.Kind) + "/" + existing.Name
		if key == itemKey {
			continue // use the new/updated version
		}
		g.AddNode(key, existing.DependsOn)
	}

	if cycle, found := g.DetectCycle(); found {
		return fmt.Errorf("dependency cycle detected: %s", strings.Join(cycle, " -> "))
	}
	return nil
}

func parseKindsQuery(r *http.Request) ([]BacklogKind, error) {
	query := r.URL.Query()
	raw := strings.TrimSpace(query.Get("kinds"))
	if raw == "" {
		raw = strings.TrimSpace(query.Get("kind"))
	}
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	kinds := make([]BacklogKind, 0, len(parts))
	for _, part := range parts {
		kind, err := ParseBacklogKind(part)
		if err != nil {
			return nil, err
		}
		kinds = append(kinds, kind)
	}
	return kinds, nil
}

// parseStatusesQuery reads the "statuses" (or "status") query parameter.
// Returns nil when no filter is specified (caller should apply default).
// The special value "all" returns an empty slice signaling no filtering.
func parseStatusesQuery(r *http.Request) ([]BacklogStatus, error) {
	query := r.URL.Query()
	raw := strings.TrimSpace(query.Get("statuses"))
	if raw == "" {
		raw = strings.TrimSpace(query.Get("status"))
	}
	if raw == "" {
		return nil, nil
	}
	if strings.EqualFold(raw, "all") {
		return []BacklogStatus{}, nil
	}

	parts := strings.Split(raw, ",")
	statuses := make([]BacklogStatus, 0, len(parts))
	for _, part := range parts {
		s := BacklogStatus(strings.TrimSpace(part))
		if s == "" {
			continue
		}
		statuses = append(statuses, s)
	}
	return statuses, nil
}

// filterByStatus applies status filtering to a list of backlog items.
//   - nil (no query param): exclude archived items (default)
//   - empty slice (status=all): no filtering, return everything
//   - non-empty slice: include only items matching one of the given statuses
func filterByStatus(items []BacklogItem, statuses []BacklogStatus) []BacklogItem {
	if statuses != nil && len(statuses) == 0 {
		return items
	}

	filtered := make([]BacklogItem, 0, len(items))
	if statuses == nil {
		for _, item := range items {
			if item.Status != StatusArchived {
				filtered = append(filtered, item)
			}
		}
		return filtered
	}

	allow := make(map[BacklogStatus]bool, len(statuses))
	for _, s := range statuses {
		allow[s] = true
	}
	for _, item := range items {
		if allow[item.Status] {
			filtered = append(filtered, item)
		}
	}
	return filtered
}
