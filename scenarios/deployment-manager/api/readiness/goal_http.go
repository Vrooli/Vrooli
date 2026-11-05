package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type goalOpener interface {
	Open(ctx context.Context, spec GoalSpec) (string, bool, error)
}

type goalRequest struct {
	Scenario    string   `json:"scenario"`
	Commit      string   `json:"commit"`
	Deliverable string   `json:"deliverable,omitempty"`
	Trigger     string   `json:"trigger,omitempty"`
	Signals     []Signal `json:"signals"`
}

// GoalHandler aggregates the supplied producer evidence and opens the
// deterministic swarm-manager readiness goal populated from the checklist.
func GoalHandler(opener goalOpener, checklist Checklist) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request goalRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid readiness goal request", http.StatusBadRequest)
			return
		}
		verdict, err := Aggregate(request.Scenario, request.Commit, checklist, request.Signals, time.Now().UTC())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		spec, err := BuildGoalSpec(request.Scenario, request.Commit, request.Deliverable, request.Trigger, checklist, verdict)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if opener == nil {
			http.Error(w, "readiness goal opener is unavailable", http.StatusServiceUnavailable)
			return
		}
		name, deduped, err := opener.Open(r.Context(), spec)
		if err != nil {
			http.Error(w, fmt.Sprintf("open readiness goal: %v", err), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"goal": name, "deduped": deduped, "verdict": verdict})
	})
}
