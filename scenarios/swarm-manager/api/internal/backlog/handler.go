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
	"fmt"
	"net/http"
	"strings"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/execution"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/runtimepaths"

	"github.com/gorilla/mux"
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

// BacklogRecordRequest carries the context the review-decide hook hands to the
// records capture seam so it can write a FILLED, searchable record (not an empty
// stub): the item's own title/description seed the record's trigger/approach,
// the acceptance globs derive the target scenario, and the initiative links it
// back. The hook already has the item loaded, so passing it here costs nothing
// and lets the record be born indexed instead of an empty stub nobody fills.
type BacklogRecordRequest struct {
	Kind            string
	Name            string
	Title           string
	Description     string
	AcceptanceAllow []string
	Initiative      string
	Status          BacklogStatus
	DecidedBy       string
}

// RecordCreator is the records capture seam: when a backlog item reaches a
// terminal status (and the client did not pass --no-record), the review-decide
// endpoint asks the implementation to write a record capturing the work, drawn
// from the item itself. The returned id is surfaced to the client so the agent
// can enrich it via `records supersede` (the record is born filled+immutable, so
// `records edit` is not the amend path). Errors are logged and dropped; capture
// must never block or fail the terminal transition.
//
// seam: backlog.RecordCreator
type RecordCreator interface {
	CreateBacklogRecord(ctx context.Context, req BacklogRecordRequest) (recordID string, err error)
}

// Handler provides HTTP handlers for backlog operations.
//
// dataRoot is where on-disk item folders live (runtime home,
// `~/.vrooli/data/vrooli/swarm-manager/<kind>/...`). repoRoot is the
// scenario source path used purely as a repo anchor (e.g. by
// validate_globs.go to resolve `.vrooli/repo-contract.json`).
type Handler struct {
	dataRoot             string
	repoRoot             string
	store                Store
	agentService         AgentSpawner
	activityChecker      AgentActivityChecker
	promptClient         promptmanager.Client
	initiativeAssigner   InitiativeAssigner
	sessionArtifacts     sessionArtifactRecorder
	executionQueuer      ExecutionQueuer
	policyProvider       execution.PolicyProvider
	governanceProvider   execution.GovernanceProvider
	eventDispatcher      dispatch.Invalidator
	eventLogger          EventLogger
	workshopTicker       *WorkshopTicker
	itemTerminalHandler  ItemTerminalHandler
	recordCreator        RecordCreator
	reviewRoundInspector ReviewRoundInspector
}

// EventLogger records state-change events for analytics.
type EventLogger interface {
	EmitBacklogCreated(entityID, kind, status string, priority int, initiative, effort string)
	EmitBacklogCreatedFromSource(entityID, kind, status string, priority int, initiative, effort, actorType, actorID string)
	EmitBacklogStatusChanged(entityID, from, to string)
	EmitBacklogPriorityChanged(entityID string, from, to int)
	EmitBacklogEffortChanged(entityID, from, to string)
	EmitBacklogDependencyAdded(entityID, target string)
	EmitBacklogDependencyRemoved(entityID, target string)
	EmitBacklogInitiativeChanged(entityID, from, to string)
	EmitBacklogArchived(entityID, previousStatus, archivedAt string)
	EmitBacklogUnarchived(entityID, archivedAt string)
	EmitBacklogDeleted(entityID string)
	EmitWorkshopRoundCompleted(entityID string, payload eventlog.WorkshopRoundPayload)
	EmitBacklogViewed(entityID, kind string)
	EmitClarificationStarted(entityID string, roundNumber int, itemID string, hasMessage bool)
	EmitClarificationResolved(entityID string, roundNumber int, itemID string, messageCount int, impactLevel string)
	EmitClarificationAction(entityID string, roundNumber int, itemID string, action string)
}

// NewHandler creates a new backlog handler.
// Empty dataRoot defaults to runtimepaths.DataPath("");
// empty repoRoot defaults to pathutil.ResolveScenarioRoot("swarm-manager").
func NewHandler(dataRoot, repoRoot string) *Handler {
	dataRoot = resolveDataRootOrDefault(dataRoot)
	repoRoot = resolveRepoRootOrDefault(repoRoot)
	return &Handler{
		dataRoot:     dataRoot,
		repoRoot:     repoRoot,
		store:        NewFileStore(dataRoot),
		agentService: nil,
		promptClient: promptmanager.NewHTTPClient(),
	}
}

// NewHandlerWithClients creates a new backlog handler with custom dependencies.
// If agentService implements AgentActivityChecker (e.g., *agentactivity.Service),
// it is also used for active-agent guards.
func NewHandlerWithClients(dataRoot, repoRoot string, agentService AgentSpawner, promptClient promptmanager.Client) *Handler {
	dataRoot = resolveDataRootOrDefault(dataRoot)
	repoRoot = resolveRepoRootOrDefault(repoRoot)
	h := &Handler{
		dataRoot:     dataRoot,
		repoRoot:     repoRoot,
		store:        NewFileStore(dataRoot),
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

// resolveDataRootOrDefault returns dataRoot if non-empty; otherwise resolves
// the runtime-home data path. Falls back to scenarioRoot on resolver error.
func resolveDataRootOrDefault(dataRoot string) string {
	if dataRoot != "" {
		return dataRoot
	}
	if p, err := runtimepaths.DataPath(""); err == nil {
		return p
	}
	return pathutil.ResolveScenarioRoot("swarm-manager")
}

// resolveRepoRootOrDefault returns repoRoot if non-empty; otherwise resolves
// the swarm-manager scenario source path.
func resolveRepoRootOrDefault(repoRoot string) string {
	if repoRoot != "" {
		return repoRoot
	}
	return pathutil.ResolveScenarioRoot("swarm-manager")
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

// SetAgentSessionArtifactRecorder wires durable session artifact attribution
// into backlog mutation chokepoints. Non-session requests are ignored by the
// recorder path because they carry no session_id in verified provenance.
func (h *Handler) SetAgentSessionArtifactRecorder(r sessionArtifactRecorder) {
	h.sessionArtifacts = r
}

// SetItemTerminalHandler wires a callback invoked after the review-decide
// endpoint flips an item to a terminal status. Passing nil clears the
// handler. The callback runs inside the request goroutine, so long-running
// work should self-dispatch.
//
// Prefer AddItemTerminalHandler when multiple subsystems need to observe
// terminal transitions (initiative review + records + future telemetry).
// SetItemTerminalHandler replaces all prior handlers, so chaining via Set
// silently overwrites earlier registrations.
func (h *Handler) SetItemTerminalHandler(f ItemTerminalHandler) {
	h.itemTerminalHandler = f
}

// SetRecordCreator wires the records capture seam. main.go installs this after
// both backlog and records services are constructed. Nil resets to no-op
// (review-decide will not capture a record).
func (h *Handler) SetRecordCreator(c RecordCreator) {
	h.recordCreator = c
}

// AddItemTerminalHandler appends a callback to the terminal-status chain. All
// registered handlers run in registration order; a panic in one does NOT
// prevent the next from running (each call is wrapped). Returns nil if f is
// nil (so callers can pass conditional handlers without guarding).
func (h *Handler) AddItemTerminalHandler(f ItemTerminalHandler) {
	if f == nil {
		return
	}
	prev := h.itemTerminalHandler
	h.itemTerminalHandler = func(ctx context.Context, kind, name string, status BacklogStatus) {
		if prev != nil {
			func() {
				defer func() { _ = recover() }()
				prev(ctx, kind, name, status)
			}()
		}
		func() {
			defer func() { _ = recover() }()
			f(ctx, kind, name, status)
		}()
	}
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
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/recover-review", h.RecoverReview).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/retry", h.Retry).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/research", h.Research).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/save", h.WorkshopSave).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/round", h.WorkshopDeleteRound).Methods("DELETE")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/workshop/reset", h.WorkshopReset).Methods("POST")
	r.HandleFunc("/api/v1/backlog/{kind}/{name}/re-workshop", h.ReWorkshop).Methods("POST")
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

	// Connect BacklogService — the typed cross-scenario feedback contract
	// (CreateItem/GetItem). See connect_service.go. Mounted alongside the REST
	// surface; the REST routes above remain for swarm-manager's own UI.
	registerBacklogConnectRoutes(r, h)
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
