// Package handlers provides HTTP request handlers for the autoheal API
// [REQ:CLI-TICK-001] [REQ:CLI-TICK-002] [REQ:CLI-STATUS-001] [REQ:CLI-STATUS-002]
// [REQ:FAIL-SAFE-001] [REQ:FAIL-OBSERVE-001] [REQ:WATCH-DETECT-001]
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/incidents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/remediation"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/systemevents"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/watchdog"

	"github.com/gorilla/mux"
	apierrors "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/errors"
)

const (
	statusFreshnessThreshold = 3 * time.Minute
	healthDependencyTimeout  = 150 * time.Millisecond
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
	CleanupOldSystemEvents(ctx context.Context, before time.Time) (int64, error)
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

// Handlers wraps the dependencies needed by HTTP handlers
type Handlers struct {
	registry           *checks.Registry
	store              StoreInterface
	platform           *platform.Capabilities
	watchdogDetector   *watchdog.Detector
	hostCollector      hostinventory.Collector
	incidentService    *incidents.Service
	remediationService *remediation.Service
	systemEventService *systemevents.Service

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
	hostCollector := hostinventory.NewCachedCollector(hostinventory.NewCollector(hostinventory.CollectorOptions{Platform: plat}), 30*time.Second)
	remediationService, _ := remediation.NewService()
	return &Handlers{
		registry:           registry,
		store:              store,
		platform:           plat,
		watchdogDetector:   watchdog.NewDetector(plat),
		hostCollector:      hostCollector,
		incidentService:    incidents.NewService(store),
		remediationService: remediationService,
	}
}

// NewWithInterface creates a new Handlers instance with an interface-based store (for testing)
func NewWithInterface(registry *checks.Registry, store StoreInterface, plat *platform.Capabilities) *Handlers {
	hostCollector := hostinventory.NewCachedCollector(hostinventory.NewCollector(hostinventory.CollectorOptions{Platform: plat}), 30*time.Second)
	remediationService, _ := remediation.NewService()
	return &Handlers{
		registry:           registry,
		store:              store,
		platform:           plat,
		watchdogDetector:   watchdog.NewDetector(plat),
		hostCollector:      hostCollector,
		incidentService:    incidents.NewService(store),
		remediationService: remediationService,
	}
}

func (h *Handlers) SetSystemEventService(service *systemevents.Service) {
	h.systemEventService = service
}

// Health returns basic service health for lifecycle checks.
//
// The dependencies map uses the structured DependencyStatus shape expected by
// the lifecycle registry's schema validator (api-core/health.Response). Emitting
// plain string values causes the registry to flag the scenario unhealthy with
// "cannot unmarshal string into Go struct field Response.dependencies".
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	pingErr := h.pingStoreForHealth()
	dbDependency := map[string]interface{}{
		"connected": pingErr == nil,
		"database":  "sqlite",
	}
	if pingErr != nil {
		status = "unhealthy"
		dbDependency["error"] = pingErr.Error()
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
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		apierrors.LogError("health", "encode_response", err)
	}
}

func (h *Handlers) pingStoreForHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), healthDependencyTimeout)
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
			"total":    summary.TotalCount,
			"ok":       summary.OkCount,
			"warning":  summary.WarnCount,
			"critical": summary.CritCount,
		},
		"checks":    summary.Checks,
		"timestamp": summary.Timestamp,
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
			"total":    summary.TotalCount,
			"ok":       summary.OkCount,
			"warning":  summary.WarnCount,
			"critical": summary.CritCount,
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

func (h *Handlers) refreshSystemEvents(ctx context.Context) {
	if h.systemEventService == nil {
		return
	}
	if _, err := h.systemEventService.Ingest(ctx); err != nil {
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
func (h *Handlers) ListChecks(w http.ResponseWriter, r *http.Request) {
	checks := h.registry.ListChecks()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(checks); err != nil {
		apierrors.LogError("list_checks", "encode_response", err)
	}
}

// CheckResult returns the result for a specific check
func (h *Handlers) CheckResult(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkID := vars["checkId"]

	result, exists := h.registry.GetResult(checkID)
	if !exists {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("checks", "check result", checkID))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		apierrors.LogError("check_result", "encode_response", err)
	}
}

// CheckHistory returns historical results for a specific check
// [REQ:PERSIST-QUERY-001] [REQ:PERSIST-QUERY-002]
func (h *Handlers) CheckHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkID := vars["checkId"]

	// Default limit to 20 entries
	limit := 20

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	results, err := h.store.GetRecentResults(ctx, checkID, limit)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("history", "retrieve check history", err))
		return
	}

	// Return empty array instead of null when no results (safe default)
	if results == nil {
		results = []checks.Result{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"checkId": checkID,
		"history": results,
		"count":   len(results),
	}); err != nil {
		apierrors.LogError("history", "encode_response", err)
	}
}

// Timeline returns recent events across all checks
// [REQ:UI-EVENTS-001]
func (h *Handlers) Timeline(w http.ResponseWriter, r *http.Request) {
	// Default limit to 50 events
	limit := 50

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	events, err := h.store.GetTimelineEvents(ctx, limit)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("timeline", "retrieve events", err))
		return
	}

	// Return empty array instead of null when no events (safe default)
	if events == nil {
		events = []persistence.TimelineEvent{}
	}

	// Group events by status for summary
	summary := map[string]int{"ok": 0, "warning": 0, "critical": 0}
	for _, e := range events {
		summary[e.Status]++
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"events":  events,
		"count":   len(events),
		"summary": summary,
	}); err != nil {
		apierrors.LogError("timeline", "encode_response", err)
	}
}

func (h *Handlers) SystemEvents(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseSystemEventFilters(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	response, err := h.store.ListSystemEvents(ctx, filters)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("system-events", "retrieve system events", err))
		return
	}
	if response == nil {
		response = &systemevents.Response{Events: []systemevents.Event{}, Sources: []systemevents.SourceStatus{}}
	}
	if response.Events == nil {
		response.Events = []systemevents.Event{}
	}
	if response.Sources == nil {
		response.Sources = []systemevents.SourceStatus{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		apierrors.LogError("system-events", "encode_response", err)
	}
}

func (h *Handlers) RefreshSystemEvents(w http.ResponseWriter, r *http.Request) {
	if h.systemEventService == nil {
		apierrors.LogAndRespond(w, apierrors.NewServiceUnavailableError("system-events", "system event service", fmt.Errorf("service unavailable")))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	summary, err := h.systemEventService.Ingest(ctx)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewServiceUnavailableError("system-events", "system event service", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(summary); err != nil {
		apierrors.LogError("system-events", "encode_refresh_response", err)
	}
}

func parseSystemEventFilters(w http.ResponseWriter, r *http.Request) (systemevents.Filters, bool) {
	q := r.URL.Query()
	filters := systemevents.Filters{Limit: 100, Correlate: q.Get("correlate") == "true"}
	if raw := q.Get("limit"); raw != "" {
		limit, err := parsePositiveInt(raw)
		if err != nil {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("system-events", "invalid limit", err))
			return filters, false
		}
		filters.Limit = limit
	}
	if filters.Limit <= 0 {
		filters.Limit = 100
	}
	if filters.Limit > 500 {
		filters.Limit = 500
	}
	if since, err := parseTimeParam(q.Get("since")); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("system-events", "invalid since", err))
		return filters, false
	} else if since != nil {
		filters.Since = since
	}
	if until, err := parseTimeParam(q.Get("until")); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("system-events", "invalid until", err))
		return filters, false
	} else if until != nil {
		filters.Until = until
	}
	filters.Category = splitCSV(q.Get("category"))
	filters.Source = splitCSV(q.Get("source"))
	filters.Platform = splitCSV(q.Get("platform"))
	for _, raw := range splitCSV(q.Get("severity")) {
		switch raw {
		case "info":
			filters.Severity = append(filters.Severity, systemevents.SeverityInfo)
		case "warning":
			filters.Severity = append(filters.Severity, systemevents.SeverityWarning)
		case "critical":
			filters.Severity = append(filters.Severity, systemevents.SeverityCritical)
		default:
			apierrors.LogAndRespond(w, apierrors.NewValidationError("system-events", "invalid severity", fmt.Errorf("invalid severity %q", raw)))
			return filters, false
		}
	}
	filters.BootID = q.Get("bootId")
	return filters, true
}

func parseTimeParam(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if duration, err := time.ParseDuration(raw); err == nil {
		ts := time.Now().UTC().Add(-duration)
		return &ts, nil
	}
	if strings.HasSuffix(raw, "d") {
		days, err := parsePositiveInt(strings.TrimSuffix(raw, "d"))
		if err == nil {
			ts := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
			return &ts, nil
		}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if ts, err := time.Parse(layout, raw); err == nil {
			utc := ts.UTC()
			return &utc, nil
		}
	}
	return nil, fmt.Errorf("expected RFC3339 timestamp or Go duration")
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// UptimeStats returns uptime statistics over a time window
// [REQ:PERSIST-HISTORY-001]
func (h *Handlers) UptimeStats(w http.ResponseWriter, r *http.Request) {
	// Default to 24 hours
	windowHours := 24

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	stats, err := h.store.GetUptimeStats(ctx, windowHours)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("uptime", "calculate uptime statistics", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		apierrors.LogError("uptime", "encode_response", err)
	}
}

// UptimeHistory returns time-bucketed uptime data for charting
// [REQ:PERSIST-HISTORY-001] [REQ:UI-EVENTS-001]
func (h *Handlers) UptimeHistory(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters with defaults
	windowHours := 24
	bucketCount := 24

	if hoursStr := r.URL.Query().Get("hours"); hoursStr != "" {
		if parsed, err := parsePositiveInt(hoursStr); err == nil && parsed > 0 && parsed <= 168 {
			windowHours = parsed
		}
	}

	if bucketsStr := r.URL.Query().Get("buckets"); bucketsStr != "" {
		if parsed, err := parsePositiveInt(bucketsStr); err == nil && parsed > 0 && parsed <= 100 {
			bucketCount = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	history, err := h.store.GetUptimeHistory(ctx, windowHours, bucketCount)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("uptime_history", "retrieve uptime history", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(history); err != nil {
		apierrors.LogError("uptime_history", "encode_response", err)
	}
}

// parsePositiveInt parses a string to a positive integer
func parsePositiveInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// CheckTrends returns per-check trend data
// [REQ:PERSIST-HISTORY-001]
func (h *Handlers) CheckTrends(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters with defaults
	windowHours := 24

	if hoursStr := r.URL.Query().Get("hours"); hoursStr != "" {
		if parsed, err := parsePositiveInt(hoursStr); err == nil && parsed > 0 && parsed <= 168 {
			windowHours = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	trends, err := h.store.GetCheckTrends(ctx, windowHours)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("check_trends", "retrieve check trends", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(trends); err != nil {
		apierrors.LogError("check_trends", "encode_response", err)
	}
}

// Transitions returns status transition events.
// [REQ:PERSIST-HISTORY-001]
func (h *Handlers) Transitions(w http.ResponseWriter, r *http.Request) {
	windowHours := 24
	limit := 50

	if hoursStr := r.URL.Query().Get("hours"); hoursStr != "" {
		if parsed, err := parsePositiveInt(hoursStr); err == nil && parsed > 0 && parsed <= 168 {
			windowHours = parsed
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := parsePositiveInt(limitStr); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	transitions, err := h.store.GetTransitions(ctx, windowHours, limit)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("transitions", "retrieve transitions", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(transitions); err != nil {
		apierrors.LogError("transitions", "encode_response", err)
	}
}

func (h *Handlers) HostInventory(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	record, err := h.store.GetLatestHostInventorySnapshot(ctx)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("host_inventory", "retrieve host inventory", err))
		return
	}
	resp := hostinventory.InventoryResponse{Snapshot: record, Fresh: false, ProbeStatus: map[string]hostinventory.ProbeState{}}
	if record != nil {
		resp.AgeSeconds = int64(time.Since(record.CollectedAt) / time.Second)
		resp.Fresh = resp.AgeSeconds <= 300
		resp.ProbeStatus = record.Inventory.ProbeStatus
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		apierrors.LogError("host_inventory", "encode_response", err)
	}
}

func (h *Handlers) CollectHostInventory(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	inv, err := h.hostCollector.Collect(ctx)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("host_inventory", "collect host inventory", err))
		return
	}
	record, changes, err := h.store.SaveHostInventorySnapshot(ctx, inv)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("host_inventory", "save host inventory", err))
		return
	}
	if changes == nil {
		changes = []hostinventory.Change{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"snapshot": record, "changes": changes}); err != nil {
		apierrors.LogError("host_inventory", "encode_collect_response", err)
	}
}

func (h *Handlers) HostInventoryChanges(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsed, err := parsePositiveInt(limitStr); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	changes, err := h.store.GetRecentHostInventoryChanges(ctx, limit)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("host_inventory", "retrieve host inventory changes", err))
		return
	}
	if changes == nil {
		changes = []hostinventory.Change{}
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"changes": changes, "total": len(changes)}); err != nil {
		apierrors.LogError("host_inventory", "encode_changes_response", err)
	}
}

func (h *Handlers) Incidents(w http.ResponseWriter, r *http.Request) {
	filters, ok := parseIncidentFilters(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	resp, err := h.store.ListIncidents(ctx, filters)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incidents", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		apierrors.LogError("incidents", "encode_response", err)
	}
}

func (h *Handlers) LatestIncidents(w http.ResponseWriter, r *http.Request) {
	r.URL.RawQuery = mergeDefaultQuery(r.URL.RawQuery, "status=open")
	h.Incidents(w, r)
}

func (h *Handlers) IncidentDetail(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	incident, err := h.store.GetIncident(ctx, id)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incident", err))
		return
	}
	if incident == nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("incidents", "incident", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(incident); err != nil {
		apierrors.LogError("incidents", "encode_detail_response", err)
	}
}

func (h *Handlers) IncidentObservations(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	limit := 50
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	observations, err := h.store.ListIncidentObservations(ctx, id, limit)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incident observations", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"observations": observations, "total": len(observations)}); err != nil {
		apierrors.LogError("incidents", "encode_observations_response", err)
	}
}

func (h *Handlers) IncidentRemediations(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	incident, err := h.store.GetIncident(ctx, id)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incident", err))
		return
	}
	if incident == nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("incidents", "incident", id))
		return
	}
	candidates := incident.RemediationCandidates
	if h.remediationService != nil {
		candidates = h.remediationService.Candidates(*incident)
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"incidentId": id, "remediations": candidates, "total": len(candidates)}); err != nil {
		apierrors.LogError("incidents", "encode_remediations_response", err)
	}
}

func (h *Handlers) GenerateIncidentRemediation(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	remediationID := mux.Vars(r)["remediationId"]
	if h.remediationService == nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "remediation service unavailable", fmt.Errorf("remediation service unavailable")))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	incident, err := h.store.GetIncident(ctx, id)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incident", err))
		return
	}
	if incident == nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("incidents", "incident", id))
		return
	}
	resp, err := h.remediationService.Generate(*incident, remediationID)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "generate remediation", err))
		return
	}
	if _, err := h.store.RecordIncidentRemediationArtifact(ctx, id, resp.Artifact); err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "record remediation artifact", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		apierrors.LogError("incidents", "encode_remediation_generation_response", err)
	}
}

func (h *Handlers) RecordIncidentRemediationOutcome(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	remediationID := mux.Vars(r)["remediationId"]
	var req remediation.OutcomeRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "decode remediation outcome", err))
			return
		}
	}
	if h.remediationService == nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "remediation service unavailable", fmt.Errorf("remediation service unavailable")))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	incident, err := h.store.GetIncident(ctx, id)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incident", err))
		return
	}
	if incident == nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("incidents", "incident", id))
		return
	}
	outcome, err := h.remediationService.Outcome(*incident, remediationID, req)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "record remediation outcome", err))
		return
	}
	updated, err := h.store.RecordIncidentRemediationOutcome(ctx, id, outcome)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "record remediation outcome", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(updated); err != nil {
		apierrors.LogError("incidents", "encode_remediation_outcome_response", err)
	}
}

func (h *Handlers) MutateIncidentStatus(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["incidentId"]
	action := mux.Vars(r)["action"]
	var status incidents.Status
	switch action {
	case "acknowledge":
		status = incidents.StatusAcknowledged
	case "resolve":
		status = incidents.StatusResolved
	case "ignore":
		status = incidents.StatusIgnored
	default:
		apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "invalid incident status action", fmt.Errorf("unsupported action %q", action)))
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	incident, err := h.store.UpdateIncidentStatus(ctx, id, status, body.Note)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "update incident status", err))
		return
	}
	if incident == nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("incidents", "incident", id))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(incident); err != nil {
		apierrors.LogError("incidents", "encode_mutation_response", err)
	}
}

func parseIncidentFilters(w http.ResponseWriter, r *http.Request) (incidents.ListFilters, bool) {
	query := r.URL.Query()
	filters := incidents.ListFilters{Limit: 50}
	if status := query.Get("status"); status != "" {
		if !incidents.ValidStatus(status) {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "invalid status filter", fmt.Errorf("invalid status %q", status)))
			return filters, false
		}
		filters.Status = incidents.Status(status)
	}
	if severity := query.Get("severity"); severity != "" {
		if !incidents.ValidSeverity(severity) {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "invalid severity filter", fmt.Errorf("invalid severity %q", severity)))
			return filters, false
		}
		filters.Severity = incidents.Severity(severity)
	}
	if typ := query.Get("type"); typ != "" {
		if !incidents.ValidType(typ) {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("incidents", "invalid type filter", fmt.Errorf("invalid type %q", typ)))
			return filters, false
		}
		filters.Type = incidents.Type(typ)
	}
	if limitStr := query.Get("limit"); limitStr != "" {
		if parsed, err := parsePositiveInt(limitStr); err == nil && parsed > 0 && parsed <= 200 {
			filters.Limit = parsed
		}
	}
	return filters, true
}

func mergeDefaultQuery(rawQuery, defaults string) string {
	if rawQuery == "" {
		return defaults
	}
	return rawQuery + "&" + defaults
}

// Watchdog returns the OS-level watchdog/service status
// [REQ:WATCH-DETECT-001]
func (h *Handlers) Watchdog(w http.ResponseWriter, r *http.Request) {
	// Check if refresh is requested
	refresh := r.URL.Query().Get("refresh") == "true"

	var status *watchdog.Status
	if refresh {
		status = h.watchdogDetector.Detect()
	} else {
		status = h.watchdogDetector.GetCached()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		apierrors.LogError("watchdog", "encode_response", err)
	}
}

// WatchdogTemplate returns the service configuration template for the current platform
// [REQ:WATCH-LINUX-001] [REQ:WATCH-MAC-001] [REQ:WATCH-WIN-001]
func (h *Handlers) WatchdogTemplate(w http.ResponseWriter, r *http.Request) {
	template, err := h.watchdogDetector.GetServiceTemplate()
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("watchdog", "service template", string(h.platform.Platform)))
		return
	}

	// Build API base URL from request
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	apiBaseURL := fmt.Sprintf("%s://%s", scheme, r.Host)

	platformStr := string(h.platform.Platform)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"platform":     h.platform.Platform,
		"template":     template,
		"instructions": getInstallInstructions(platformStr),
		"oneLiner":     getOneLinerInstall(platformStr, apiBaseURL),
	}); err != nil {
		apierrors.LogError("watchdog_template", "encode_response", err)
	}
}

// getInstallInstructions returns platform-specific installation instructions
func getInstallInstructions(platformStr string) string {
	switch platformStr {
	case "linux":
		return `1. Save the template to /etc/systemd/system/vrooli-autoheal.service
2. Run: sudo systemctl daemon-reload
3. Run: sudo systemctl enable vrooli-autoheal
4. Run: sudo systemctl start vrooli-autoheal`
	case "macos":
		return `1. Save the template to ~/Library/LaunchAgents/com.vrooli.autoheal.plist
2. Run: launchctl load ~/Library/LaunchAgents/com.vrooli.autoheal.plist`
	case "windows":
		return `1. Save the template as VrooliAutoheal.xml
2. Run as Administrator: schtasks /Create /TN VrooliAutoheal /XML VrooliAutoheal.xml`
	default:
		return "Watchdog installation not supported on this platform"
	}
}

// getOneLinerInstall returns a one-liner command to install the watchdog service
func getOneLinerInstall(platformStr, apiBaseURL string) string {
	switch platformStr {
	case "linux":
		return fmt.Sprintf(`curl -s %s/api/v1/watchdog/template | jq -r '.template' | sudo tee /etc/systemd/system/vrooli-autoheal.service > /dev/null && sudo systemctl daemon-reload && sudo systemctl enable --now vrooli-autoheal`, apiBaseURL)
	case "macos":
		return fmt.Sprintf(`curl -s %s/api/v1/watchdog/template | jq -r '.template' > ~/Library/LaunchAgents/com.vrooli.autoheal.plist && launchctl load ~/Library/LaunchAgents/com.vrooli.autoheal.plist`, apiBaseURL)
	case "windows":
		return fmt.Sprintf(`(Invoke-WebRequest -Uri %s/api/v1/watchdog/template).Content | ConvertFrom-Json | Select-Object -ExpandProperty template | Out-File VrooliAutoheal.xml; schtasks /Create /TN VrooliAutoheal /XML VrooliAutoheal.xml`, apiBaseURL)
	default:
		return ""
	}
}

// WatchdogInstall handles installation of the OS watchdog service
// [REQ:WATCH-INSTALL-001]
func (h *Handlers) WatchdogInstall(w http.ResponseWriter, r *http.Request) {
	// Parse installation options from request body
	var opts watchdog.InstallOptions
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
			apierrors.LogAndRespond(w, apierrors.NewValidationError("watchdog", "invalid request body", err))
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	result := h.watchdogDetector.Install(ctx, opts)

	w.Header().Set("Content-Type", "application/json")
	if !result.Success {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		apierrors.LogError("watchdog_install", "encode_response", err)
	}
}

// WatchdogUninstall handles removal of the OS watchdog service
func (h *Handlers) WatchdogUninstall(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Minute)
	defer cancel()

	result := h.watchdogDetector.Uninstall(ctx)

	w.Header().Set("Content-Type", "application/json")
	if !result.Success {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		apierrors.LogError("watchdog_uninstall", "encode_response", err)
	}
}

// WatchdogEnableLinger enables systemd lingering for user services (Linux only)
func (h *Handlers) WatchdogEnableLinger(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result := h.watchdogDetector.EnableLingering(ctx)

	w.Header().Set("Content-Type", "application/json")
	if !result.Success {
		w.WriteHeader(http.StatusInternalServerError)
	}
	if err := json.NewEncoder(w).Encode(result); err != nil {
		apierrors.LogError("watchdog_linger", "encode_response", err)
	}
}

// WatchdogStatus returns detailed installation status
func (h *Handlers) WatchdogStatus(w http.ResponseWriter, r *http.Request) {
	status := h.watchdogDetector.GetInstallStatus()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		apierrors.LogError("watchdog_status", "encode_response", err)
	}
}

// GetCheckActions returns available recovery actions for a check
// [REQ:HEAL-ACTION-001]
func (h *Handlers) GetCheckActions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkID := vars["checkId"]

	// Get the check and verify it's healable
	healable, ok := h.registry.GetHealableCheck(checkID)
	if !ok {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("actions", "healable check", checkID))
		return
	}

	// Get last result to determine available actions
	lastResult, _ := h.registry.GetResult(checkID)
	var lastResultPtr *checks.Result
	if lastResult.CheckID != "" {
		lastResultPtr = &lastResult
	}

	actions := healable.RecoveryActions(lastResultPtr)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"checkId": checkID,
		"actions": actions,
	}); err != nil {
		apierrors.LogError("get_check_actions", "encode_response", err)
	}
}

// ExecuteCheckAction executes a recovery action for a check
// [REQ:HEAL-ACTION-001]
func (h *Handlers) ExecuteCheckAction(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	checkID := vars["checkId"]
	actionID := vars["actionId"]

	// Get the check and verify it's healable
	healable, ok := h.registry.GetHealableCheck(checkID)
	if !ok {
		apierrors.LogAndRespond(w, apierrors.NewNotFoundError("actions", "healable check", checkID))
		return
	}

	// Execute the action with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	result := healable.ExecuteAction(ctx, actionID)

	// Log the action to the database
	if err := h.store.SaveActionLog(
		ctx,
		result.CheckID,
		result.ActionID,
		result.Success,
		result.TimedOut,
		result.Message,
		result.Output,
		result.Error,
		result.Duration.Milliseconds(),
	); err != nil {
		apierrors.LogError("execute_action", "save_action_log", err)
	}

	// Return the result
	w.Header().Set("Content-Type", "application/json")
	statusCode := http.StatusOK
	if !result.Success {
		statusCode = http.StatusInternalServerError
	}
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		apierrors.LogError("execute_action", "encode_response", err)
	}
}

// GetActionHistory returns the action log history
// [REQ:HEAL-ACTION-001]
func (h *Handlers) GetActionHistory(w http.ResponseWriter, r *http.Request) {
	// Parse optional checkId filter from query
	checkID := r.URL.Query().Get("checkId")
	limit := 50

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var logs *persistence.ActionLogsResponse
	var err error

	if checkID != "" {
		logs, err = h.store.GetActionLogsForCheck(ctx, checkID, limit)
	} else {
		logs, err = h.store.GetActionLogs(ctx, limit)
	}

	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("action_history", "retrieve action logs", err))
		return
	}

	// Return empty array instead of null
	if logs.Logs == nil {
		logs.Logs = []persistence.ActionLog{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		apierrors.LogError("action_history", "encode_response", err)
	}
}
