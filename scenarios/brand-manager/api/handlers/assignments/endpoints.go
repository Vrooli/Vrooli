package assignments

import (
	"brand-manager/internal/module"

	assignmentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments/assignments_v1connect"
)

// Endpoints is the machine-readable description of the assignments module's
// public surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in assignments.proto breaks this file at
// compile time. The global parity test in modules/registry_test.go asserts
// every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "assignments_list",
		Path:        assignmentsconnect.AssignmentsServiceListAssignmentsProcedure,
		Method:      "POST",
		Summary:     "List brand assignments",
		Description: "Returns assignments ordered newest-applied first, optionally filtered to one brand.",
		Category:    "assignments",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"brand_id": "string (optional filter; empty = all)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"assignments": "array<Assignment>"}},
		Errors:      []module.ErrorDesc{{Status: 500, Code: "internal", Description: "Repository read failure"}},
		Examples:    []module.Example{{Name: "List assignments", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.assignments.AssignmentsService/ListAssignments -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "assignments_assign",
		Path:        assignmentsconnect.AssignmentsServiceAssignBrandProcedure,
		Method:      "POST",
		Summary:     "Assign a brand to a scenario",
		Description: "Upserts a scenario's brand link and pins the brand's current version. Re-assigning replaces the prior link. brand_id and scenario_name are required and the brand must exist.",
		Category:    "assignments",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"brand_id":      "string (required, must exist)",
			"scenario_name": "string (required)",
			"elements":      "array<string> (optional applied elements)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"assignment": "Assignment"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing brand_id/scenario_name or unknown brand"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{{Name: "Assign brand", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.assignments.AssignmentsService/AssignBrand -H 'Content-Type: application/json' -d '{\"brand_id\":\"abc123\",\"scenario_name\":\"web-console\"}'"}},
	},
	{
		ID:          "assignments_status",
		Path:        assignmentsconnect.AssignmentsServiceGetScenarioStatusProcedure,
		Method:      "POST",
		Summary:     "Get a scenario's branding status",
		Description: "Returns whether a scenario has a brand assigned and, if so, the pinned brand id/version, applied elements, and applied-at time. An unbranded scenario yields has_brand=false (not an error).",
		Category:    "assignments",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario_name": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"status": "ScenarioStatus"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario_name"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{{Name: "Scenario status", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.assignments.AssignmentsService/GetScenarioStatus -H 'Content-Type: application/json' -d '{\"scenario_name\":\"web-console\"}'"}},
	},
	{
		ID:          "assignments_unassign",
		Path:        assignmentsconnect.AssignmentsServiceUnassignScenarioProcedure,
		Method:      "POST",
		Summary:     "Unassign a scenario's brand (idempotent)",
		Description: "Removes a scenario's brand assignment. Unassigning a scenario with no brand returns success.",
		Category:    "assignments",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario_name": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario_name"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{{Name: "Unassign scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.brand_manager.v1.assignments.AssignmentsService/UnassignScenario -H 'Content-Type: application/json' -d '{\"scenario_name\":\"web-console\"}'"}},
	},
}
