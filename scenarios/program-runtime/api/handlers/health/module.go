package health

import (
	"net/http"
	"time"

	"program-runtime/internal/database"
	"program-runtime/internal/module"

	"github.com/gorilla/mux"
)

type DescriptorMetadata func() (digest string, generation uint64, loadedAt, artifactMTime time.Time, reloadErr error)

// Module returns the health domain's contribution to the API: the
// /health and /api/v1/health routes plus the descriptor the codegen
// reads. The single handler is mounted at both paths — /health is the
// probe convention infrastructure (LB, Kubernetes) reaches for;
// /api/v1/health is what API clients use so they only have to know
// one base path.
func Module(pinger database.Pinger, service, version string, skippedManifestCount ...func() int) module.Module {
	var skipped func() int
	if len(skippedManifestCount) > 0 {
		skipped = skippedManifestCount[0]
	}
	return moduleWithMetadata(pinger, service, version, skipped, nil)
}

func ModuleWithDescriptor(pinger database.Pinger, service, version string, skippedManifestCount func() int, descriptorMetadata DescriptorMetadata) module.Module {
	return moduleWithMetadata(pinger, service, version, skippedManifestCount, descriptorMetadata)
}

func moduleWithMetadata(pinger database.Pinger, service, version string, skipped func() int, metadata DescriptorMetadata) module.Module {
	h := NewHandler(Deps{Pinger: pinger, Service: service, Version: version, SkippedManifestCount: skipped, DescriptorMetadata: metadata})
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
