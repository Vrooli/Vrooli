package domains

import (
	"architecture-cartographer/internal/domains"
	"github.com/vrooli/api-core/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains/domains_v1connect"
)

// Module returns the domains owner's Connect-RPC and scoped zone-classifier
// routes plus endpoint descriptors.
func Module(svc domains.Service, locator ...domains.ScenarioLocator) module.Module {
	h := NewHandler(svc, locator...)
	pattern, connectHandler := domains_v1connect.NewDomainsServiceHandler(h)
	return module.Module{
		Name: "domains",
		Mount: func(r *mux.Router) {
			r.HandleFunc(classifyZonesPath, h.ClassifyZones).Methods("POST")
			r.PathPrefix(pattern).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the domains domain is stateless. The domain map is
// derived on demand from the target scenario's on-disk sources and is not
// persisted.
func Schema() string { return "" }
