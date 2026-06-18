package identity

import (
	"log"

	"device-sync-hub/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	internalidentity "device-sync-hub/internal/identity"

	identityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/identity/identity_v1connect"
)

// Module returns the identity domain's contribution to the API: the generated
// Connect-RPC IdentityService handler. resolver resolves scenario-authenticator's
// API URL by name (api-core/discovery in production); the forwarder relays
// Login/Register to it. No persistence — this domain owns no tables.
func Module(resolver internalidentity.URLResolver, logger *log.Logger) module.Module {
	fwd := internalidentity.NewForwarder(internalidentity.Config{Resolver: resolver})
	path, handler := identityconnect.NewIdentityServiceHandler(NewConnectHandler(Deps{
		Forwarder: fwd,
		Logger:    logger,
	}))
	return module.Module{
		Name: "identity",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — identity is a stateless forwarder with no tables. The
// modules registry includes this re-export for uniformity with stateful
// domains.
func Schema() string { return "" }
