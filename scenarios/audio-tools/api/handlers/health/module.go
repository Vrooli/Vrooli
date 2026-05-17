package health

import (
	"net/http"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/database"
	"audio-tools/internal/modulekit"

	"github.com/gorilla/mux"
)

// Module returns the health domain's contribution to the API: the
// /health and /api/v1/health routes plus the descriptor the codegen
// reads. The single handler is mounted at both paths — /health is the
// probe convention infrastructure (LB, Kubernetes) reaches for;
// /api/v1/health is what API clients use so they only have to know
// one base path.
//
// `reg` may be nil — the providers dependency is only registered when
// a capabilities.Registry is supplied. This keeps the test path
// (and other minimal embeds) free of the registry dep.
func Module(pinger database.Pinger, reg *capabilities.Registry, service, version string) modulekit.Module {
	h := NewHandler(Deps{Pinger: pinger, Registry: reg, Service: service, Version: version})
	return modulekit.Module{
		Name: "health",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/health", h).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/health", h).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — health is stateless, no tables to own. The
// modules registry includes this re-export anyway so adding a stateful
// domain later is a uniform "create the file, return the SQL" pattern
// instead of "remember to also add a Schema() function."
func Schema() string { return "" }
