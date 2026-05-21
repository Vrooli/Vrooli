package manifest

import (
	"architecture-cartographer/internal/manifest"
	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/manifest/manifest_v1connect"
)

// Module returns the manifest domain's contribution to the API
// router: the ManifestService Connect-RPC routes plus the descriptor
// the codegen reads.
func Module(svc manifest.Service) module.Module {
	h := NewHandler(svc)
	pattern, connectHandler := manifest_v1connect.NewManifestServiceHandler(h)
	return module.Module{
		Name: "manifest",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the manifest domain's SQL contribution so the
// modules registry stays a uniform "every domain has a Schema()"
// pattern.
func Schema() string { return manifest.Schema() }
