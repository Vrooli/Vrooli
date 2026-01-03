package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"scenario-to-cloud/domain"
)

// handleGetHistory returns the deployment history timeline.
// GET /api/v1/deployments/{id}/history
func (s *Server) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Get deployment from database
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Get deployment history
	history, err := s.repo.GetDeploymentHistory(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
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

	writeJSON(w, http.StatusOK, map[string]interface{}{
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

// handleGetLogs returns aggregated logs from the VPS.
// GET /api/v1/deployments/{id}/logs?source=all&level=all&tail=100&search=
func (s *Server) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

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

	// Get deployment
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
			Code:    "not_found",
			Message: "Deployment not found",
		})
		return
	}

	// Parse manifest
	var manifest CloudManifest
	if err := json.Unmarshal(deployment.Manifest, &manifest); err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "manifest_parse_failed",
			Message: "Failed to parse deployment manifest",
			Hint:    err.Error(),
		})
		return
	}

	normalized, _ := ValidateAndNormalizeManifest(manifest)
	if normalized.Target.VPS == nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "no_vps_target",
			Message: "Deployment does not have a VPS target",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	// Fetch logs from VPS
	logs, sources, err := fetchAggregatedLogs(ctx, normalized, s.sshRunner, tail, source, level, search)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "logs_failed",
			Message: "Failed to fetch logs from VPS",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, LogsResponse{
		OK:       true,
		Logs:     logs,
		Total:    len(logs),
		Filtered: len(logs),
		Sources:  sources,
	})
}

// fetchAggregatedLogs fetches logs from multiple sources on the VPS.
func fetchAggregatedLogs(ctx context.Context, manifest CloudManifest, sshRunner SSHRunner, tail int, sourceFilter, levelFilter, search string) ([]LogEntry, []string, error) {
	cfg := sshConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	scenarioID := manifest.Scenario.ID

	var allLogs []LogEntry
	sources := []string{}

	// Determine which sources to fetch
	sourcesToFetch := []struct {
		id  string
		cmd string
	}{}

	if sourceFilter == "all" || sourceFilter == "scenario" || sourceFilter == scenarioID {
		sourcesToFetch = append(sourcesToFetch, struct {
			id  string
			cmd string
		}{
			id:  scenarioID,
			cmd: vrooliCommand(workdir, "vrooli scenario logs "+shellQuoteSingle(scenarioID)+" --tail "+intToStr(tail)),
		})
	}

	if sourceFilter == "all" || sourceFilter == "caddy" {
		sourcesToFetch = append(sourcesToFetch, struct {
			id  string
			cmd string
		}{
			id:  "caddy",
			cmd: "journalctl -u caddy --no-pager -n " + intToStr(tail) + " 2>/dev/null || echo 'No caddy logs available'",
		})
	}

	// Add resource logs
	if sourceFilter == "all" {
		for _, res := range manifest.Dependencies.Resources {
			sourcesToFetch = append(sourcesToFetch, struct {
				id  string
				cmd string
			}{
				id:  res,
				cmd: vrooliCommand(workdir, "vrooli resource logs "+shellQuoteSingle(res)+" --tail "+intToStr(tail/4)+" 2>/dev/null || echo 'No logs for "+res+"'"),
			})
		}
	} else {
		// Check if sourceFilter is a specific resource
		for _, res := range manifest.Dependencies.Resources {
			if sourceFilter == res {
				sourcesToFetch = append(sourcesToFetch, struct {
					id  string
					cmd string
				}{
					id:  res,
					cmd: vrooliCommand(workdir, "vrooli resource logs "+shellQuoteSingle(res)+" --tail "+intToStr(tail)+" 2>/dev/null || echo 'No logs for "+res+"'"),
				})
			}
		}
	}

	// Fetch logs from each source
	for _, src := range sourcesToFetch {
		result, err := sshRunner.Run(ctx, cfg, src.cmd)
		if err != nil {
			continue // Skip failed sources
		}

		sources = append(sources, src.id)

		// Parse the log output
		entries := parseLogOutput(result.Stdout, src.id)

		// Apply level filter
		if levelFilter != "all" {
			filtered := []LogEntry{}
			for _, entry := range entries {
				if strings.EqualFold(entry.Level, levelFilter) {
					filtered = append(filtered, entry)
				}
			}
			entries = filtered
		}

		// Apply search filter
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

	// Sort all logs by timestamp
	sort.Slice(allLogs, func(i, j int) bool {
		return allLogs[i].Timestamp > allLogs[j].Timestamp
	})

	// Limit to tail
	if len(allLogs) > tail {
		allLogs = allLogs[:tail]
	}

	return allLogs, sources, nil
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

	req, err := decodeJSON[domain.HistoryEvent](r.Body, 1<<20)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, APIError{
			Code:    "invalid_json",
			Message: "Request body must be valid JSON",
			Hint:    err.Error(),
		})
		return
	}

	// Verify deployment exists
	deployment, err := s.repo.GetDeployment(r.Context(), id)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "get_failed",
			Message: "Failed to get deployment",
			Hint:    err.Error(),
		})
		return
	}

	if deployment == nil {
		writeAPIError(w, http.StatusNotFound, APIError{
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
		writeAPIError(w, http.StatusInternalServerError, APIError{
			Code:    "append_failed",
			Message: "Failed to append history event",
			Hint:    err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ok":        true,
		"event":     req,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
