package discovery

import (
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"github.com/ecosystem-manager/api/internal/module"
	"github.com/ecosystem-manager/api/pkg/prompts"
	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ecosystem-manager/v1/discovery/discovery_v1connect"
)

// Module returns the discovery domain's contribution to the API: the Connect
// DiscoveryService handler mounted at its generated procedure paths.
func Module(assembler *prompts.Assembler) module.Module {
	connectPath, connectHandler := discoveryconnect.NewDiscoveryServiceHandler(NewConnectHandler(Deps{Assembler: assembler}))
	return module.Module{
		Name: "discovery",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Endpoints describes the discovery domain's Connect procedures for the
// codegen pipeline (.vrooli/endpoints.json via `make endpoints`).
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "discovery_list_resources",
		Path:        discoveryconnect.DiscoveryServiceListResourcesProcedure,
		Method:      "POST",
		Summary:     "List discovered resources",
		Description: "Enumerates every local resource available to compose into tasks.",
		Category:    "discovery",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"refresh": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"resources": "array<Resource>", "count": "int32"}},
		Examples: []module.Example{{
			Name: "List resources",
			Curl: "curl http://localhost:${API_PORT}/vrooli.ecosystem_manager.v1.discovery.DiscoveryService/ListResources -H 'Content-Type: application/json' -d '{}'",
		}},
	},
	{
		ID:          "discovery_list_scenarios",
		Path:        discoveryconnect.DiscoveryServiceListScenariosProcedure,
		Method:      "POST",
		Summary:     "List discovered scenarios",
		Description: "Enumerates every scenario available to compose into tasks.",
		Category:    "discovery",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"refresh": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenarios": "array<Scenario>", "count": "int32"}},
		Examples: []module.Example{{
			Name: "List scenarios",
			Curl: "curl http://localhost:${API_PORT}/vrooli.ecosystem_manager.v1.discovery.DiscoveryService/ListScenarios -H 'Content-Type: application/json' -d '{}'",
		}},
	},
	{
		ID:          "discovery_get_resource",
		Path:        discoveryconnect.DiscoveryServiceGetResourceProcedure,
		Method:      "POST",
		Summary:     "Get a resource by name",
		Description: "Returns one discovered resource by name.",
		Category:    "discovery",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"name": "string", "refresh": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"resource": "Resource"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No resource with that name"}},
	},
	{
		ID:          "discovery_get_scenario",
		Path:        discoveryconnect.DiscoveryServiceGetScenarioProcedure,
		Method:      "POST",
		Summary:     "Get a scenario by name",
		Description: "Returns one discovered scenario by name.",
		Category:    "discovery",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"name": "string", "refresh": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "Scenario"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No scenario with that name"}},
	},
	{
		ID:          "discovery_list_operations",
		Path:        discoveryconnect.DiscoveryServiceListOperationsProcedure,
		Method:      "POST",
		Summary:     "List task operations",
		Description: "Returns the configured task operation types.",
		Category:    "discovery",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"operations": "array<Operation>"}},
	},
	{
		ID:          "discovery_list_categories",
		Path:        discoveryconnect.DiscoveryServiceListCategoriesProcedure,
		Method:      "POST",
		Summary:     "List task categories",
		Description: "Returns the resource/scenario category groupings the create-task form offers.",
		Category:    "discovery",
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"resource_categories": "map<string,string>", "scenario_categories": "map<string,string>"}},
	},
}
