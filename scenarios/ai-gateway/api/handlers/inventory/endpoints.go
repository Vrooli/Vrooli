package inventory

import (
	"ai-gateway/internal/module"

	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory/inventory_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "inventory_list_provider_roles",
		Path:        inventoryconnect.InventoryServiceListProviderRolesProcedure,
		Method:      "POST",
		Summary:     "List provider role inventory",
		Description: "List provider role inventory from resource-owned policy command output.",
		Category:    "inventory",
		Request:     &module.Schema{Type: "ListProviderRolesRequest", Properties: map[string]string{"provider": "string"}},
		Response:    &module.Schema{Type: "ListProviderRolesResponse", Properties: map[string]string{"roles": "array<ProviderRole>", "warnings": "array<string>"}},
	},
	{
		ID:          "inventory_smoke_provider",
		Path:        inventoryconnect.InventoryServiceSmokeProviderProcedure,
		Method:      "POST",
		Summary:     "Smoke test provider inventory command",
		Description: "Runs a bounded provider policy command through the resource CLI seam and maps failures to typed inventory status.",
		Category:    "inventory",
		Request:     &module.Schema{Type: "SmokeProviderRequest", Properties: map[string]string{"provider": "string"}},
		Response:    &module.Schema{Type: "SmokeProviderResponse", Properties: map[string]string{"provider": "string", "status": "string", "code": "string", "message": "string", "warnings": "array<string>"}},
	},
}
