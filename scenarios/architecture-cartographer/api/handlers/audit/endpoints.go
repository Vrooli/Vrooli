package audit

import (
	"architecture-cartographer/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/audit/audit_v1connect"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// Endpoints describes the audit domain's Connect-RPC routes.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "audit.run",
		Path:        audit_v1connect.AuditServiceRunProcedure,
		Method:      "POST",
		Summary:     "Run a CI-shaped drift audit",
		Description: "Orchestrates graph extract (if needed), domains derivation, and conflicts detection; applies severity / type filters; returns a deterministic summary and exit code mapping.",
		Category:    "audit",
	},
	{
		ID:          "audit.run-all",
		Path:        audit_v1connect.AuditServiceRunAllProcedure,
		Method:      "POST",
		Summary:     "Sweep every discoverable scenario",
		Description: "Walks scenarios/*/.vrooli/service.json, runs Audit on each, and returns per-scenario reports plus a totals rollup. Honors include_scenarios / exclude_scenarios filters.",
		Category:    "audit",
	},
	{
		ID:          "validation.validate-scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate architecture posture",
		Description: "Runs the architecture audit in advisory mode and returns the shared scenario-validation response; the native AuditRunResponse is packed into native_detail.",
		Category:    "validation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":      "string",
				"status":        "scenario_validation.v1.ValidationStatus",
				"assessment":    "common.v1.MaturityAssessment",
				"native_detail": "google.protobuf.Any<architecture_cartographer.v1.audit.AuditRunResponse>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Scenario is missing or invalid"},
		},
		Examples: []module.Example{
			{Name: "Validate architecture", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"proto-health\"}'"},
		},
	},
}
