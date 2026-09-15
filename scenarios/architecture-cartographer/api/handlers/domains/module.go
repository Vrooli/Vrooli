package domains

import (
	"architecture-cartographer/internal/domains"
	"architecture-cartographer/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains/domains_v1connect"
)

// Module returns the domains domain's contribution to the API router: the
// DomainsService Connect-RPC routes plus the descriptor the codegen reads.
func Module(svc domains.Service) module.Module {
	h := NewHandler(svc)
	pattern, connectHandler := domains_v1connect.NewDomainsServiceHandler(h)
	return module.Module{
		Name: "domains",
		Mount: func(r *mux.Router) {
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the domains domain is stateless. The domain map is
// derived on demand from the target scenario's on-disk sources and is not
// persisted.
func Schema() string { return "" }
