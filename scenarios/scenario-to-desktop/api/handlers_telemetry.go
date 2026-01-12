package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// telemetryIngestHandler stores deployment telemetry emitted by generated desktop bundles.
func (s *Server) telemetryIngestHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		ScenarioName   string                   `json:"scenario_name"`
		DeploymentMode string                   `json:"deployment_mode"`
		Source         string                   `json:"source"`
		Events         []map[string]interface{} `json:"events"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	if request.ScenarioName == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}
	if len(request.Events) == 0 {
		http.Error(w, "events array cannot be empty", http.StatusBadRequest)
		return
	}
	if request.DeploymentMode == "" {
		request.DeploymentMode = "external-server"
	}

	vrooliRoot := detectVrooliRoot()
	telemetryDir := filepath.Join(vrooliRoot, ".vrooli", "deployment", "telemetry")
	if err := os.MkdirAll(telemetryDir, 0o755); err != nil {
		http.Error(w, fmt.Sprintf("failed to prepare telemetry directory: %s", err), http.StatusInternalServerError)
		return
	}

	filePath := filepath.Join(telemetryDir, fmt.Sprintf("%s.jsonl", request.ScenarioName))
	s.telemetryMux.Lock()
	defer s.telemetryMux.Unlock()

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open telemetry file: %s", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	for _, event := range request.Events {
		normalized := normalizeTelemetryEvent(event, request.ScenarioName, request.DeploymentMode, request.Source, now)
		data, err := json.Marshal(normalized)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to marshal telemetry event: %s", err), http.StatusBadRequest)
			return
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			http.Error(w, fmt.Sprintf("failed to write telemetry: %s", err), http.StatusInternalServerError)
			return
		}
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status":          "ok",
		"events_ingested": len(request.Events),
		"output_path":     filePath,
	})
}

func normalizeTelemetryEvent(event map[string]interface{}, scenario, deploymentMode, source, ingestedAt string) map[string]interface{} {
	normalized := make(map[string]interface{}, len(event)+6)
	for key, value := range event {
		normalized[key] = value
	}

	delete(normalized, "deploymentMode")
	delete(normalized, "deployment_mode")
	delete(normalized, "serverType")
	delete(normalized, "server_type")
	delete(normalized, "scenario_name")
	delete(normalized, "source")
	delete(normalized, "ingested_at")

	if serverType, ok := event["serverType"].(string); ok && serverType != "" {
		normalized["server_type"] = serverType
	}
	if serverType, ok := event["server_type"].(string); ok && serverType != "" {
		normalized["server_type"] = serverType
	}

	normalized["scenario_name"] = scenario
	normalized["deployment_mode"] = deploymentMode
	normalized["source"] = source
	normalized["ingested_at"] = ingestedAt
	if _, ok := normalized["level"]; !ok {
		normalized["level"] = "info"
	}

	return normalized
}

type telemetrySummaryResponse struct {
	ScenarioName   string `json:"scenario_name"`
	Exists         bool   `json:"exists"`
	FilePath       string `json:"file_path,omitempty"`
	FileSizeBytes  int64  `json:"file_size_bytes,omitempty"`
	EventCount     int    `json:"event_count,omitempty"`
	LastIngestedAt string `json:"last_ingested_at,omitempty"`
}

type telemetryTailEntry struct {
	Raw   string                 `json:"raw"`
	Event map[string]interface{} `json:"event,omitempty"`
	Error string                 `json:"error,omitempty"`
}

type telemetryTailResponse struct {
	ScenarioName string               `json:"scenario_name"`
	Exists       bool                 `json:"exists"`
	Limit        int                  `json:"limit"`
	TotalLines   int                  `json:"total_lines,omitempty"`
	Entries      []telemetryTailEntry `json:"entries,omitempty"`
}

type telemetryInsightsResponse struct {
	ScenarioName  string                       `json:"scenario_name"`
	Exists        bool                         `json:"exists"`
	LastSession   *telemetrySessionInsight     `json:"last_session,omitempty"`
	LastSmokeTest *telemetrySmokeTestInsight   `json:"last_smoke_test,omitempty"`
	LastError     *telemetryErrorInsight       `json:"last_error,omitempty"`
}

type telemetrySessionInsight struct {
	SessionID   string `json:"session_id,omitempty"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at,omitempty"`
	ReadyAt     string `json:"ready_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type telemetrySmokeTestInsight struct {
	SessionID   string `json:"session_id,omitempty"`
	Status      string `json:"status"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	Error       string `json:"error,omitempty"`
}

type telemetryErrorInsight struct {
	Event     string `json:"event"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message,omitempty"`
}

func (s *Server) telemetrySummaryHandler(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	filePath := s.telemetryFilePath(scenario)
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSONResponse(w, http.StatusOK, telemetrySummaryResponse{
				ScenarioName: scenario,
				Exists:       false,
			})
			return
		}
		http.Error(w, fmt.Sprintf("failed to read telemetry file: %s", err), http.StatusInternalServerError)
		return
	}

	eventCount, lastIngestedAt, err := readTelemetrySummary(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read telemetry summary: %s", err), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, telemetrySummaryResponse{
		ScenarioName:   scenario,
		Exists:         true,
		FilePath:       filePath,
		FileSizeBytes:  info.Size(),
		EventCount:     eventCount,
		LastIngestedAt: lastIngestedAt,
	})
}

func (s *Server) telemetryInsightsHandler(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	filePath := s.telemetryFilePath(scenario)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			writeJSONResponse(w, http.StatusOK, telemetryInsightsResponse{
				ScenarioName: scenario,
				Exists:       false,
			})
			return
		}
		http.Error(w, fmt.Sprintf("failed to read telemetry file: %s", err), http.StatusInternalServerError)
		return
	}

	insights, err := readTelemetryInsights(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read telemetry insights: %s", err), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, telemetryInsightsResponse{
		ScenarioName:  scenario,
		Exists:        true,
		LastSession:   insights.lastSession,
		LastSmokeTest: insights.lastSmokeTest,
		LastError:     insights.lastError,
	})
}

func (s *Server) telemetryTailHandler(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	limit := parseTelemetryTailLimit(r)
	filePath := s.telemetryFilePath(scenario)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			writeJSONResponse(w, http.StatusOK, telemetryTailResponse{
				ScenarioName: scenario,
				Exists:       false,
				Limit:        limit,
			})
			return
		}
		http.Error(w, fmt.Sprintf("failed to read telemetry file: %s", err), http.StatusInternalServerError)
		return
	}

	entries, totalLines, err := readTelemetryTail(filePath, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read telemetry tail: %s", err), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, telemetryTailResponse{
		ScenarioName: scenario,
		Exists:       true,
		Limit:        limit,
		TotalLines:   totalLines,
		Entries:      entries,
	})
}

func (s *Server) telemetryDownloadHandler(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	filePath := s.telemetryFilePath(scenario)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "telemetry not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to read telemetry file: %s", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	http.ServeFile(w, r, filePath)
}

func (s *Server) telemetryDeleteHandler(w http.ResponseWriter, r *http.Request) {
	scenario := mux.Vars(r)["scenario_name"]
	if scenario == "" {
		http.Error(w, "scenario_name is required", http.StatusBadRequest)
		return
	}

	filePath := s.telemetryFilePath(scenario)
	s.telemetryMux.Lock()
	defer s.telemetryMux.Unlock()
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "telemetry not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to delete telemetry: %s", err), http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"status": "deleted",
	})
}

func parseTelemetryTailLimit(r *http.Request) int {
	const (
		defaultLimit = 200
		maxLimit     = 1000
	)
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultLimit
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return defaultLimit
	}
	if parsed > maxLimit {
		return maxLimit
	}
	return parsed
}

func readTelemetrySummary(filePath string) (int, string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var count int
	var lastIngested time.Time
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		count++
		var payload map[string]interface{}
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		if raw, ok := payload["ingested_at"].(string); ok && raw != "" {
			if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
				if parsed.After(lastIngested) {
					lastIngested = parsed
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, "", err
	}
	lastIngestedAt := ""
	if !lastIngested.IsZero() {
		lastIngestedAt = lastIngested.Format(time.RFC3339)
	}
	return count, lastIngestedAt, nil
}

type telemetryInsights struct {
	lastSession   *telemetrySessionInsight
	lastSmokeTest *telemetrySmokeTestInsight
	lastError     *telemetryErrorInsight
}

var telemetryErrorEvents = map[string]bool{
	"startup_error":          true,
	"bundled_runtime_failed": true,
	"runtime_error":          true,
	"dependency_unreachable": true,
	"smoke_test_failed":      true,
	"migration_failed":       true,
	"asset_missing":          true,
	"asset_checksum_mismatch": true,
	"asset_size_exceeded":    true,
	"secrets_missing":        true,
	"runtime_secrets_missing": true,
	"service_not_ready":      true,
}

func readTelemetryInsights(filePath string) (*telemetryInsights, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	insights := &telemetryInsights{}
	var lastSessionTime time.Time
	var lastSmokeTime time.Time
	var lastErrorTime time.Time
	var lastAppStart time.Time
	var lastAppReady time.Time
	var lastAppShutdown time.Time

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		event, _ := payload["event"].(string)
		if event == "" {
			continue
		}
		ts, ok := eventTimestamp(payload)
		if !ok {
			continue
		}

		switch event {
		case "app_start":
			if ts.After(lastAppStart) {
				lastAppStart = ts
			}
		case "app_ready":
			if ts.After(lastAppReady) {
				lastAppReady = ts
			}
		case "app_shutdown":
			if ts.After(lastAppShutdown) {
				lastAppShutdown = ts
			}
		}

		if event == "app_session_succeeded" || event == "app_session_failed" {
			if ts.After(lastSessionTime) {
				lastSessionTime = ts
				insights.lastSession = buildSessionInsight(payload, event, ts)
			}
		}

		if event == "smoke_test_started" || event == "smoke_test_passed" || event == "smoke_test_failed" {
			if ts.After(lastSmokeTime) {
				lastSmokeTime = ts
				insights.lastSmokeTest = buildSmokeTestInsight(payload, event, ts)
			}
		}

		if isTelemetryError(payload, event) && ts.After(lastErrorTime) {
			lastErrorTime = ts
			insights.lastError = buildErrorInsight(payload, event, ts)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if insights.lastSession == nil && !lastAppStart.IsZero() {
		status := "failed"
		if lastAppReady.After(lastAppStart) {
			status = "succeeded"
		}
		insights.lastSession = &telemetrySessionInsight{
			Status:      status,
			StartedAt:   lastAppStart.Format(time.RFC3339),
			ReadyAt:     formatTimeIfSet(lastAppReady),
			CompletedAt: formatTimeIfSet(lastAppShutdown),
			Reason:      "",
		}
		if status == "failed" {
			insights.lastSession.Reason = "app_exit_before_ready"
		}
	}

	return insights, nil
}

func buildSessionInsight(payload map[string]interface{}, event string, ts time.Time) *telemetrySessionInsight {
	status := "succeeded"
	if event == "app_session_failed" {
		status = "failed"
	}
	insight := &telemetrySessionInsight{
		SessionID:   stringFromPayload(payload, "session_id"),
		Status:      status,
		CompletedAt: ts.Format(time.RFC3339),
	}
	if details, ok := payload["details"].(map[string]interface{}); ok {
		insight.StartedAt = stringFromMap(details, "started_at")
		insight.ReadyAt = stringFromMap(details, "ready_at")
		insight.Reason = stringFromMap(details, "reason")
	}
	return insight
}

func buildSmokeTestInsight(payload map[string]interface{}, event string, ts time.Time) *telemetrySmokeTestInsight {
	status := "started"
	if event == "smoke_test_passed" {
		status = "passed"
	}
	if event == "smoke_test_failed" {
		status = "failed"
	}
	insight := &telemetrySmokeTestInsight{
		SessionID: stringFromPayload(payload, "session_id"),
		Status:    status,
	}
	if event == "smoke_test_started" {
		insight.StartedAt = ts.Format(time.RFC3339)
	} else {
		insight.CompletedAt = ts.Format(time.RFC3339)
	}
	if details, ok := payload["details"].(map[string]interface{}); ok {
		insight.Error = stringFromMap(details, "error")
	}
	return insight
}

func buildErrorInsight(payload map[string]interface{}, event string, ts time.Time) *telemetryErrorInsight {
	message := ""
	if details, ok := payload["details"].(map[string]interface{}); ok {
		message = stringFromMap(details, "error")
		if message == "" {
			message = stringFromMap(details, "message")
		}
	}
	return &telemetryErrorInsight{
		Event:     event,
		Timestamp: ts.Format(time.RFC3339),
		Message:   message,
	}
}

func isTelemetryError(payload map[string]interface{}, event string) bool {
	if level, ok := payload["level"].(string); ok && level == "error" {
		return true
	}
	return telemetryErrorEvents[event]
}

func eventTimestamp(payload map[string]interface{}) (time.Time, bool) {
	for _, key := range []string{"timestamp", "ts", "ingested_at"} {
		if raw, ok := payload[key].(string); ok && raw != "" {
			parsed, err := time.Parse(time.RFC3339, raw)
			if err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

func formatTimeIfSet(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339)
}

func stringFromPayload(payload map[string]interface{}, key string) string {
	if raw, ok := payload[key].(string); ok {
		return raw
	}
	return ""
}

func stringFromMap(payload map[string]interface{}, key string) string {
	if raw, ok := payload[key].(string); ok {
		return raw
	}
	return ""
}

func readTelemetryTail(filePath string, limit int) ([]telemetryTailEntry, int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var totalLines int
	ring := make([]string, 0, limit)
	index := 0
	full := false

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		totalLines++
		if len(ring) < limit {
			ring = append(ring, line)
			continue
		}
		ring[index] = line
		full = true
		index++
		if index >= limit {
			index = 0
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, 0, err
	}

	ordered := ring
	if full && len(ring) > 0 {
		ordered = append(ring[index:], ring[:index]...)
	}

	entries := make([]telemetryTailEntry, 0, len(ordered))
	for _, line := range ordered {
		entry := telemetryTailEntry{Raw: line}
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			entry.Error = err.Error()
		} else {
			entry.Event = payload
		}
		entries = append(entries, entry)
	}

	return entries, totalLines, nil
}

func (s *Server) telemetryFilePath(scenario string) string {
	return filepath.Join(s.getVrooliRoot(), ".vrooli", "deployment", "telemetry", fmt.Sprintf("%s.jsonl", scenario))
}
