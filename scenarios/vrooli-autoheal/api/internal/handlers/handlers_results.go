package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	apierrors "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/errors"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/persistence"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/systemevents"
)

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
	summary := map[string]int{"ok": 0, "warning": 0, "critical": 0, "not-applicable": 0}
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
