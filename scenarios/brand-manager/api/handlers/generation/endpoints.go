package generation

import (
	"brand-manager/internal/module"

	generationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/generation/generation_v1connect"
)

// Endpoints is the machine-readable description of the generation module's
// public surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in generation.proto breaks this file at compile
// time. The global parity test in modules/registry_test.go asserts every rpc
// has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "generation_provider_status",
		Path:        generationconnect.GenerationServiceGetProviderStatusProcedure,
		Method:      "POST",
		Summary:     "Report AI provider availability",
		Description: "Returns whether at least one AI provider in the chain is reachable, plus per-provider status (Ollama first, then OpenRouter).",
		Category:    "generation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"available": "bool (at least one provider reachable)",
			"providers": "array<ProviderStatus>",
		}},
		Examples: []module.Example{{Name: "Provider status", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.generation.GenerationService/GetProviderStatus -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "generation_elements",
		Path:        generationconnect.GenerationServiceGenerateBrandElementsProcedure,
		Method:      "POST",
		Summary:     "Generate brand text facets",
		Description: "Generates the requested facets (colors, typography, voice) via the AI chain and merges them onto the brand (partial-merge; each apply bumps the version). Per-element provider/parse failures are reported per-element, not as a call error.",
		Category:    "generation",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id": "string (required, must exist)",
			"elements": "array<string> (subset of colors, typography, voice)",
			"model":    "string (optional provider model/role override)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"results":       "array<ElementResult>",
			"applied":       "array<string>",
			"provider":      "string",
			"model":         "string",
			"brand_version": "int32 (0 when nothing applied)",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing brand_id or empty elements"},
			{Status: 404, Code: "not_found", Description: "No brand with that id exists"},
			{Status: 503, Code: "unavailable", Description: "No AI provider is currently reachable"},
			{Status: 500, Code: "internal", Description: "Applying generated facets failed"},
		},
		Examples: []module.Example{{Name: "Generate colors", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.generation.GenerationService/GenerateBrandElements -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"elements\":[\"colors\"]}'"}},
	},
	{
		ID:          "generation_image",
		Path:        generationconnect.GenerationServiceGenerateBrandImageProcedure,
		Method:      "POST",
		Summary:     "Generate a brand image",
		Description: "Generates a logo or favicon via the AI chain and stores it as a brand asset (upsert by brand_id + derived filename, so regenerating replaces the bytes in place).",
		Category:    "generation",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id": "string (required, must exist)",
			"type":     "string (logo | favicon)",
			"model":    "string (optional provider model override)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"asset_id":  "string",
			"filename":  "string",
			"mime_type": "string",
			"size":      "int64",
			"provider":  "string",
			"model":     "string",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing brand_id or type not in {logo, favicon}"},
			{Status: 404, Code: "not_found", Description: "No brand with that id exists"},
			{Status: 503, Code: "unavailable", Description: "No AI provider is currently reachable"},
			{Status: 500, Code: "internal", Description: "Image generation or asset write failed"},
		},
		Examples: []module.Example{{Name: "Generate logo", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.generation.GenerationService/GenerateBrandImage -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"type\":\"logo\"}'"}},
	},
}
