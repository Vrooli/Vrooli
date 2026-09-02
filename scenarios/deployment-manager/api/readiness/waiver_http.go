package readiness

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type waiverRecorder interface {
	RecordReadinessWaiver(context.Context, string, string, string, string) error
}

type waiverRequest struct {
	ProfileID string `json:"profile_id"`
	Commit    string `json:"commit"`
	Reason    string `json:"reason"`
	Actor     string `json:"actor"`
}

// WaiverHandler is the narrow operator exception seam. It records only a
// reasoned, actor-bound waiver for an exact commit; it never marks readiness
// approved or changes the deployment gate itself.
func WaiverHandler(recorder waiverRecorder) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if recorder == nil {
			http.Error(w, "readiness waiver recorder is unavailable", http.StatusServiceUnavailable)
			return
		}
		var request waiverRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid readiness waiver request", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(request.ProfileID) == "" || strings.TrimSpace(request.Commit) == "" || strings.TrimSpace(request.Reason) == "" || strings.TrimSpace(request.Actor) == "" {
			http.Error(w, "profile_id, commit, reason, and actor are required", http.StatusBadRequest)
			return
		}
		if err := recorder.RecordReadinessWaiver(r.Context(), request.ProfileID, request.Commit, request.Reason, request.Actor); err != nil {
			http.Error(w, fmt.Sprintf("record readiness waiver: %v", err), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"recorded": true, "profile_id": request.ProfileID, "commit": request.Commit, "actor": request.Actor})
	})
}
