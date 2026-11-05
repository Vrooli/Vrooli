package safety

import (
	"image-tools/internal/module"
	internalsafety "image-tools/internal/safety"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	safetyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety/safety_v1connect"
)

// Module returns the safety domain's contribution: the SafetyService discovery
// handler (Connect-RPC, GetPolicy) reporting the resolved Responsible-Use policy
// for the running deployment tier. Enforcement itself lives on the AI submit
// edge (handlers/ai), which holds the gate; this surface is pure discovery so
// the UI/CLI can show what is enforced.
func Module(tier internalsafety.Tier) module.Module {
	connectPath, connectHandler := safetyconnect.NewSafetyServiceHandler(NewConnectHandler(tier))
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
