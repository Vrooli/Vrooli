package health

import (
	"net/http"

	"github.com/gorilla/mux"

	"{{SCENARIO_ID}}/internal/module"
	"{{SCENARIO_ID}}/internal/store"
)

// Module returns the health domain's contribution to the API: the
// /health and /api/v1/health routes plus the descriptor the codegen
// reads. The single handler is mounted at both paths — /health is the
// probe convention infrastructure (LB, Kubernetes) reaches for;
// /api/v1/health is what API clients use so they only have to know
// one base path.
func Module(pinger store.Pinger, service, version string) module.Module {
	h := NewHandler(Deps{Pinger: pinger, Service: service, Version: version})
	return module.Module{
		Name: "health",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/health", h).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/health", h).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}
