package safety

import (
	"image-tools/internal/module"
	internalsafety "image-tools/internal/safety"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	safetyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety/safety_v1connect"
)

// Module returns the safety domain's contribution: the SafetyService discovery
// handler (Connect-RPC, GetPolicy) reporting the RESOLVED Responsible-Use policy
// (preset + per-control overrides) for the running deployment. Enforcement
// itself lives on the AI submit edge (handlers/ai), which holds the same policy
// through its gate; this surface is pure discovery so the UI/CLI can show exactly
// what is enforced.
func Module(policy internalsafety.Policy) module.Module {
	connectPath, connectHandler := safetyconnect.NewSafetyServiceHandler(NewConnectHandler(policy))
	return module.Module{
		Name: "safety",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the consent-log SQL contribution.
func Schema() string { return internalsafety.Schema() }
