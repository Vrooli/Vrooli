package pairing

import (
	"log"

	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/nodeauth"
	internalpairing "vrooli-bridge/internal/pairing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	pairingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing/pairing_v1connect"
)

// Module returns the pairing domain's contribution to the API: the generated
// Connect PairingService handler. The pairing service (with its NodeRegistrar
// wired to the registry domain) and the control-plane public key are built in
// main.go and passed in, because the same pairing repository is also shared
// with the nodeauth verifier and the registry atomic-revoke.
func Module(svc *internalpairing.Service, controlPlanePublicKey string, defaultScopes []string, verifier *nodeauth.Verifier, logger *log.Logger) module.Module {
	path, handler := pairingconnect.NewPairingServiceHandler(NewConnectHandler(Deps{
		Service:               svc,
		ControlPlanePublicKey: controlPlanePublicKey,
		DefaultScopes:         defaultScopes,
		Logger:                logger,
		NodeVerifier:          verifier,
	}))
	return module.Module{
		Name: "pairing",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the pairing domain schema for the modules registry.
func Schema() string { return internalpairing.Schema() }
