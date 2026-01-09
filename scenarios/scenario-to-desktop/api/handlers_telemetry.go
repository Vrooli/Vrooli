package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "ok",
		"events_ingested": len(request.Events),
		"output_path":     filePath,
	})
}
