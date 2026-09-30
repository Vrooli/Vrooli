package journeys

import (
	"encoding/json"
	"net/http"

	"scenario-to-android/internal/journeys"
)

func ConformancePlan(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(journeys.AndroidPlan())
}
