package readiness

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"deployment-manager/releases"
)

type latestStateReader interface {
	GetLatestReadiness(context.Context, string) (*releases.ReadinessRecord, error)
}

// StateHandler exposes the latest release-side projection. It is intentionally
// a projection only; deployment-manager remains the owner of the verdict.
func StateHandler(reader latestStateReader) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		profile := strings.TrimSpace(r.URL.Query().Get("scenario"))
		if profile == "" {
			http.Error(w, "scenario is required", http.StatusBadRequest)
			return
		}
		if reader == nil {
			http.Error(w, "readiness state unavailable", http.StatusServiceUnavailable)
			return
		}
		record, err := reader.GetLatestReadiness(r.Context(), profile)
		if err != nil {
			http.Error(w, "readiness state unavailable", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"goal_exists": strings.TrimSpace(record.ReadinessGoalRef) != "", "goal_closed": record.GoalClosed, "approved_commit": record.ApprovedAtCommit})
	})
}
