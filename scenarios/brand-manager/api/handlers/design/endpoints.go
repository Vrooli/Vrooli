package design

import (
	"brand-manager/internal/module"

	designconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/design/design_v1connect"
)

// Endpoints is the machine-readable description of the design module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in design.proto breaks this file at compile
// time. The global parity test in modules/registry_test.go asserts every rpc has
// exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "design_generate",
		Path:        designconnect.DesignServiceGenerateDesignLanguageProcedure,
		Method:      "POST",
		Summary:     "Render a brand as a DESIGN.md document",
		Description: "Reads a brand and renders a canonical DESIGN.md document — front matter, identity, a color-system table, typography, voice, derived visual patterns, and notes. Read-only: writes nothing; the markdown is returned in the response.",
		Category:    "design",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id": "string (required, brand must exist)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id": "string (echoes the request)",
			"markdown": "string (the complete DESIGN.md document)",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing brand_id"},
			{Status: 404, Code: "not_found", Description: "No brand matches the id"},
		},
		Examples: []module.Example{{Name: "Generate DESIGN.md", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.design.DesignService/GenerateDesignLanguage -H 'Content-Type: application/json' -d '{\"brand_id\":\"brand-1\"}'"}},
	},
}
