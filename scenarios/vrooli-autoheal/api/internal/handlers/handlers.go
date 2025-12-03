// Package handlers provides HTTP request handlers for the autoheal API
// [REQ:CLI-TICK-001] [REQ:CLI-TICK-002] [REQ:CLI-STATUS-001] [REQ:CLI-STATUS-002]
// [REQ:FAIL-SAFE-001] [REQ:FAIL-OBSERVE-001] [REQ:WATCH-DETECT-001]
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"vrooli-autoheal/internal/checks"
	apierrors "vrooli-autoheal/internal/errors"
	"vrooli-autoheal/internal/persistence"
	"vrooli-autoheal/internal/platform"
	"vrooli-autoheal/internal/watchdog"
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
	GetIncidents(ctx context.Context, windowHours, limit int) (*persistence.IncidentsResponse, error)
}

// Handlers wraps the dependencies needed by HTTP handlers
type Handlers struct {
	registry         *checks.Registry
	store            StoreInterface
	platform         *platform.Capabilities
	watchdogDetector *watchdog.Detector
}

// New creates a new Handlers instance
func New(registry *checks.Registry, store *persistence.Store, plat *platform.Capabilities) *Handlers {
	return &Handlers{
		registry:         registry,
		store:            store,
		platform:         plat,
		watchdogDetector: watchdog.NewDetector(plat),
	}
}

// NewWithInterface creates a new Handlers instance with an interface-based store (for testing)
func NewWithInterface(registry *checks.Registry, store StoreInterface, plat *platform.Capabilities) *Handlers {
	return &Handlers{
		registry:         registry,
		store:            store,
		platform:         plat,
		watchdogDetector: watchdog.NewDetector(plat),
	}
}

// Health returns basic service health for lifecycle checks
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := h.store.Ping(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Vrooli Autoheal API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Platform returns detected platform capabilities
func (h *Handlers) Platform(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.platform)
}

// Status returns the current health summary
func (h *Handlers) Status(w http.ResponseWriter, r *http.Request) {
	summary := h.registry.GetSummary()

	response := map[string]interface{}{
		"status":   summary.Status,
		"platform": h.platform,
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
	json.NewEncoder(w).Encode(response)
}

// Tick runs a single health check cycle
func (h *Handlers) Tick(w http.ResponseWriter, r *http.Request) {
	// Parse force parameter
	forceAll := r.URL.Query().Get("force") == "true"

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	results := h.registry.RunAll(ctx, forceAll)

	// Store results in database - log failures but don't block the response
	// [REQ:FAIL-SAFE-001] Tick completes even if persistence fails
	var persistenceErrors int
	for _, result := range results {
		if err := h.store.SaveResult(ctx, result); err != nil {
			persistenceErrors++
			apierrors.LogError("tick", "save_result:"+result.CheckID, err)
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
		"results":   results,
		"timestamp": time.Now().UTC(),
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

// ListChecks returns all registered checks
func (h *Handlers) ListChecks(w http.ResponseWriter, r *http.Request) {
	checks := h.registry.ListChecks()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(checks)
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

// Incidents returns status transition events
// [REQ:PERSIST-HISTORY-001]
func (h *Handlers) Incidents(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters with defaults
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

	incidents, err := h.store.GetIncidents(ctx, windowHours, limit)
	if err != nil {
		apierrors.LogAndRespond(w, apierrors.NewDatabaseError("incidents", "retrieve incidents", err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(incidents); err != nil {
		apierrors.LogError("incidents", "encode_response", err)
	}
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
