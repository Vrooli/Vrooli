// Package flows wires the Connect-RPC FlowsService for flow discovery,
// inspection, and lifecycle (CreateFlow/ValidateFlow/ExplainFlow). The
// CLI's `flows new|list|show|validate|explain` subcommands all route
// through these RPCs after the cutover.
package flows

import (
	"log"

	"flow-verifier/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	flowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows/flows_v1connect"
)

// Module returns the flows domain's Connect-RPC contribution. svc may
// be nil for tests exercising the legacy root-passed branch only; the
// scenario-aware branches return Internal in that case.
func Module(svc ScenariosService) module.Module {
	return ModuleWithLogger(svc, log.Default())
}

// ModuleWithLogger is the test-friendly variant.
func ModuleWithLogger(svc ScenariosService, logger *log.Logger) module.Module {
	path, handler := flowsconnect.NewFlowsServiceHandler(NewConnectHandler(Deps{Scenarios: svc, Logger: logger}))
	return module.Module{
		Name: "flows",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — flow inventory is filesystem-truth, no tables.
func Schema() string { return "" }
