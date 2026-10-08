package readiness

import (
	"encoding/json"
	"net/http"

	"scenario-to-ios/internal/module"
	"scenario-to-ios/internal/readiness"

	"github.com/gorilla/mux"
)

func Module(probe readiness.Probe) module.Module {
	return module.Module{Name: "ios-readiness", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/ios/readiness", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, readiness.FromProbe(probe)) }).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
