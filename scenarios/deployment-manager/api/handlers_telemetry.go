package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type telemetryUploadRequest struct {
	ScenarioName   string                   `json:"scenario_name"`
	DeploymentMode string                   `json:"deployment_mode"`
	Source         string                   `json:"source"`
	Events         []map[string]interface{} `json:"events"`
}

type telemetrySummary struct {
	Scenario       string                   `json:"scenario"`
	Path           string                   `json:"path"`
	TotalEvents    int                      `json:"total_events"`
	LastEvent      string                   `json:"last_event,omitempty"`
	LastTimestamp  string                   `json:"last_timestamp,omitempty"`
	FailureCounts  map[string]int           `json:"failure_counts,omitempty"`
	RecentFailures []map[string]interface{} `json:"recent_failures,omitempty"`
	RecentEvents   []map[string]interface{} `json:"recent_events,omitempty"`
}

func telemetryDir() string {
	return GetConfigResolver().ResolveTelemetryDir()
}

func (s *Server) handleUploadTelemetry(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Parse events from request body (supports JSON and JSONL formats)
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	events, req, err := parseEventsFromRequest(r.Body, contentType, 4<<20)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(events) == 0 {
		http.Error(w, "no telemetry events provided", http.StatusBadRequest)
		return
	}

	// Resolve parameters from query string and request body
	params := resolveUploadParams(r.URL.Query(), req)

	// Enrich events with default fields
	now := GetTimeProvider().Now().UTC().Format(time.RFC3339)
	enrichEvents(events, params, now)

	// Ensure telemetry directory exists
	dir := telemetryDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, fmt.Sprintf("prepare telemetry dir: %s", err), http.StatusInternalServerError)
		return
	}

	// Append events to scenario-specific file
	path := filepath.Join(dir, fmt.Sprintf("%s.jsonl", params.Scenario))
	if err := appendEventsToFile(path, events); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"status":          "ok",
		"events_ingested": len(events),
		"scenario":        params.Scenario,
		"path":            path,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleListTelemetry(w http.ResponseWriter, r *http.Request) {
	dir := telemetryDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]telemetrySummary{})
			return
		}
		http.Error(w, fmt.Sprintf("read telemetry dir: %s", err), http.StatusInternalServerError)
		return
	}

	summaries := []telemetrySummary{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		summary, err := summarizeTelemetryFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		summaries = append(summaries, summary)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summaries)
}

func parseJSONL(r io.Reader) ([]map[string]interface{}, error) {
	var events []map[string]interface{}
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 1024*64), 1024*1024) // allow up to 1MB per line
	lineNum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lineNum++
		if line == "" {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
		events = append(events, evt)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func summarizeTelemetryFile(path string) (telemetrySummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return telemetrySummary{}, err
	}
	defer file.Close()

	scenario := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	summary := telemetrySummary{
		Scenario:       scenario,
		Path:           path,
		FailureCounts:  map[string]int{},
		RecentFailures: []map[string]interface{}{},
		RecentEvents:   []map[string]interface{}{},
	}

	const maxRecentEvents = 5
	const maxRecentFailures = 5

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1024*64), 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}

		summary.TotalEvents++
		if ts, ok := evt["timestamp"].(string); ok {
			summary.LastTimestamp = ts
		}
		if ev, ok := evt["event"].(string); ok {
			summary.LastEvent = ev
			if isFailureEvent(ev) {
				summary.FailureCounts[ev]++
				if len(summary.RecentFailures) < maxRecentFailures {
					summary.RecentFailures = append(summary.RecentFailures, evt)
				}
			}
		}

		// Keep the most recent N events using a sliding window
		summary.RecentEvents = append(summary.RecentEvents, evt)
		if len(summary.RecentEvents) > maxRecentEvents {
			summary.RecentEvents = summary.RecentEvents[1:] // drop oldest
		}
	}

	return summary, nil
}

// isFailureEvent delegates to domain_telemetry.go for failure event classification.
// See IsFailureEvent for the decision criteria and event list.
func isFailureEvent(event string) bool {
	return IsFailureEvent(event)
}

// telemetryUploadParams holds resolved parameters for telemetry uploads.
type telemetryUploadParams struct {
	Scenario string
	Mode     string
	Source   string
}

// parseEventsFromRequest extracts telemetry events from the request body.
// Supports both JSON (with wrapper) and JSONL formats based on Content-Type.
func parseEventsFromRequest(r io.Reader, contentType string, maxBytes int64) ([]map[string]interface{}, *telemetryUploadRequest, error) {
	limited := io.LimitReader(r, maxBytes)

	// JSON format: expects {"events": [...], "scenario_name": "...", ...}
	if strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "jsonl") {
		var req telemetryUploadRequest
		if err := json.NewDecoder(limited).Decode(&req); err != nil {
			return nil, nil, fmt.Errorf("invalid json payload: %w", err)
		}
		return req.Events, &req, nil
	}

	// JSONL format: one JSON object per line
	events, err := parseJSONL(limited)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid jsonl payload: %w", err)
	}
	return events, nil, nil
}

// resolveUploadParams extracts scenario/mode/source from query params and request body,
// with query params taking precedence. Returns sensible defaults for missing values.
func resolveUploadParams(query url.Values, req *telemetryUploadRequest) telemetryUploadParams {
	params := telemetryUploadParams{
		Scenario: "unknown",
		Mode:     "bundled",
		Source:   "upload",
	}

	// Try query params first, then request body
	if v := strings.TrimSpace(query.Get("scenario")); v != "" {
		params.Scenario = v
	} else if req != nil && strings.TrimSpace(req.ScenarioName) != "" {
		params.Scenario = strings.TrimSpace(req.ScenarioName)
	}

	if v := strings.TrimSpace(query.Get("mode")); v != "" {
		params.Mode = v
	} else if req != nil && strings.TrimSpace(req.DeploymentMode) != "" {
		params.Mode = strings.TrimSpace(req.DeploymentMode)
	}

	if v := strings.TrimSpace(query.Get("source")); v != "" {
		params.Source = v
	} else if req != nil && strings.TrimSpace(req.Source) != "" {
		params.Source = strings.TrimSpace(req.Source)
	}

	return params
}

// enrichEvents adds default fields (timestamp, scenario, mode, source) to events that lack them.
func enrichEvents(events []map[string]interface{}, params telemetryUploadParams, timestamp string) {
	for _, evt := range events {
		if _, ok := evt["timestamp"]; !ok {
			evt["timestamp"] = timestamp
		}
		if _, ok := evt["scenario_name"]; !ok {
			evt["scenario_name"] = params.Scenario
		}
		if _, ok := evt["deployment_mode"]; !ok {
			evt["deployment_mode"] = params.Mode
		}
		if _, ok := evt["source"]; !ok {
			evt["source"] = params.Source
		}
	}
}

// appendEventsToFile writes events as JSONL to the specified file, creating it if needed.
func appendEventsToFile(path string, events []map[string]interface{}) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open telemetry file: %w", err)
	}
	defer file.Close()

	for _, evt := range events {
		data, err := json.Marshal(evt)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("write telemetry: %w", err)
		}
	}
	return nil
}
