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
		Summary:     "Report text AI provider availability",
		Description: "Returns whether at least one text AI provider in the chain is reachable, plus per-provider status (Ollama first, then OpenRouter). Text facets only — image readiness is GetImageBackendStatus.",
		Category:    "generation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"available": "bool (at least one text provider reachable)",
			"providers": "array<ProviderStatus>",
		}},
		Examples: []module.Example{{Name: "Provider status", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.generation.GenerationService/GetProviderStatus -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "generation_image_backend_status",
		Path:        generationconnect.GenerationServiceGetImageBackendStatusProcedure,
		Method:      "POST",
		Summary:     "Report image-tools readiness",
		Description: "Returns whether image-tools is reachable plus per-operation readiness (generate/edit/remove_background): the model image-tools would select, the tier (local-gpu/local-cpu/byok-cloud), and an actionable hint when an operation is not ready.",
		Category:    "generation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"available":  "bool (image-tools reachable)",
			"detail":     "string (set when unavailable)",
			"operations": "array<ImageOperationStatus>",
		}},
		Examples: []module.Example{{Name: "Image backend status", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.generation.GenerationService/GetImageBackendStatus -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "generation_elements",
		Path:        generationconnect.GenerationServiceGenerateBrandElementsProcedure,
		Method:      "POST",
		Summary:     "Generate brand text facets",
		Description: "Generates the requested facets (colors, typography, voice) via the text AI chain and merges them onto the brand (partial-merge; each apply bumps the version). Per-element provider/parse failures are reported per-element, not as a call error.",
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
			{Status: 503, Code: "unavailable", Description: "No text AI provider is currently reachable"},
			{Status: 500, Code: "internal", Description: "Applying generated facets failed"},
		},
		Examples: []module.Example{{Name: "Generate colors", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.generation.GenerationService/GenerateBrandElements -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"elements\":[\"colors\"]}'"}},
	},
	{
		ID:          "generation_image",
		Path:        generationconnect.GenerationServiceGenerateBrandImageProcedure,
		Method:      "POST",
		Summary:     "Generate a brand image",
		Description: "Generates a logo or favicon through image-tools (text_to_image) from a brand-aware prompt and stores it as a brand asset under a unique filename. Promotes to the canonical logo.png/favicon.png when set_canonical is true or the brand has no canonical yet.",
		Category:    "generation",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id":       "string (required, must exist)",
			"type":           "string (logo | favicon)",
			"model_override": "string (optional image-tools model id)",
			"allow_byok":     "bool (permit metered BYOK cloud fallback)",
			"seed":           "int64 (optional; 0 = random)",
			"set_canonical":  "bool (also write canonical asset)",
		}},
		Response: imageAssetSchema(),
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing brand_id or type not in {logo, favicon}"},
			{Status: 404, Code: "not_found", Description: "No brand with that id exists"},
			{Status: 503, Code: "unavailable", Description: "image-tools is not reachable"},
			{Status: 412, Code: "failed_precondition", Description: "image-tools model/backend not installed or BYOK key missing"},
			{Status: 500, Code: "internal", Description: "Image job failed or asset write failed"},
		},
		Examples: []module.Example{{Name: "Generate logo", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.generation.GenerationService/GenerateBrandImage -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"type\":\"logo\"}'"}},
	},
	{
		ID:          "generation_edit_image",
		Path:        generationconnect.GenerationServiceEditBrandImageProcedure,
		Method:      "POST",
		Summary:     "Edit a brand image by instruction",
		Description: "Edits an existing brand image through image-tools (edit_instruct) using a natural-language instruction and stores the edited result as a new brand asset.",
		Category:    "generation",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id":        "string (required, must exist)",
			"source_asset_id": "string (required, existing brand asset)",
			"instruction":     "string (required edit instruction)",
			"model_override":  "string (optional image-tools model id)",
			"allow_byok":      "bool (permit metered BYOK cloud fallback)",
			"seed":            "int64 (optional; 0 = random)",
			"set_canonical":   "bool (also write canonical logo.png)",
		}},
		Response: imageAssetSchema(),
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing brand_id, source_asset_id, or instruction"},
			{Status: 404, Code: "not_found", Description: "Brand or source asset not found"},
			{Status: 503, Code: "unavailable", Description: "image-tools is not reachable"},
			{Status: 412, Code: "failed_precondition", Description: "image-tools model/backend not ready"},
			{Status: 500, Code: "internal", Description: "Image job failed or asset write failed"},
		},
		Examples: []module.Example{{Name: "Edit logo", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.generation.GenerationService/EditBrandImage -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"source_asset_id\":\"asset1\",\"instruction\":\"make the background navy\"}'"}},
	},
	{
		ID:          "generation_remove_background",
		Path:        generationconnect.GenerationServiceRemoveBrandImageBackgroundProcedure,
		Method:      "POST",
		Summary:     "Remove a brand image background",
		Description: "Isolates the mark in an existing brand image through image-tools (background_removal) and stores the transparent cutout as a new brand asset.",
		Category:    "generation",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id":        "string (required, must exist)",
			"source_asset_id": "string (required, existing brand asset)",
			"model_override":  "string (optional image-tools model id)",
			"allow_byok":      "bool (permit metered BYOK cloud fallback)",
			"set_canonical":   "bool (also write canonical logo-transparent.png)",
		}},
		Response: imageAssetSchema(),
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing brand_id or source_asset_id"},
			{Status: 404, Code: "not_found", Description: "Brand or source asset not found"},
			{Status: 503, Code: "unavailable", Description: "image-tools is not reachable"},
			{Status: 412, Code: "failed_precondition", Description: "image-tools model/backend not ready"},
			{Status: 500, Code: "internal", Description: "Image job failed or asset write failed"},
		},
		Examples: []module.Example{{Name: "Remove background", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.generation.GenerationService/RemoveBrandImageBackground -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"source_asset_id\":\"asset1\"}'"}},
	},
	{
		ID:          "generation_derive_icons",
		Path:        generationconnect.GenerationServiceDeriveBrandIconsProcedure,
		Method:      "POST",
		Summary:     "Derive a platform icon set",
		Description: "Produces a deterministic set of platform icon variants (transparent favicons + solid-background Apple-touch/maskable launcher icons) from a source asset using image-tools' deterministic resize/flatten. Idempotent: re-deriving overwrites byte-identically. When no include flag is set, all variant families are produced.",
		Category:    "generation",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id":            "string (required, must exist)",
			"source_asset_id":     "string (required, existing brand asset)",
			"include_maskable":    "bool (emit maskable 192/512)",
			"include_apple_touch": "bool (emit apple-touch 180)",
			"include_favicon":     "bool (emit favicon 16/32/196)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"icons":    "array<BrandImageAsset>",
			"warnings": "array<string>",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing brand_id or source_asset_id"},
			{Status: 404, Code: "not_found", Description: "Brand or source asset not found"},
			{Status: 503, Code: "unavailable", Description: "image-tools is not reachable"},
			{Status: 500, Code: "internal", Description: "A derivation op or asset write failed"},
		},
		Examples: []module.Example{{Name: "Derive icons", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.generation.GenerationService/DeriveBrandIcons -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"source_asset_id\":\"asset1\"}'"}},
	},
}

// imageAssetSchema is the shared BrandImageAsset response shape for the image
// generation/edit/removal RPCs.
func imageAssetSchema() *module.Schema {
	return &module.Schema{Type: "object", Properties: map[string]string{
		"brand_id":  "string",
		"asset_id":  "string",
		"kind":      "string (logo | favicon | logo-transparent | icon variant)",
		"filename":  "string",
		"mime_type": "string",
		"size":      "int64",
		"model_id":  "string (image-tools model; empty for deterministic)",
		"tier":      "string (local-gpu | local-cpu | byok-cloud | deterministic)",
		"canonical": "bool (also written as canonical for its kind)",
		"warnings":  "array<string>",
	}}
}
