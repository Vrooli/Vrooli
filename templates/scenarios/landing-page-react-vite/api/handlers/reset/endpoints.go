package reset

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the reset module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "admin_reset_demo_data", Path: landingconnect.AdminResetServiceResetDemoDataProcedure, Method: "POST",
		Summary: "Reset demo data", Description: "TRUNCATEs and reseeds demo tables (admin, ENABLE_ADMIN_RESET-gated).", Category: "admin",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"reset": "bool", "timestamp": "string"}},
	},
}
