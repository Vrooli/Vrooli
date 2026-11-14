package healthapi

import (
	"net/http"

	"research-desk/internal/module"

	"github.com/gorilla/mux"
)

// Module mounts the health handler at /health (lifecycle probes) and
// /api/v1/health (API clients).
func Module(d Deps) module.Module {
	h := NewHandler(d)
	return module.Module{
		Name: "health",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/health", h).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/health", h).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}
