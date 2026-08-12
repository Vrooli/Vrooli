package health

import (
	"net/http"

	"proto-health/internal/database"
	"proto-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
)

// Module returns the health domain's contribution to the API: the
// /health and /api/v1/health routes plus the descriptor the codegen
// reads. The single handler is mounted at both paths — /health is the
// probe convention infrastructure (LB, Kubernetes) reaches for;
// /api/v1/health is what API clients use so they only have to know
// one base path.
func Module(pinger database.Pinger, service, version string, sources ...*descriptorimage.Source) module.Module {
	var source *descriptorimage.Source
	if len(sources) > 0 {
		source = sources[0]
	}
	h := NewHandler(Deps{Pinger: pinger, Service: service, Version: version, DescriptorSource: source})
	return module.Module{
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
