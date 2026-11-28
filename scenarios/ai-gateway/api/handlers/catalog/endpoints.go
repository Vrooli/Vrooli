package catalog

import (
	"ai-gateway/internal/module"

	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/catalog/catalog_v1connect"
)

var Endpoints = []module.EndpointDescriptor{{
	ID:          "catalog_get_capability_catalog",
	Path:        catalogconnect.CatalogServiceGetCapabilityCatalogProcedure,
	Method:      "POST",
	Summary:     "Get AI Gateway capability catalog",
	Description: "Read-only discovery projection of native profiles, request kinds, roles, and providers; it does not confer authority or drive dispatch.",
	Category:    "catalog",
	Request:     &module.Schema{Type: "GetCapabilityCatalogRequest"},
	Response:    &module.Schema{Type: "common.v1.AIGatewayCapabilityCatalog"},
}}
