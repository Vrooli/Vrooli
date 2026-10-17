package health

import (
	"context"
	"net/http"

	"notification-hub/internal/database"
	"notification-hub/internal/module"

	"github.com/gorilla/mux"
)

// Module returns the health domain's contribution to the API: the
// /health and /api/v1/health routes plus the descriptor the codegen
// reads. The single handler is mounted at both paths — /health is the
// probe convention infrastructure (LB, Kubernetes) reaches for;
// /api/v1/health is what API clients use so they only have to know
// one base path.

func Module(pinger database.Pinger, service, version string, postures ...string) module.Module {
	return ModuleWithIdentity(pinger, nil, service, version, postures...)
}

// ModuleWithIdentity additionally reports authenticator reachability. The
// optional seam keeps isolated handler tests focused on the database while
// production wires the shared owneridentity client.
func ModuleWithIdentity(pinger database.Pinger, identityChecker interface{ Reachable(context.Context) error }, service, version string, postures ...string) module.Module {
	posture := "personal"
	if len(postures) > 0 && postures[0] != "" {
		posture = postures[0]
	}
	h := NewHandler(Deps{Pinger: pinger, Identity: identityChecker, Service: service, Version: version, TrustPosture: posture})
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
