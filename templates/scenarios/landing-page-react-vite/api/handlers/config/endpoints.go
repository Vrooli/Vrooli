package config

import (
	"landing-page-react-vite-api/internal/module"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"
)

// Endpoints describes the landing-config module's Connect-RPC surface for codegen.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "landing_config_get", Path: landingconnect.LandingConfigServiceGetLandingConfigProcedure, Method: "POST",
		Summary: "Get landing config", Description: "Returns the aggregated public landing payload for a variant (or a selected one).", Category: "config",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"variant_slug": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"*": "aggregated landing payload"}},
	},
}
