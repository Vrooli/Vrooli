package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// handleCustomize accepts a customization request and returns the queued job
// envelope. The execution engine remains a future capability, but transport
// ownership belongs outside the composition root.
func (s *Server) handleCustomize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScenarioID string   `json:"scenario_id"`
		Brief      string   `json:"brief"`
		Assets     []string `json:"assets"`
		Preview    bool     `json:"preview"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"job_id":   fmt.Sprintf("job-%d", time.Now().Unix()),
		"status":   "queued",
		"agent_id": "agent-claude-code-1",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
