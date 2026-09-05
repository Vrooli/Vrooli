package health

import (
	"context"
	"net/http"

	"backdrop-studio/internal/buildinfo"
	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/database"
	"backdrop-studio/internal/module"

	"github.com/gorilla/mux"
)

// Module returns the health domain's contribution to the API: the
// /health and /api/v1/health routes plus the descriptor the codegen
// reads. The single handler is mounted at both paths — /health is the
// probe convention infrastructure (LB, Kubernetes) reaches for;
// /api/v1/health is what API clients use so they only have to know
// one base path.
// applied reports the catalog seed version the connected install has applied.
// It is a function parameter rather than a store dependency so the health
// handler keeps its one job — reporting — and stays testable without a
// database.
func Module(pinger database.Pinger, service, version string, applied func(context.Context) (int, error)) module.Module {
	seedVersion, err := catalog.SeedVersion()
	if err != nil {
		// The embedded seed failing to load is a build defect, not a runtime
		// condition; reporting version 0 makes the freshness endpoint say so
		// rather than claiming a version the binary does not carry.
		seedVersion = 0
	}
	deps := Deps{Pinger: pinger, Service: service, Version: version, Fingerprint: buildinfo.Fingerprint(), SeedVersion: seedVersion}
	h := NewHandler(deps)
	build := NewBuildHandler(deps, applied)
	return module.Module{
		Name: "health",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/health", h).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/health", h).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/build", build).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — health is stateless, no tables to own. The
// modules registry includes this re-export anyway so adding a stateful
// domain later is a uniform "create the file, return the SQL" pattern
// instead of "remember to also add a Schema() function."
func Schema() string { return "" }
