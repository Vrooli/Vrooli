package catalog

import (
	"image-tools/internal/module"

	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/catalog/catalog_v1connect"
)

// Endpoints describes the catalog module's surface. Connect procedure constants
// are referenced from the generated client so a renamed rpc breaks this file at
// compile time; the global parity test asserts every rpc has exactly one entry.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "catalog_get_operation_catalog",
		Path:        catalogconnect.CatalogServiceGetOperationCatalogProcedure,
		Method:      "POST",
		Summary:     "Publish the operation catalog",
		Description: "Returns this scenario's operations in the shared common.v1 media-operation contract: typed ports, requirement facts, consent weight, ordered backends, and parameter schema. A workflow builder in another scenario reads this instead of importing image-tools.",
		Category:    "catalog",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"family": "string (optional: generate|edit|enhance|restore|extract|compose)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"contract_version": "string",
				"scenario":         "string",
				"operations":       "array<common.v1.MediaOperation>",
			},
		},
		Examples: []module.Example{
			{Name: "Whole catalog", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.catalog.CatalogService/GetOperationCatalog -H 'Content-Type: application/json' -d '{}'"},
			{Name: "Only generation operations", Curl: "curl http://localhost:${API_PORT}/vrooli.image_tools.v1.catalog.CatalogService/GetOperationCatalog -H 'Content-Type: application/json' -d '{\"family\":\"generate\"}'"},
		},
	},
}
