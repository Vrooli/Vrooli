package validation

import (
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

var ScenarioValidationEndpoints = []module.EndpointDescriptor{
	{
		ID:          "scenario_validation_validate",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate template standing",
		Description: "Statically validates a scenario's template provenance, orientation visibility, drift, version lag, and inherited-debt standing.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "include_execution": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "status": "scenario_validation.v1.ValidationStatus", "assessment": "common.v1.MaturityAssessment"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Missing target or invalid target path"}},
	},
	{
		ID:          "validation_describe_provider",
		Path:        scenariovalidationconnect.ScenarioValidationServiceDescribeProviderProcedure,
		Method:      "POST",
		Summary:     "Describe this provider's identity and contract",
		Description: "Reports provider identity, backed phase, maturity spec version, contract, build provenance, and capabilities. Inspects no target, so readiness consumers can confirm this provider is live and current without paying for a full validation run.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"provider":     "string",
			"phase":        "string",
			"spec_version": "string",
			"contract":     "string",
			"build":        "scenario_validation.v1.ProviderBuild",
			"capabilities": "scenario_validation.v1.ProviderCapabilities",
		}},
		Examples: []module.Example{{Name: "Describe provider", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/DescribeProvider -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "scenario_validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic template fixes",
		Description: "Dry-runs deterministic template standing repairs such as adopted provenance stamping.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Missing target or invalid fix request"}},
	},
	{
		ID:          "scenario_validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic template fixes",
		Description: "Applies deterministic template standing repairs selected explicitly by the caller.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Missing target or invalid fix request"}},
	},
}
