package discovery

import (
	"brand-manager/internal/module"

	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/discovery/discovery_v1connect"
)

// Endpoints is the machine-readable description of the discovery module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in discovery.proto breaks this file at compile
// time. The global parity test in modules/registry_test.go asserts every rpc has
// exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "discovery_scan",
		Path:        discoveryconnect.DiscoveryServiceDiscoverScenarioProcedure,
		Method:      "POST",
		Summary:     "Scan a scenario for branding state",
		Description: "Reads a scenario's .vrooli/service.json, .vrooli/branding.json, ui/public/manifest.json, theme CSS, and ui/public assets, infers a draft brand from whatever it finds, and reports each source with a confidence score. Read-only: creates nothing.",
		Category:    "discovery",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"scenario_name": "string (required, scenario directory must exist)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"scenario":    "string",
			"sources":     "array<DiscoverySource>",
			"draft_brand": "DraftBrand (null when no sources matched)",
			"confidence":  "double (0.0-1.0)",
			"suggestions": "array<string>",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario_name"},
			{Status: 404, Code: "not_found", Description: "The scenario directory does not exist"},
		},
		Examples: []module.Example{{Name: "Discover scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.discovery.DiscoveryService/DiscoverScenario -H 'Content-Type: application/json' -d '{\"scenario_name\":\"web-console\"}'"}},
	},
	{
		ID:          "discovery_import",
		Path:        discoveryconnect.DiscoveryServiceImportBrandProcedure,
		Method:      "POST",
		Summary:     "Import a brand from discovered state",
		Description: "Re-scans a scenario and persists the inferred draft as a new brand through the brands domain. A scenario with no branding state yields FAILED_PRECONDITION (nothing to import).",
		Category:    "discovery",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"scenario_name": "string (required, scenario directory must exist)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id":      "string",
			"brand_name":    "string",
			"brand_version": "int32 (always 1)",
			"sources":       "array<DiscoverySource>",
			"confidence":    "double (0.0-1.0)",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario_name"},
			{Status: 404, Code: "not_found", Description: "The scenario directory does not exist"},
			{Status: 412, Code: "failed_precondition", Description: "No branding state found to import"},
			{Status: 500, Code: "internal", Description: "Creating the brand failed"},
		},
		Examples: []module.Example{{Name: "Import brand", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.discovery.DiscoveryService/ImportBrand -H 'Content-Type: application/json' -d '{\"scenario_name\":\"web-console\"}'"}},
	},
}
