package discovery

import (
	"log"

	"data-backup-manager/internal/discovery"
	"data-backup-manager/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/discovery/discovery_v1connect"
)

// Module returns the discovery domain's contribution to the API: the generated
// DiscoveryService Connect-RPC handler. Unlike the leaf domains, discovery
// composes cross-domain seams (the target/destination catalogs and the
// protected-path provider), so the fully-wired service is built in the
// composition root (main.go) and passed in — mirroring restores/runs.
func Module(svc discovery.Service, logger *log.Logger) module.Module {
	connectPath, connectHandler := discoveryconnect.NewDiscoveryServiceHandler(NewConnectHandler(Deps{
		Service: svc,
		Logger:  logger,
	}))
	return module.Module{
		Name: "discovery",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the discovery domain SQL (the dismissals table) so the
// modules registry collects endpoints and schema from one symbol per handler
// package.
func Schema() string { return discovery.Schema() }
