package journeys

import (
	"encoding/json"
	"net/http"

	"scenario-to-ios/internal/journeys"
	"scenario-to-ios/internal/module"

	"github.com/gorilla/mux"
)

func Module() module.Module {
	return module.Module{Name: "ios-journeys", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/ios/conformance-plan", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, journeys.Plan()) }).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
