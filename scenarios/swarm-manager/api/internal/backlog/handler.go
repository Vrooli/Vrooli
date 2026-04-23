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
	"strings"
	"time"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/settings"
	"swarm-manager/internal/workshop"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

type AgentSpawner interface {
	IsEnabled() bool
	SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
	ContinueRun(ctx context.Context, runID string, message string) error
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
}

// AgentActivityChecker checks whether a backlog item has an active agent.
// Used by non-spawn guards (WorkshopDeleteRound, WorkshopReset) to prevent
// mutations while an agent is working on the item.
type AgentActivityChecker interface {
	HasActiveAgent(ctx context.Context, ownerKind, ownerName string) bool
}

// ItemTerminalHandler is invoked after the review-decide endpoint flips an
// item to a terminal status (completed / failed / needs_followup). The
// callback fires synchronously inside the HTTP handler so downstream effects
// (initiative-level review trigger, future per-item telemetry) see the
// decision before the response is sent, but implementations are expected to
// be cheap or self-gate expensive work to a goroutine.
type ItemTerminalHandler func(ctx context.Context, kind, name string, status BacklogStatus)

// Handler provides HTTP handlers for backlog operations.
type Handler struct {
	rootDir             string
	store               Store
	agentService        AgentSpawner
	activityChecker     AgentActivityChecker
	promptClient        promptmanager.Client
	initiativeAssigner  InitiativeAssigner
	executionQueuer     ExecutionQueuer
	policyProvider      execution.PolicyProvider
	governanceProvider  execution.GovernanceProvider
	eventDispatcher     dispatch.Invalidator
	eventLogger         EventLogger
	workshopTicker      *WorkshopTicker
	itemTerminalHandler ItemTerminalHandler
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
// If agentService implements AgentActivityChecker (e.g., *agentactivity.Service),
// it is also used for active-agent guards.
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
	if checker, ok := agentService.(AgentActivityChecker); ok {
		h.activityChecker = checker
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

// SetItemTerminalHandler wires a callback invoked after the review-decide
// endpoint flips an item to a terminal status. Passing nil clears the
// handler. The callback runs inside the request goroutine, so long-running
// work should self-dispatch.
func (h *Handler) SetItemTerminalHandler(f ItemTerminalHandler) {
	h.itemTerminalHandler = f
}

// SetAIIndexer wires an optional AI search indexer that receives fire-and-forget
// notifications from the underlying FileStore after every SaveItem/DeleteItem.
// Silently no-ops if the backing store is not a FileStore.
func (h *Handler) SetAIIndexer(indexer AIIndexer) {
	if fs, ok := h.store.(*FileStore); ok {
		fs.SetAIIndexer(indexer)
	}
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
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/validation", h.GetValidation).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/process-preflight", h.ProcessPreflight).Methods("GET")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive-item", h.Archive).Methods("PATCH")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/archive-item", h.Unarchive).Methods("DELETE")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/queue", h.Queue).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/review-decide", h.ReviewDecide).Methods("POST")
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
	if validationErr := validateUpdateBacklogItemRequest(update, fields, existing.Kind, existing.Status); validationErr != "" {
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

	ref := string(kind) + "/" + name
	initiativeChanged := fields.Has(updateFieldInitiative) && oldInitiative != existing.Initiative
	if initiativeChanged && h.initiativeAssigner != nil && oldInitiative != "" {
		if err := h.initiativeAssigner.ForgetItem(oldInitiative, ref); err != nil {
			slog.Error("failed to detach item from old initiative", "ref", ref, "initiative", oldInitiative, "err", err)
			apierr.MapError(w, "[backlog] update", apierr.Internal("failed to update old initiative membership"))
			return
		}
	}

	if err := h.store.SaveItem(existing); err != nil {
		if initiativeChanged && h.initiativeAssigner != nil && oldInitiative != "" {
			if rErr := h.initiativeAssigner.RememberItem(oldInitiative, ref); rErr != nil {
				slog.Error("failed to re-attach to old initiative after save failure", "ref", ref, "err", rErr)
			}
		}
		slog.Error("failed to save item", "name", name, "err", err)
		apierr.MapError(w, "[backlog] update", apierr.Internal("failed to save backlog item"))
		return
	}

	if initiativeChanged && h.initiativeAssigner != nil && existing.Initiative != "" {
		if err := h.initiativeAssigner.RememberItem(existing.Initiative, ref); err != nil {
			slog.Error("failed to attach item to new initiative", "ref", ref, "initiative", existing.Initiative, "err", err)
			apierr.MapError(w, "[backlog] update", apierr.Internal("failed to update new initiative membership"))
			return
		}
	}

	h.logAndEmitUpdate(kind, name, oldStatus, existing.Status, oldPriority, existing.Priority, oldEffort, existing.Effort, oldInitiative, existing.Initiative, oldDependsOn, existing.DependsOn)
	h.maybeManuallyAcceptExecution(r.Context(), kind, name, oldStatus, existing.Status)
	h.maybeCascadeWorkshop(oldStatus, existing)

	resp := &apipb.BacklogItemResponse{Item: backlogToProto(existing)}
	h.invalidateAllGraphLenses()
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] update", apierr.Internal("failed to encode response"))
	}
}

// logAndEmitUpdate logs the update and emits analytics events for changed fields.
func (h *Handler) logAndEmitUpdate(
	kind BacklogKind, name string,
	oldStatus, newStatus BacklogStatus,
	oldPriority, newPriority int,
	oldEffort, newEffort string,
	oldInitiative, newInitiative string,
	oldDeps, newDeps []string,
) {
	if oldStatus != newStatus || oldPriority != newPriority {
		slog.Info("item updated", "name", name, "old_status", oldStatus, "new_status", newStatus, "old_priority", oldPriority, "new_priority", newPriority)
	} else {
		slog.Info("item updated", "name", name)
	}

	if h.eventLogger == nil {
		return
	}
	entityID := string(kind) + "/" + name
	if oldStatus != newStatus {
		h.eventLogger.EmitBacklogStatusChanged(entityID, string(oldStatus), string(newStatus))
	}
	if oldPriority != newPriority {
		h.eventLogger.EmitBacklogPriorityChanged(entityID, oldPriority, newPriority)
	}
	if oldEffort != newEffort {
		h.eventLogger.EmitBacklogEffortChanged(entityID, oldEffort, newEffort)
	}
	if oldInitiative != newInitiative {
		h.eventLogger.EmitBacklogInitiativeChanged(entityID, oldInitiative, newInitiative)
	}
	h.emitDependencyChanges(entityID, oldDeps, newDeps)
}

// maybeManuallyAcceptExecution recognizes a user-initiated override of the
// agent's verdict: the user changed the backlog item from failed to completed
// without re-running it. Flip the latest failed/needs_fixup execution to
// Completed with ManuallyAccepted=true so Agent-tab stats count the run as a
// success and surface the human override separately.
func (h *Handler) maybeManuallyAcceptExecution(
	ctx context.Context,
	kind BacklogKind, name string,
	oldStatus, newStatus BacklogStatus,
) {
	if oldStatus != StatusFailed || newStatus != StatusCompleted {
		return
	}
	if h.executionQueuer == nil {
		return
	}
	ref := string(kind) + "/" + name
	execID, accepted, err := h.executionQueuer.ManuallyAcceptLatestForBacklog(ctx, string(kind), name, "user", "user accepted failed run via backlog status change")
	if err != nil {
		slog.Error("manual-accept failed", "ref", ref, "err", err)
		return
	}
	if accepted {
		slog.Info("execution manually accepted", "ref", ref, "execution_id", execID)
	}
}

// maybeCascadeWorkshop triggers workshops for dependents when a status
// transition unblocks them.
func (h *Handler) maybeCascadeWorkshop(oldStatus BacklogStatus, item BacklogItem) {
	if oldStatus == item.Status {
		return
	}
	if !blockingDepStatuses[oldStatus] || blockingDepStatuses[item.Status] {
		return
	}
	cfg, cfgErr := settings.NewStore("").Load()
	if cfgErr != nil {
		slog.Warn("cascade settings load error, using defaults", "err", cfgErr)
		cfg = settings.DefaultSettings()
	}
	if workshop.ShouldCascade(cfg.AutoCascadeWorkshop) {
		go h.cascadeWorkshopTrigger(item)
	}
}

// Delete deletes a backlog item and cascades referential integrity:
//   - Removes the item's "kind/name" ref from every other item's depends_on.
//   - Removes the ref from its enclosing initiative's items[] list.
//
// Cascade runs before the item file is deleted so that a partial failure
// leaves a consistent "item still exists, references intact" state. After
// the item file is removed, the depends_on sweep is run as a best-effort
// cleanup of refs that now point at a non-existent item.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "delete")
	if !ok {
		return
	}

	existing, err := h.store.LoadItem(kind, name)
	if errors.Is(err, ErrNotFound) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err != nil {
		slog.Error("failed to load item for delete", "name", name, "err", err)
		apierr.MapError(w, "[backlog] delete", apierr.Internal("failed to load backlog item"))
		return
	}

	ref := string(kind) + "/" + name
	if strings.TrimSpace(existing.Initiative) != "" && h.initiativeAssigner != nil {
		if err := h.initiativeAssigner.ForgetItem(existing.Initiative, ref); err != nil {
			slog.Error("failed to forget item from initiative", "ref", ref, "initiative", existing.Initiative, "err", err)
			apierr.MapError(w, "[backlog] delete", apierr.Internal("failed to update initiative membership"))
			return
		}
	}

	if err := h.store.DeleteItem(kind, name); err != nil {
		if existing.Initiative != "" && h.initiativeAssigner != nil {
			if rollbackErr := h.initiativeAssigner.RememberItem(existing.Initiative, ref); rollbackErr != nil {
				slog.Error("failed to roll back initiative membership after delete failure", "ref", ref, "err", rollbackErr)
			}
		}
		slog.Error("failed to delete item", "name", name, "err", err)
		apierr.MapError(w, "[backlog] delete", apierr.Internal("failed to delete backlog item"))
		return
	}

	if n, err := h.store.RemoveDependencyRef(ref); err != nil {
		slog.Error("failed to clean up dependency references", "ref", ref, "err", err)
	} else if n > 0 {
		slog.Info("cleaned up dependency references", "ref", ref, "updated_items", n)
	}

	slog.Info("item deleted", "name", name, "kind", kind)
	if h.eventLogger != nil {
		h.eventLogger.EmitBacklogDeleted(ref)
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
