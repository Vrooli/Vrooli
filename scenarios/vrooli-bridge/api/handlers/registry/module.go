package registry

import (
	"log"

	"vrooli-bridge/internal/module"
	internalregistry "vrooli-bridge/internal/registry"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"
)

// Module returns the registry domain's contribution to the API: the generated
// Connect-RPC NodeRegistryService handler. presence (optional) backs the live
// online/offline overlay on the read path; nil disables the overlay (every node
// reads offline) without changing the stored node data — the Phase-1 presence
// step threads the real hub in.
func Module(svc internalregistry.Service, presence Presence, credentials CredentialRevoker, disconnect Disconnector, logger *log.Logger) module.Module {
	path, handler := registryconnect.NewNodeRegistryServiceHandler(NewConnectHandler(Deps{
		Service:     svc,
		Presence:    presence,
		Credentials: credentials,
		Disconnect:  disconnect,
		Logger:      logger,
	}))
	return module.Module{
		Name: "registry",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the registry domain schema so the modules registry collects
// endpoint descriptors and schema from one handler package.
func Schema() string { return internalregistry.Schema() }
