package readiness

import (
	"encoding/json"
	"net/http"
	"time"

	"deployment-manager/crossosgate"
)

type aggregateRequest struct {
	Scenario string               `json:"scenario"`
	Commit   string               `json:"commit"`
	Signals  []Signal             `json:"signals"`
	CrossOS  *crossosgate.Verdict `json:"cross_os_verdict,omitempty"`
}

// Handler exposes the typed readiness aggregation seam. Signal producers post
// their already-attributable observations; this handler never invents a pass.
func Handler(checklist Checklist) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var request aggregateRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid readiness request", http.StatusBadRequest)
			return
		}
		now := time.Now().UTC()
		if request.CrossOS != nil {
			request.Signals = append(request.Signals, SignalFromCrossOS(*request.CrossOS, now))
		}
		verdict, err := Aggregate(request.Scenario, request.Commit, checklist, request.Signals, now)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(verdict)
	})
}
