package content

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Customize preserves the current asynchronous job-envelope contract while
// keeping transport independent of the API composition root.
func Customize(now func() time.Time) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ScenarioID string   `json:"scenario_id"`
			Brief      string   `json:"brief"`
			Assets     []string `json:"assets"`
			Preview    bool     `json:"preview"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		response := map[string]any{"job_id": fmt.Sprintf("job-%d", now().Unix()), "status": "queued", "agent_id": "agent-claude-code-1"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
