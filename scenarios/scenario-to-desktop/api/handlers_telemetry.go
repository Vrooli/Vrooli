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
		event["scenario_name"] = request.ScenarioName
		event["deployment_mode"] = request.DeploymentMode
		event["source"] = request.Source
		event["ingested_at"] = now

		data, err := json.Marshal(event)
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
