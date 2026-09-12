package targets

import (
	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets/targets_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the target catalog's Connect-RPC surface. Procedure
// constants come from generated code so schema/service renames fail here at
// compile time instead of silently drifting from the endpoint registry.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "targets_list", Path: targetsconnect.TargetCatalogServiceListProcedure, Method: "POST",
		Summary: "List terminal locations", Description: "Returns local and remote locations with safe readiness metadata.", Category: "targets",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"state": "CatalogState", "targets": "[]Target"}},
	},
	{
		ID: "targets_get", Path: targetsconnect.TargetCatalogServiceGetProcedure, Method: "POST",
		Summary: "Get a terminal location", Description: "Returns one safe target projection by catalog ID.", Category: "targets",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"target": "Target"}},
		Errors:   []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Target catalog ID not found"}},
	},
	{
		ID: "targets_doctor", Path: targetsconnect.TargetCatalogServiceDoctorProcedure, Method: "POST",
		Summary: "Diagnose a terminal location", Description: "Returns readiness facts and a first recovery action for one target.", Category: "targets",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"target": "Target", "summary": "string"}},
		Errors:   []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "Target catalog ID not found"}},
	},
}
