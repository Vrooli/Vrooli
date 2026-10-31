package apply

import (
	"brand-manager/internal/module"

	applyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/apply/apply_v1connect"
)

// Endpoints is the machine-readable description of the apply module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in apply.proto breaks this file at compile time.
// The global parity test in modules/registry_test.go asserts every rpc has
// exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "apply_preview",
		Path:        applyconnect.ApplyServicePreviewApplyProcedure,
		Method:      "POST",
		Summary:     "Preview applying a brand to a scenario",
		Description: "Computes exactly which files applying a brand to a scenario would write (CSS, manifest.json, copied logo/favicon), WITHOUT touching the filesystem or recording an assignment.",
		Category:    "apply",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id":      "string (required, must exist)",
			"scenario_name": "string (required, scenario directory must exist)",
			"elements":      "array<string> (subset of colors, typography, identity, favicon, logo; empty = all)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"scenario":      "string",
			"brand_version": "int32",
			"dry_run":       "bool (always true)",
			"applied":       "array<ApplyAction>",
			"skipped":       "array<SkipReason>",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing brand_id or scenario_name"},
			{Status: 404, Code: "not_found", Description: "No such brand, or the scenario directory does not exist"},
		},
		Examples: []module.Example{{Name: "Preview apply", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.apply.ApplyService/PreviewApply -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"scenario_name\":\"web-console\"}'"}},
	},
	{
		ID:          "apply_brand",
		Path:        applyconnect.ApplyServiceApplyBrandProcedure,
		Method:      "POST",
		Summary:     "Apply a brand to a scenario",
		Description: "Writes the requested brand elements into the scenario's source tree (CSS custom properties, a manifest.json merge, copied logo/favicon bytes) and records the brand↔scenario assignment. Convergent: re-applying overwrites the same managed files.",
		Category:    "apply",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id":      "string (required, must exist)",
			"scenario_name": "string (required, scenario directory must exist)",
			"elements":      "array<string> (subset of colors, typography, identity, favicon, logo; empty = all)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"scenario":      "string",
			"brand_version": "int32",
			"dry_run":       "bool (always false)",
			"applied":       "array<ApplyAction>",
			"skipped":       "array<SkipReason>",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing brand_id or scenario_name"},
			{Status: 404, Code: "not_found", Description: "No such brand, or the scenario directory does not exist"},
			{Status: 500, Code: "internal", Description: "Writing a file or recording the assignment failed"},
		},
		Examples: []module.Example{{Name: "Apply brand", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.apply.ApplyService/ApplyBrand -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"scenario_name\":\"web-console\",\"elements\":[\"colors\",\"logo\"]}'"}},
	},
}
