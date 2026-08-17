package readiness

import (
	"encoding/json"
	"net/http"

	"scenario-to-android/internal/readiness"
)

func Handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(readiness.GoogleReadiness(false, false, false, true, false, false))
}
