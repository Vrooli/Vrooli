package main

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/vps"

	"github.com/gorilla/mux"
)

// handleGetHistory returns the deployment history timeline.
// GET /api/v1/deployments/{id}/history
func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Get deployment from database
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Get deployment history
	history, err := s.repo.GetDeploymentHistory(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "history_failed",
			Message: "Failed to get deployment history",
			Hint:    err.Error(),
		})
		return
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp.After(history[j].Timestamp)
	})

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"history":   history,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// LogEntry represents a single log line.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// LogsResponse contains the aggregated logs from VPS.
type LogsResponse struct {
	OK       bool       `json:"ok"`
	Logs     []LogEntry `json:"logs"`
	Total    int        `json:"total"`
	Filtered int        `json:"filtered"`
	Sources  []string   `json:"sources"`
}

// handleGetLogs returns aggregated logs from the target.
// GET /api/v1/deployments/{id}/logs?source=all&level=all&tail=100&search=
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	source := r.URL.Query().Get("source")
	if source == "" {
		source = "all"
	}
	level := r.URL.Query().Get("level")
	if level == "" {
		level = "all"
	}
	tailStr := r.URL.Query().Get("tail")
	tail := 200
	if tailStr != "" {
		if t, err := strconv.Atoi(tailStr); err == nil && t > 0 && t <= 2000 {
			tail = t
		}
	}
	search := r.URL.Query().Get("search")

	dc := s.FetchDeploymentContext(w, r)
	if dc == nil {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	logs, sources := fetchAggregatedLogs(ctx, dc.Manifest, s.proberFor(dc), tail, source, level, search)
	if logs == nil {
		logs = []LogEntry{}
	}

	httputil.WriteJSON(w, http.StatusOK, LogsResponse{
		OK:       true,
		Logs:     logs,
		Total:    len(logs),
		Filtered: len(logs),
		Sources:  sources,
	})
}

// logSource is one typed log read: the lifecycle owner's logs verbs for
// scenarios and resources, journalctl for the edge proxy unit.
type logSource struct {
	id  string
	run func(ctx context.Context) (reach.Result, error)
}

// logSources lists the reads for a source filter. Every identifier is
// validated as an argv value before it can reach a transport.
func logSources(manifest domain.CloudManifest, prober vps.Prober, tail int, sourceFilter string) []logSource {
	scenarioID := manifest.Scenario.ID
	var sources []logSource
	if sourceFilter == "all" || sourceFilter == "scenario" || sourceFilter == scenarioID {
		sources = append(sources, logSource{id: scenarioID, run: func(ctx context.Context) (reach.Result, error) {
			return prober.Verb(ctx, "scenario logs", scenarioID, "--tail", strconv.Itoa(tail))
		}})
	}
	if sourceFilter == "all" || sourceFilter == "caddy" {
		journalTail := tail
		if journalTail > 200 {
			journalTail = 200
		}
		sources = append(sources, logSource{id: "caddy", run: func(ctx context.Context) (reach.Result, error) {
			return prober.Observe(ctx, "journalctl", "-u", "caddy", "--no-pager", "-n", strconv.Itoa(journalTail))
		}})
	}
	for _, res := range manifest.Dependencies.Resources {
		res := res
		resourceTail := tail
		if sourceFilter == "all" {
			resourceTail = tail / 4
		} else if sourceFilter != res {
			continue
		}
		if resourceTail <= 0 {
			resourceTail = 1
		}
		sources = append(sources, logSource{id: res, run: func(ctx context.Context) (reach.Result, error) {
			return prober.Verb(ctx, "resource logs", res, "--tail", strconv.Itoa(resourceTail))
		}})
	}
	return sources
}

// fetchAggregatedLogs fetches logs from every selected source through the
// prober. A source that fails to answer is skipped and left out of the
// sources list so the caller can see which producers reported.
func fetchAggregatedLogs(ctx context.Context, manifest domain.CloudManifest, prober vps.Prober, tail int, sourceFilter, levelFilter, search string) ([]LogEntry, []string) {
	var allLogs []LogEntry
	sources := []string{}

	for _, src := range logSources(manifest, prober, tail, sourceFilter) {
		result, err := src.run(ctx)
		if err != nil || result.ExitCode != 0 {
			continue
		}

		sources = append(sources, src.id)
		entries := parseLogOutput(result.Stdout, src.id)

		if levelFilter != "all" {
			filtered := []LogEntry{}
			for _, entry := range entries {
				if strings.EqualFold(entry.Level, levelFilter) {
					filtered = append(filtered, entry)
				}
			}
			entries = filtered
		}

		if search != "" {
			filtered := []LogEntry{}
			searchLower := strings.ToLower(search)
			for _, entry := range entries {
				if strings.Contains(strings.ToLower(entry.Message), searchLower) {
					filtered = append(filtered, entry)
				}
			}
			entries = filtered
		}

		allLogs = append(allLogs, entries...)
	}

	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Timestamp > allLogs[j].Timestamp
	})

	if len(allLogs) > tail {
		allLogs = allLogs[:tail]
	}

	return allLogs, sources
}

// parseLogOutput parses log output into structured log entries.
func parseLogOutput(output, source string) []LogEntry {
	var entries []LogEntry
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		entry := LogEntry{
			Source:  source,
			Message: line,
			Level:   "INFO",
		}

		// Try to parse timestamp from common formats
		// Format 1: "2025-12-30T12:34:56Z ..."
		if len(line) > 20 && (line[4] == '-' && line[7] == '-') {
			parts := strings.SplitN(line, " ", 2)
			if len(parts) == 2 {
				entry.Timestamp = parts[0]
				line = parts[1]
			}
		}

		// Format 2: journalctl format "Dec 30 12:34:56 hostname ..."
		if entry.Timestamp == "" && len(line) > 15 {
			months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
			for _, m := range months {
				if strings.HasPrefix(line, m+" ") {
					parts := strings.Fields(line)
					if len(parts) >= 4 {
						// Construct timestamp
						entry.Timestamp = parts[0] + " " + parts[1] + " " + parts[2]
						line = strings.Join(parts[3:], " ")
						break
					}
				}
			}
		}

		// If still no timestamp, use current time
		if entry.Timestamp == "" {
			entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
		}

		// Detect log level from content
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "err]") || strings.Contains(lineLower, "[err") {
			entry.Level = "ERROR"
		} else if strings.Contains(lineLower, "warn") || strings.Contains(lineLower, "warning") {
			entry.Level = "WARN"
		} else if strings.Contains(lineLower, "debug") {
			entry.Level = "DEBUG"
		}

		entry.Message = line
		entries = append(entries, entry)
	}

	return entries
}

// handleAddHistoryEvent allows manually adding a history event (for testing/admin).
// POST /api/v1/deployments/{id}/history
func (s *Server) handleAddHistoryEvent(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	req, err := httputil.DecodeJSON[domain.HistoryEvent](r.Body, 1<<20)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusBadRequest, httputil.APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	// Verify deployment exists
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		httputil.WriteAPIError(w, http.StatusNotFound, httputil.APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Set timestamp if not provided
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now().UTC()
	}

	// Append the event
	if err := s.repo.AppendHistoryEvent(r.Context(), id, req); err != nil {
		httputil.WriteAPIError(w, http.StatusInternalServerError, httputil.APIError{
			Code:    "append_failed",
			Message: "Failed to append history event",
			Hint:    err.Error(),
		})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"event":     req,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
