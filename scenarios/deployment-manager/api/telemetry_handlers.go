package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	if override := strings.TrimSpace(os.Getenv("DEPLOYMENT_MANAGER_TELEMETRY_DIR")); override != "" {
		return override
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", ".vrooli", "deployment", "telemetry")
	}
	return filepath.Join(home, ".vrooli", "deployment", "telemetry")
}

func (s *Server) handleUploadTelemetry(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	var req telemetryUploadRequest
	var events []map[string]interface{}

	switch {
	case strings.Contains(contentType, "application/json") && !strings.Contains(contentType, "jsonl"):
		if err := json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("invalid json payload: %s", err), http.StatusBadRequest)
			return
		}
		events = req.Events
	default:
		parsed, err := parseJSONL(io.LimitReader(r.Body, 4<<20))
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid jsonl payload: %s", err), http.StatusBadRequest)
			return
		}
		events = parsed
	}

	if len(events) == 0 {
		http.Error(w, "no telemetry events provided", http.StatusBadRequest)
		return
	}

	scenario := strings.TrimSpace(r.URL.Query().Get("scenario"))
	if scenario == "" {
		scenario = strings.TrimSpace(req.ScenarioName)
	}
	if scenario == "" {
		scenario = "unknown"
	}
	mode := strings.TrimSpace(r.URL.Query().Get("mode"))
	if mode == "" {
		mode = strings.TrimSpace(req.DeploymentMode)
	}
	if mode == "" {
		mode = "bundled"
	}
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source == "" {
		source = strings.TrimSpace(req.Source)
	}
	if source == "" {
		source = "upload"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, evt := range events {
		if _, ok := evt["timestamp"]; !ok {
			evt["timestamp"] = now
		}
		if _, ok := evt["scenario_name"]; !ok {
			evt["scenario_name"] = scenario
		}
		if _, ok := evt["deployment_mode"]; !ok {
			evt["deployment_mode"] = mode
		}
		if _, ok := evt["source"]; !ok {
			evt["source"] = source
		}
	}

	dir := telemetryDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		http.Error(w, fmt.Sprintf("prepare telemetry dir: %s", err), http.StatusInternalServerError)
		return
	}

	path := filepath.Join(dir, fmt.Sprintf("%s.jsonl", scenario))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		http.Error(w, fmt.Sprintf("open telemetry file: %s", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	for _, evt := range events {
		data, err := json.Marshal(evt)
		if err != nil {
			http.Error(w, fmt.Sprintf("marshal event: %s", err), http.StatusBadRequest)
			return
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			http.Error(w, fmt.Sprintf("write telemetry: %s", err), http.StatusInternalServerError)
			return
		}
	}

	resp := map[string]interface{}{
		"status":          "ok",
		"events_ingested": len(events),
		"scenario":        scenario,
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
	var lastEvent, lastTimestamp string
	failureCounts := map[string]int{}
	recentFailures := []map[string]interface{}{}
	recentEvents := []map[string]interface{}{}
	total := 0

	decoder := bufio.NewScanner(file)
	decoder.Buffer(make([]byte, 0, 1024*64), 1024*1024)
	for decoder.Scan() {
		line := strings.TrimSpace(decoder.Text())
		if line == "" {
			continue
		}
		var evt map[string]interface{}
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}
		total++
		if ts, ok := evt["timestamp"].(string); ok {
			lastTimestamp = ts
		}
		if ev, ok := evt["event"].(string); ok {
			lastEvent = ev
			if isFailureEvent(ev) {
				failureCounts[ev]++
				if len(recentFailures) < 5 {
					recentFailures = append(recentFailures, evt)
				}
			}
		}
		if len(recentEvents) < 5 {
			recentEvents = append(recentEvents, evt)
		} else {
			recentEvents[total%5] = evt
		}
	}

	return telemetrySummary{
		Scenario:       scenario,
		Path:           path,
		TotalEvents:    total,
		LastEvent:      lastEvent,
		LastTimestamp:  lastTimestamp,
		FailureCounts:  failureCounts,
		RecentFailures: recentFailures,
		RecentEvents:   recentEvents,
	}, nil
}

func isFailureEvent(event string) bool {
	switch event {
	case "dependency_unreachable", "swap_missing_asset", "asset_missing", "migration_failed", "api_unreachable", "secrets_missing", "health_failed":
		return true
	default:
		return false
	}
}
