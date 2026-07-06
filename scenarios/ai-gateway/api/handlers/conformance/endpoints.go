package conformance

import (
	"ai-gateway/internal/module"

	conformanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance/conformance_v1connect"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "conformance_scan_scenario",
		Path:        conformanceconnect.ConformanceServiceScanScenarioProcedure,
		Method:      "POST",
		Summary:     "Scan a scenario for AI gateway conformance",
		Description: "Scans a scenario tree for unsafe AI/provider coupling, embedding metadata risks, and gateway adoption signals.",
		Category:    "conformance",
		Request:     &module.Schema{Type: "ScanScenarioRequest", Properties: map[string]string{"scenario": "string", "path": "string"}},
		Response:    &module.Schema{Type: "ScanScenarioResponse", Properties: map[string]string{"scenario": "string", "maturity_level": "string", "findings": "array<ConformanceFinding>", "recommendations": "array<string>"}},
	},
	{
		ID:          "conformance_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate AI conformance through Test Genie contract",
		Description: "Scans a target scenario for unsafe AI/provider coupling and returns a shared scenario-validation maturity assessment for Test Genie.",
		Category:    "conformance",
		Request:     &module.Schema{Type: "ValidateScenarioRequest", Properties: map[string]string{"scenario": "string", "path": "string", "include_execution": "bool"}},
		Response:    &module.Schema{Type: "ValidateScenarioResponse", Properties: map[string]string{"scenario": "string", "status": "scenario_validation.v1.ValidationStatus", "assessment": "common.v1.MaturityAssessment", "native_detail": "google.protobuf.Any", "metrics": "common.v1.ExecutionMetrics"}},
	},
	{
		ID:          "conformance_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview AI conformance migration guidance",
		Description: "Reports that deterministic source rewrites are not implemented and points callers at scan migration guidance.",
		Category:    "conformance",
		Request:     &module.Schema{Type: "FixRequest", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "FixResponse", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
	},
	{
		ID:          "conformance_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply AI conformance migration guidance",
		Description: "No-op apply endpoint reserved for future deterministic AI Gateway migrations.",
		Category:    "conformance",
		Request:     &module.Schema{Type: "FixRequest", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>"}},
		Response:    &module.Schema{Type: "FixResponse", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<scenario_validation.v1.FixCandidate>", "messages": "array<string>"}},
	},
}
