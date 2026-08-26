// Package handlers provides HTTP request handlers for the autoheal API
// [REQ:CLI-TICK-001] [REQ:CLI-TICK-002] [REQ:CLI-STATUS-001] [REQ:CLI-STATUS-002]
// [REQ:FAIL-SAFE-001] [REQ:FAIL-OBSERVE-001] [REQ:WATCH-DETECT-001]
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reconcile"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/remediation"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/systemevents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/watchdog"

	apierrors "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/errors"
)

const (
	statusFreshnessThreshold = 3 * time.Minute
	healthDependencyTimeout  = 150 * time.Millisecond
	// healthDependencyConfirm is the second, longer probe a timed-out first
	// probe escalates to before the database is called unreachable.
	//
	// 150ms is a latency budget, not a liveness test, and conflating the two is
	// what turned a working retention cycle into an outage: every probe during
	// the cycle expired against a busy connection pool, /health reported the
	// database disconnected, and the supervisor restarted an API whose database
	// was fine. A probe that has to distinguish "slow" from "gone" must be
	// willing to wait longer than the fast path before it accuses anything.
	healthDependencyConfirm = 5 * time.Second
)

// StoreInterface defines the database operations needed by handlers
type StoreInterface interface {
	Ping(ctx context.Context) error
	SaveResult(ctx context.Context, result checks.Result) error
	GetRecentResults(ctx context.Context, checkID string, limit int) ([]checks.Result, error)
	GetTimelineEvents(ctx context.Context, limit int) ([]persistence.TimelineEvent, error)
	GetUptimeStats(ctx context.Context, windowHours int) (*persistence.UptimeStats, error)
	GetUptimeHistory(ctx context.Context, windowHours, bucketCount int) (*persistence.UptimeHistory, error)
	GetCheckTrends(ctx context.Context, windowHours int) (*persistence.CheckTrendsResponse, error)
	GetTransitions(ctx context.Context, windowHours, limit int) (*persistence.TransitionsResponse, error)
	UpsertSystemEvents(ctx context.Context, events []systemevents.Event) (int, int, error)
	UpsertSystemEventSource(ctx context.Context, source systemevents.SourceStatus) error
	ListSystemEvents(ctx context.Context, filters systemevents.Filters) (*systemevents.Response, error)
	GetSystemEventSources(ctx context.Context) ([]systemevents.SourceStatus, error)
	SaveHostInventorySnapshot(ctx context.Context, inv hostinventory.HostInventory) (*hostinventory.SnapshotRecord, []hostinventory.Change, error)
	GetLatestHostInventorySnapshot(ctx context.Context) (*hostinventory.SnapshotRecord, error)
	GetRecentHostInventoryChanges(ctx context.Context, limit int) ([]hostinventory.Change, error)
	UpsertIncident(ctx context.Context, input incidents.UpsertInput) (*incidents.Incident, error)
	ListIncidents(ctx context.Context, filters incidents.ListFilters) (*incidents.ListResponse, error)
	GetIncident(ctx context.Context, id string) (*incidents.Incident, error)
	ListIncidentObservations(ctx context.Context, incidentID string, limit int) ([]incidents.Observation, error)
	UpdateIncidentStatus(ctx context.Context, incidentID string, status incidents.Status, note string) (*incidents.Incident, error)
	RecordIncidentRemediationArtifact(ctx context.Context, incidentID string, artifact incidents.RemediationArtifact) (*incidents.Incident, error)
	RecordIncidentRemediationOutcome(ctx context.Context, incidentID string, outcome incidents.Outcome) (*incidents.Incident, error)
	// Action log operations [REQ:HEAL-ACTION-001]
	SaveActionLog(ctx context.Context, checkID, actionID string, success, timedOut bool, message, output, errMsg string, durationMs int64) error
	GetActionLogs(ctx context.Context, limit int) (*persistence.ActionLogsResponse, error)
	GetActionLogsForCheck(ctx context.Context, checkID string, limit int) (*persistence.ActionLogsResponse, error)
}

type operationalHistoryPruner interface {
	PruneOperationalHistory(ctx context.Context, before time.Time, batchSize int) (persistence.RetentionResult, error)
}

type operationalRetentionReporter interface {
	OperationalRetentionStatus(ctx context.Context) (persistence.RetentionStatus, error)
}

// Handlers wraps the dependencies needed by HTTP handlers
type Handlers struct {
	registry               *checks.Registry
	store                  StoreInterface
	platform               *platform.Capabilities
	watchdogDetector       *watchdog.Detector
	hostCollector          hostinventory.IntegrityCollector
	incidentService        *incidents.Service
	remediationService     *remediation.Service
	remediationAskVerifier remediation.AskVerifier
	systemEventService     *systemevents.Service
	historyRetentionHours  func() int
	lastRetentionAt        time.Time
	lastRetentionResult    persistence.RetentionResult
	lastRetentionErr       string
	reconcileProvider      reconcile.Provider
	installedProvider      reconcile.InstalledProvider
	startedAt              time.Time
	starter                string
	// skipLog keeps the action history to one row per auto-heal skip state
	// change rather than one per tick.
	skipLog *skipLogGate

	// tickLock prevents concurrent tick executions
	tickLock    sync.Mutex
	tickRunning bool
	tickStarted time.Time
	tickEnded   time.Time
}

func (h *Handlers) getTickState() (bool, *time.Time) {
	h.tickLock.Lock()
	defer h.tickLock.Unlock()

	if !h.tickRunning {
		return false, nil
	}

	if h.tickStarted.IsZero() {
		return true, nil
	}

	started := h.tickStarted.UTC()
	return true, &started
}

func (h *Handlers) getLastCompletedTick() *time.Time {
	h.tickLock.Lock()
	defer h.tickLock.Unlock()

	if h.tickEnded.IsZero() {
		return nil
	}

	completed := h.tickEnded.UTC()
	return &completed
}

// New creates a new Handlers instance
func New(registry *checks.Registry, store *persistence.Store, plat *platform.Capabilities) *Handlers {
	hostCollector := hostinventory.NewCachedIntegrityCollector(hostinventory.NewIntegrityCollector(hostinventory.IntegrityCollectorOptions{}), 30*time.Second)
	remediationService, _ := remediation.NewService()
	incidentService := incidents.NewService(store)
	registry.SetHealIncidentReporter(incidentService)
	return &Handlers{
		registry:           registry,
		store:              store,
		platform:           plat,
		watchdogDetector:   watchdog.NewDetector(plat),
		hostCollector:      hostCollector,
		incidentService:    incidentService,
		remediationService: remediationService,
		reconcileProvider:  reconcile.NewCoreSetProvider(),
		installedProvider:  reconcile.NewFilesystemInstalledProvider(),
		startedAt:          time.Now().UTC(),
		starter:            "unknown",
		skipLog:            newSkipLogGate(),
	}
}

func (h *Handlers) SetIncidentEventPublisher(publisher incidents.EventPublisher) {
	if h != nil && h.incidentService != nil {
		h.incidentService.SetEventPublisher(publisher)
	}
}

// SetRemediationAskVerifier wires the notification-hub read path. Keeping it
// behind an interface makes the execution gate testable without weakening the
// production rule that approval must come from the durable ask store.
func (h *Handlers) SetRemediationAskVerifier(verifier remediation.AskVerifier) {
	if h != nil {
		h.remediationAskVerifier = verifier
	}
}

// NewWithInterface creates a new Handlers instance with an interface-based store (for testing)
func NewWithInterface(registry *checks.Registry, store StoreInterface, plat *platform.Capabilities) *Handlers {
	hostCollector := hostinventory.NewCachedIntegrityCollector(hostinventory.NewIntegrityCollector(hostinventory.IntegrityCollectorOptions{}), 30*time.Second)
	remediationService, _ := remediation.NewService()
	incidentService := incidents.NewService(store)
	registry.SetHealIncidentReporter(incidentService)
	return &Handlers{
		registry:           registry,
		store:              store,
		platform:           plat,
		watchdogDetector:   watchdog.NewDetector(plat),
		hostCollector:      hostCollector,
		incidentService:    incidentService,
		remediationService: remediationService,
		reconcileProvider:  reconcile.NewCoreSetProvider(),
		installedProvider:  reconcile.NewFilesystemInstalledProvider(),
		startedAt:          time.Now().UTC(),
		starter:            "unknown",
		skipLog:            newSkipLogGate(),
	}
}

func (h *Handlers) SetReconcileProvider(provider reconcile.Provider) {
	if provider != nil {
		h.reconcileProvider = provider
	}
}

// SetInstalledProvider overrides the installed-target set used to tell a ghost
// check from a check that is merely out of supervision scope.
func (h *Handlers) SetInstalledProvider(provider reconcile.InstalledProvider) {
	if provider != nil {
		h.installedProvider = provider
	}
}

func (h *Handlers) SetSystemEventService(service *systemevents.Service) {
	h.systemEventService = service
}

// SetHistoryRetentionHoursProvider wires the live user configuration into the
// tick-owned retention pass. Keeping it as a narrow function avoids coupling
// HTTP handlers to the configuration manager implementation.
func (h *Handlers) SetHistoryRetentionHoursProvider(provider func() int) {
	h.historyRetentionHours = provider
}

// pruneOperationalHistory is the scenario's own pre-framework retention pass.
//
// It is NO LONGER CALLED FROM THE TICK, and that is deliberate. Until
// 2026-08-01 this ran on the tick path at the same time as the framework
// retention engine declared in .vrooli/service.json — two independent cleanup
// systems, on the same tables, through the same one-connection pool, each
// unaware of the other. The framework engine is the one with byte ceilings,
// batching, a wall-clock allowance, and findings; this one has an hour timer and
// an age cutoff, and its only remaining effect was to add write contention to
// the path the ceilings were already covering.
//
// It is kept, unwired, because it is still reachable from the operator retention
// command and remains the right tool there: an explicit, immediate, age-only
// prune with no schedule attached. Anything that wires it back into the tick is
// reintroducing the second system.
func (h *Handlers) pruneOperationalHistory(ctx context.Context) {
	if h.historyRetentionHours == nil || !h.lastRetentionAt.IsZero() && time.Since(h.lastRetentionAt) < time.Hour {
		return
	}
	pruner, ok := h.store.(operationalHistoryPruner)
	if !ok {
		return
	}
	hours := h.historyRetentionHours()
	if hours < 1 {
		return
	}
	result, err := pruner.PruneOperationalHistory(ctx, time.Now().Add(-time.Duration(hours)*time.Hour), 1000)
	if err != nil {
		h.lastRetentionErr = err.Error()
		apierrors.LogError("retention", "prune_operational_history", err)
		return
	}
	h.lastRetentionAt = time.Now()
	h.lastRetentionResult = result
	h.lastRetentionErr = ""
	if result.HealthResults+result.ActionLogs+result.Actions+result.SystemEvents > 0 {
		apierrors.LogInfo("retention", "pruned operational history", "health_results", result.HealthResults, "action_logs", result.ActionLogs, "actions", result.Actions, "system_events", result.SystemEvents)
	}
}

// Health returns basic service health for lifecycle checks.
//
// The dependencies map uses the structured DependencyStatus shape expected by
// the lifecycle registry's schema validator (api-core/health.Response). Emitting
// plain string values causes the registry to flag the scenario unhealthy with
// "cannot unmarshal string into Go struct field Response.dependencies".
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	busy, pingErr := h.pingStoreForHealth()
	dbDependency := map[string]interface{}{
		"connected": pingErr == nil,
		"database":  "sqlite",
	}
	switch {
	case pingErr != nil:
		status = "unhealthy"
		dbDependency["error"] = pingErr.Error()
	case busy:
		// Connected, answering, and slow. Readiness stays true: the service can
		// serve, and the one thing that must not happen is a supervisor reading
		// contention as death and restarting into the same contention.
		dbDependency["busy"] = true
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Vrooli Autoheal API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]interface{}{
			"database": dbDependency,
		},
		"summary": func() map[string]int {
			summary := h.registry.GetSummary()
			return map[string]int{"total": summary.TotalCount, "ok": summary.OkCount, "warning": summary.WarnCount, "critical": summary.CritCount, "notApplicable": summary.NotApplicableCount}
		}(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		apierrors.LogError("health", "encode_response", err)
	}
}

// pingStoreForHealth probes the database and reports whether it is reachable and
// whether it is merely busy.
//
// The two are different facts and the caller must not collapse them. A busy
// database is a healthy database under load — maintenance holding the write
// lock, a slow query ahead in the queue — and reporting it as a failed
// dependency invites whatever is watching to restart a process that has nothing
// wrong with it. That is not hypothetical: it is what happened on 2026-08-01,
// and the restart aborted the retention cycle that would have ended the load.
//
// So a timed-out fast probe escalates rather than concludes. Only a probe that
// fails outright, or that cannot complete even with seconds to work in, is
// reported as unreachable.
func (h *Handlers) pingStoreForHealth() (busy bool, err error) {
	if err := h.pingStoreWithin(healthDependencyTimeout); err == nil {
		return false, nil
	} else if !errors.Is(err, context.DeadlineExceeded) {
		// A real error from the driver is an answer, and a fast one. Only a
		// timeout is ambiguous enough to be worth a second probe.
		return false, err
	}

	if err := h.pingStoreWithin(healthDependencyConfirm); err != nil {
		return false, err
	}
	return true, nil
}

// pingStoreWithin runs one probe under its own deadline.
//
// The probe runs on its own goroutine because a driver that is blocked on a lock
// may not observe context cancellation promptly, and the health handler must
// return within its budget regardless of what the driver does.
func (h *Handlers) pingStoreWithin(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- h.store.Ping(ctx)
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Platform returns detected platform capabilities
func (h *Handlers) Platform(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(h.platform); err != nil {
		apierrors.LogError("platform", "encode_response", err)
	}
}

// Status returns the current health summary
func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	summary := h.registry.GetSummary()
	tickRunning, tickStartedAt := h.getTickState()
	lastCompletedTickAt := h.getLastCompletedTick()
	statusFresh, statusAgeSeconds, staleReason := evaluateStatusFreshness(lastCompletedTickAt, time.Now())

	response := map[string]interface{}{
		"status":                          summary.Status,
		"platform":                        h.platform,
		"tickRunning":                     tickRunning,
		"tickStartedAt":                   tickStartedAt,
		"lastCompletedTickAt":             lastCompletedTickAt,
		"statusFresh":                     statusFresh,
		"statusAgeSeconds":                statusAgeSeconds,
		"statusFreshnessThresholdSeconds": int(statusFreshnessThreshold.Seconds()),
		"statusStaleReason":               staleReason,
		"summary": map[string]interface{}{
			"total":         summary.TotalCount,
			"ok":            summary.OkCount,
			"warning":       summary.WarnCount,
			"critical":      summary.CritCount,
			"notApplicable": summary.NotApplicableCount,
		},
		"checks":    summary.Checks,
		"timestamp": summary.Timestamp,
	}
	if h.systemEventService != nil {
		response["systemEvents"] = map[string]interface{}{
			"journalctlExecsAvoided": h.systemEventService.ExecsAvoided(),
		}
	}
	// A skipped heal is an operational outcome, not an absence of data. Keep
	// the reason in the same durable action history as executed heals and expose
	// the recent entries on status so operators can distinguish policy,
	// cooldown, and unavailable-action decisions.
	if logs, err := h.store.GetActionLogs(r.Context(), 100); err == nil {
		skips := make([]persistence.ActionLog, 0)
		issues := make(map[string]persistence.ActionLog)
		latestOutcomeSeen := make(map[string]bool)
		for _, log := range logs.Logs {
			if log.ActionID == "autoheal-skip" {
				skips = append(skips, log)
			}
			// Action history is newest-first. The latest outcome, whether
			// successful or not, is authoritative for the current card. Do not
			// resurrect an older failure after a later recovery succeeded.
			if latestOutcomeSeen[log.CheckID] {
				continue
			}
			latestOutcomeSeen[log.CheckID] = true
			if !log.Success {
				issues[log.CheckID] = log
			}
		}
		response["autoHealSkips"] = skips
		response["autoHealIssues"] = issues
	} else {
		response["autoHealSkips"] = []persistence.ActionLog{}
		response["autoHealIssues"] = map[string]persistence.ActionLog{}
	}
	if reporter, ok := h.store.(operationalRetentionReporter); ok {
		ctx, cancel := context.WithTimeout(r.Context(), healthDependencyTimeout)
		retention, err := reporter.OperationalRetentionStatus(ctx)
		cancel()
		if err != nil {
			response["retention"] = map[string]interface{}{"degraded": true, "error": err.Error()}
		} else {
			response["retention"] = map[string]interface{}{
				"database":          retention,
				"lastPrunedAt":      h.lastRetentionAt,
				"lastPrune":         h.lastRetentionResult,
				"degraded":          h.lastRetentionErr != "",
				"degradationReason": h.lastRetentionErr,
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		apierrors.LogError("status", "encode_response", err)
	}
}

// Tick runs a single health check cycle
// Uses a lock to prevent concurrent executions - if a tick is already running,
// returns immediately with a 409 Conflict status.
func (h *Handlers) Tick(w http.ResponseWriter, r *http.Request) {
	const maxTickRuntime = 6 * time.Minute
	compactResponse := r.URL.Query().Get("compact") == "true"

	// Try to acquire the tick lock
	h.tickLock.Lock()
	if h.tickRunning {
		// Safety valve: if a tick appears stuck far beyond normal runtime,
		// reset the lock to restore service availability.
		if !h.tickStarted.IsZero() && time.Since(h.tickStarted) > maxTickRuntime {
			apierrors.LogInfo("tick", "resetting stale tick lock", "age", time.Since(h.tickStarted).String())
			h.tickRunning = false
			h.tickStarted = time.Time{}
		}
	}
	if h.tickRunning {
		h.tickLock.Unlock()
		apierrors.LogAndRespond(w, apierrors.NewConflictError("tick",
			"A health check cycle is already running. Please wait for it to complete."))
		return
	}
	h.tickRunning = true
	h.tickStarted = time.Now()
	h.tickLock.Unlock()

	// Ensure we release the lock when done
	defer func() {
		h.tickLock.Lock()
		h.tickRunning = false
		h.tickStarted = time.Time{}
		h.tickEnded = time.Now()
		h.tickLock.Unlock()
	}()

	// Parse force parameter
	forceAll := r.URL.Query().Get("force") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	results := h.registry.RunAll(ctx, forceAll)

	inventoryChanges := h.collectAndPersistHostInventory(ctx)

	// Store results in database - log failures but don't block the response
	// [REQ:FAIL-SAFE-001] Tick completes even if persistence fails
	var persistenceErrors int
	for _, result := range results {
		if err := h.store.SaveResult(ctx, result); err != nil {
			persistenceErrors++
			apierrors.LogError("tick", "save_result:"+result.CheckID, err)
		}
	}
	if h.incidentService != nil {
		if _, err := h.incidentService.UpsertFromCheckResults(ctx, results); err != nil {
			apierrors.LogError("tick", "upsert_incidents", err)
		}
	}
	h.upsertIncidentsFromInventoryChanges(ctx, inventoryChanges)
	h.refreshSystemEvents(ctx)

	// Run auto-heal for critical checks with auto-heal enabled
	// [REQ:CONFIG-CHECK-001] [REQ:HEAL-ACTION-001]
	autoHealResults := h.registry.RunAutoHeal(ctx, results)

	// Log auto-heal actions to database
	for _, ahr := range autoHealResults {
		if ahr.Attempted {
			if err := h.store.SaveActionLog(
				ctx,
				ahr.ActionResult.CheckID,
				ahr.ActionResult.ActionID,
				ahr.ActionResult.Success,
				ahr.ActionResult.TimedOut,
				"[auto-heal] "+ahr.ActionResult.Message,
				ahr.ActionResult.Output,
				ahr.ActionResult.Error,
				ahr.ActionResult.Duration.Milliseconds(),
			); err != nil {
				apierrors.LogError("tick", "save_autoheal_log:"+ahr.CheckID, err)
			}
		} else if h.skipLog.ShouldLog(ahr.CheckID, ahr.Reason) {
			// One row per state change, not one per tick. The first skip and
			// every skip that says something new are recorded; the repeats that
			// say nothing new are not.
			if err := h.store.SaveActionLog(
				ctx,
				ahr.CheckID,
				"autoheal-skip",
				false,
				false,
				"[auto-heal skipped] "+ahr.Reason,
				"",
				ahr.Reason,
				0,
			); err != nil {
				apierrors.LogError("tick", "save_autoheal_skip:"+ahr.CheckID, err)
			}
		}
		if ahr.Attempted {
			// An attempt ends the skip state, so the next skip is a fresh
			// state change worth recording.
			h.skipLog.Clear(ahr.CheckID)
		}
	}

	// Re-check any items where autoheal was attempted to update their status
	// This is critical: autoheal may have fixed issues but check results were from BEFORE the fix
	// We re-check even if autoheal reports failure, because the action might have actually
	// started something that just wasn't ready during verification polling
	var recheckIDs []string
	for _, ahr := range autoHealResults {
		if ahr.Attempted {
			recheckIDs = append(recheckIDs, ahr.CheckID)
		}
	}
	if len(recheckIDs) > 0 {
		recheckResults := h.registry.RunChecksForIDs(ctx, recheckIDs)
		for _, result := range recheckResults {
			// Update results array with new result
			for i, r := range results {
				if r.CheckID == result.CheckID {
					results[i] = result
					break
				}
			}
			// Persist updated result
			if err := h.store.SaveResult(ctx, result); err != nil {
				apierrors.LogError("tick", "save_recheck_result:"+result.CheckID, err)
			}
			if h.incidentService != nil {
				if _, _, err := h.incidentService.UpsertFromCheckResult(ctx, result); err != nil {
					apierrors.LogError("tick", "upsert_recheck_incident:"+result.CheckID, err)
				}
			}
		}
	}

	// Get updated summary
	summary := h.registry.GetSummary()

	response := map[string]interface{}{
		"success": true,
		"status":  summary.Status,
		"summary": map[string]interface{}{
			"total":         summary.TotalCount,
			"ok":            summary.OkCount,
			"warning":       summary.WarnCount,
			"critical":      summary.CritCount,
			"notApplicable": summary.NotApplicableCount,
		},
		"timestamp": time.Now().UTC(),
	}
	if len(inventoryChanges) > 0 {
		response["hostInventoryChanges"] = inventoryChanges
	}
	if !compactResponse {
		response["results"] = results
		response["autoHeal"] = autoHealResults
	}

	// Include warning about persistence issues without failing the request
	if persistenceErrors > 0 {
		response["warnings"] = []string{
			"Some results could not be persisted to database",
		}
		apierrors.LogInfo("tick", "completed with persistence errors", persistenceErrors)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		apierrors.LogError("tick", "encode_response", err)
	}
}

// refreshSystemEvents runs system-event ingestion on the tick path, but gated
// behind the service's own (coarser) interval so the expensive journalctl
// kernel-grep does not run on every 60s tick. The explicit refresh endpoint
// still forces an immediate ingest.
func (h *Handlers) refreshSystemEvents(ctx context.Context) {
	if h.systemEventService == nil {
		return
	}
	if _, _, err := h.systemEventService.IngestIfDue(ctx); err != nil {
		apierrors.LogError("system-events", "ingest", err)
	}
}

func (h *Handlers) collectAndPersistHostInventory(ctx context.Context) []hostinventory.Change {
	if h.hostCollector == nil || h.store == nil {
		return nil
	}
	inv, err := h.hostCollector.Collect(ctx)
	if err != nil {
		apierrors.LogError("tick", "collect_host_inventory", err)
		return nil
	}
	_, changes, err := h.store.SaveHostInventorySnapshot(ctx, inv)
	if err != nil {
		apierrors.LogError("tick", "save_host_inventory", err)
		return nil
	}
	return changes
}

func (h *Handlers) upsertIncidentsFromInventoryChanges(ctx context.Context, changes []hostinventory.Change) {
	if len(changes) == 0 || h.store == nil {
		return
	}
	for _, change := range changes {
		if change.Severity != "warning" && change.Severity != "critical" {
			continue
		}
		severity := incidents.SeverityWarning
		if change.Severity == "critical" {
			severity = incidents.SeverityCritical
		}
		_, err := h.store.UpsertIncident(ctx, incidents.UpsertInput{
			Fingerprint:   incidents.Fingerprint(string(incidents.TypeHostIntegrity), "host-inventory-change", change.ChangeType, change.ToSnapshotID),
			Type:          incidents.TypeHostIntegrity,
			Severity:      severity,
			Title:         "Host inventory changed",
			Summary:       change.Summary,
			ObservedAt:    change.CreatedAt,
			SourceCheckID: "host-capability-drift",
			Evidence: map[string]any{
				"changeType":     change.ChangeType,
				"fromSnapshotId": change.FromSnapshotID,
				"toSnapshotId":   change.ToSnapshotID,
				"details":        change.Details,
			},
			Recommendations: []string{
				"Compare the previous and current host inventory snapshots before assuming workload-level causes.",
				"Review kernel, module, runtime, and device binding changes around the incident time.",
			},
		})
		if err != nil {
			apierrors.LogError("tick", "upsert_inventory_change_incident", err)
		}
	}
}

func evaluateStatusFreshness(lastCompletedTickAt *time.Time, now time.Time) (bool, int64, string) {
	if lastCompletedTickAt == nil || lastCompletedTickAt.IsZero() {
		return false, 0, "no completed tick recorded"
	}

	age := now.Sub(*lastCompletedTickAt)
	if age < 0 {
		age = 0
	}
	ageSeconds := int64(age / time.Second)

	if age > statusFreshnessThreshold {
		return false, ageSeconds, fmt.Sprintf("last completed tick is older than %s", statusFreshnessThreshold)
	}

	return true, ageSeconds, ""
}

// ListChecks returns all registered checks
