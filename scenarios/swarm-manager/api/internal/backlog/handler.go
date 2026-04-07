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
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/dispatch"
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
	ContinueRun(ctx context.Context, runID string, message string) error
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
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
	governanceProvider execution.GovernanceProvider
	eventDispatcher    dispatch.Invalidator
	eventLogger        EventLogger
	workshopTicker     *WorkshopTicker
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
	EmitBacklogArchived(entityID, previousStatus, archivedAt string)
	EmitBacklogUnarchived(entityID, archivedAt string)
	EmitBacklogDeleted(entityID string)
	EmitWorkshopRoundCompleted(entityID string, roundNumber int)
	EmitBacklogViewed(entityID, kind string)
	EmitClarificationStarted(entityID string, roundNumber int, itemID string, hasMessage bool)
	EmitClarificationResolved(entityID string, roundNumber int, itemID string, messageCount int, impactLevel string)
	EmitClarificationAction(entityID string, roundNumber int, itemID string, action string)
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

// SetGovernanceProvider injects a governance provider for execution service creation.
func (h *Handler) SetGovernanceProvider(gp execution.GovernanceProvider) {
	h.governanceProvider = gp
}

// SetEventDispatcher injects an optional graph invalidation dispatcher.
func (h *Handler) SetEventDispatcher(d dispatch.Invalidator) {
	h.eventDispatcher = d
}

// SetEventLogger injects an optional event logger for analytics tracking.
func (h *Handler) SetEventLogger(l EventLogger) {
	h.eventLogger = l
}

// StartWorkshopTicker starts the background ticker that fires deferred
// auto-advance spawns. It also recovers any pending advances from disk
// that survived a server restart.
func (h *Handler) StartWorkshopTicker() {
	t := newWorkshopTicker(h)
	h.workshopTicker = t
	t.RecoverPending()
	t.Start()
}

// StopWorkshopTicker stops the background ticker.
func (h *Handler) StopWorkshopTicker() {
	if h.workshopTicker != nil {
		h.workshopTicker.Stop()
	}
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
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive-item", h.Archive).Methods("PATCH")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive-item", h.Unarchive).Methods("DELETE")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/queue", h.Queue).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/research", h.Research).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/save", h.WorkshopSave).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/round", h.WorkshopDeleteRound).Methods("DELETE")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/reset", h.WorkshopReset).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/pending-advance", h.WorkshopCancelPendingAdvance).Methods("DELETE")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/clarification", h.CreateClarification).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/clarification/{threadId}", h.GetClarification).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/clarification/{threadId}/continue", h.ContinueClarification).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/clarification/{threadId}/action", h.ClarificationAction).Methods("POST")
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

// Update updates an existing backlog item.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "update")
	if !ok {
		return
	}

	existing, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "[backlog] update", apierr.NotFound("backlog item not found"))
			return
		}
		slog.Error("failed to load item for update", "name", name, "err", err)
		apierr.MapError(w, "[backlog] update", apierr.Internal("%s", httputil.TruncateErrorMessage(err, 240)))
		return
	}

	update, fields, err := decodeUpdateBacklogPatch(r)
	if err != nil {
		apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", err.Error()))
		return
	}
	if validationErr := validateUpdateBacklogItemRequest(update, fields, existing.Kind); validationErr != "" {
		apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", validationErr))
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
			apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", err.Error()))
			return
		}
		update.Effort = &normalized
	}

	applyUpdateBacklogPatch(&existing, update, fields)
	existing.Updated = time.Now().UTC().Format(time.RFC3339)

	if fields.Has(updateFieldInitiative) {
		if err := h.validateInitiativeReference(existing.Initiative); err != nil {
			apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", err.Error()))
			return
		}
	}

	if fields.Has(updateFieldDependsOn) && len(existing.DependsOn) > 0 {
		if err := h.store.ValidateDependencies(existing.DependsOn); err != nil {
			apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", err.Error()))
			return
		}
		if err := h.checkDependencyCycles(existing); err != nil {
			apierr.MapError(w, "[backlog] update", apierr.BadRequest("%s", err.Error()))
			return
		}
	}

	if err := h.store.SaveItem(existing); err != nil {
		slog.Error("failed to save item", "name", name, "err", err)
		apierr.MapError(w, "[backlog] update", apierr.Internal("failed to save backlog item"))
		return
	}

	if oldStatus != existing.Status || oldPriority != existing.Priority {
		slog.Info("item updated", "name", name, "old_status", oldStatus, "new_status", existing.Status, "old_priority", oldPriority, "new_priority", existing.Priority)
	} else {
		slog.Info("item updated", "name", name)
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
		!blockingDepStatuses[existing.Status] &&
		blockingDepStatuses[oldStatus] {
		cfg, cfgErr := settings.NewStore("").Load()
		if cfgErr != nil {
			slog.Warn("cascade settings load error, using defaults", "err", cfgErr)
			cfg = settings.DefaultSettings()
		}
		if workshop.ShouldCascade(cfg.AutoCascadeWorkshop) {
			go h.cascadeWorkshopTrigger(existing)
		}
	}

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(existing)}
	h.invalidateAllGraphLenses()
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] update", apierr.Internal("failed to encode response"))
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
		slog.Error("failed to delete item", "name", name, "err", err)
		apierr.MapError(w, "[backlog] delete", apierr.Internal("failed to delete backlog item"))
		return
	}

	slog.Info("item deleted", "name", name, "kind", kind)
	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogDeleted(string(kind) + "/" + name)
	}
	h.invalidateAllGraphLenses()
	w.WriteHeader(http.StatusNoContent)
}

// Archive sets archived_at on a backlog item. The item must be in a terminal
// status (completed or failed).
func (h *Handler) Archive(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "archive")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] archive", apierr.Internal("%s", err.Error()))
		return
	}

	if item.ArchivedAt != nil {
		resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
		if err := httputil.ProtoJSON(w, resp); err != nil {
			apierr.MapError(w, "[backlog] archive", apierr.Internal("failed to encode response"))
		}
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item.ArchivedAt = &now
	item.Updated = now

	if err := h.store.SaveItem(item); err != nil {
		apierr.MapError(w, "[backlog] archive", apierr.Internal("failed to save item"))
		return
	}

	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogArchived(string(kind)+"/"+name, string(item.Status), now)
	}
	h.invalidateAllGraphLenses()

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] archive", apierr.Internal("failed to encode response"))
	}
}

// Unarchive clears archived_at on a backlog item.
func (h *Handler) Unarchive(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "unarchive")
	if !ok {
		return
	}

	item, err := h.store.LoadItem(kind, name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			apierr.MapError(w, "", apierr.NotFound("backlog item not found"))
			return
		}
		apierr.MapError(w, "[backlog] unarchive", apierr.Internal("%s", err.Error()))
		return
	}

	if item.ArchivedAt == nil {
		resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
		if err := httputil.ProtoJSON(w, resp); err != nil {
			apierr.MapError(w, "[backlog] unarchive", apierr.Internal("failed to encode response"))
		}
		return
	}

	prevArchivedAt := *item.ArchivedAt
	item.ArchivedAt = nil
	item.Updated = time.Now().UTC().Format(time.RFC3339)

	if err := h.store.SaveItem(item); err != nil {
		apierr.MapError(w, "[backlog] unarchive", apierr.Internal("failed to save item"))
		return
	}

	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogUnarchived(string(kind)+"/"+name, prevArchivedAt)
	}
	h.invalidateAllGraphLenses()

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(item)}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] unarchive", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) parseKindAndName(w http.ResponseWriter, r *http.Request, action string) (BacklogKind, string, bool) {
	vars := mux.Vars(r)
	kindRaw := vars["kind"]
	name := vars["name"]
	kind, err := ParseBacklogKind(kindRaw)
	if err != nil {
		apierr.MapError(w, "[backlog] "+action, apierr.BadRequest("invalid kind"))
		return "", "", false
	}
	if strings.TrimSpace(name) == "" {
		apierr.MapError(w, "[backlog] "+action, apierr.BadRequest("name is required"))
		return "", "", false
	}
	return kind, name, true
}
