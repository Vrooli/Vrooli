package safety

import (
	"net/http"

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
// what is enforced. The policy-only constructor leaves the feedback verbs
// unimplemented; production uses ModuleWithDeps.
func Module(policy internalsafety.Policy, health ...func() internalsafety.ReviewHealth) module.Module {
	return ModuleWithDeps(Deps{Policy: policy, Health: firstHealth(health)})
}

// ModuleWithDeps returns the full safety module, including the labelled
// refusal/flag corpus that sits on top of the reviewer evidence.
func ModuleWithDeps(d Deps) module.Module {
	handler := NewConnectHandlerWithDeps(d)
	connectPath, serviceHandler := safetyconnect.NewSafetyServiceHandler(handler)
	return module.Module{
		Name: "safety",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: serviceHandler})
			r.HandleFunc("/api/v1/safety/quarantine/{id}", handler.quarantineImage).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

func firstHealth(health []func() internalsafety.ReviewHealth) func() internalsafety.ReviewHealth {
	if len(health) == 0 {
		return nil
	}
	return health[0]
}

// Schema returns the consent-log and reviewer-feedback SQL contribution.
func Schema() string { return internalsafety.Schema() }
