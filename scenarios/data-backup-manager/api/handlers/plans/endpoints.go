package plans

import (
	"data-backup-manager/internal/module"

	plansconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans/plans_v1connect"
)

// Endpoints is the machine-readable description of the plans module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so adding or renaming an RPC in plans.proto breaks this file at
// compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "plans_create",
		Path:        plansconnect.PlansServiceCreatePlanProcedure,
		Method:      "POST",
		Summary:     "Create a backup plan",
		Description: "Creates a new plan binding targets to destinations with a schedule and retention policy.",
		Category:    "plans",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"name":            "string (required)",
				"target_ids":      "array<string> (required, ≥1)",
				"destination_ids": "array<string> (required, ≥1)",
				"schedule":        "string (duration, optional)",
				"retention":       "RetentionPolicy (optional)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"plan": "Plan"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing name, target_ids, or destination_ids"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Create a nightly plan", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.plans.PlansService/CreatePlan -H 'Content-Type: application/json' -d '{\"name\":\"nightly\",\"targetIds\":[\"tgt-1\"],\"destinationIds\":[\"dst-1\"],\"schedule\":\"24h\"}'"},
		},
	},
	{
		ID:          "plans_get",
		Path:        plansconnect.PlansServiceGetPlanProcedure,
		Method:      "POST",
		Summary:     "Get a plan by id",
		Description: "Returns the plan including its target and destination membership lists.",
		Category:    "plans",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"plan": "Plan"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No plan with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get a plan", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.plans.PlansService/GetPlan -H 'Content-Type: application/json' -d '{\"id\":\"plan-1\"}'"},
		},
	},
	{
		ID:          "plans_list",
		Path:        plansconnect.PlansServiceListPlansProcedure,
		Method:      "POST",
		Summary:     "List backup plans",
		Description: "Lists all plans with their membership lists, with cursor pagination.",
		Category:    "plans",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"page_size":  "int32",
				"page_token": "string",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"plans":           "array<Plan>",
				"next_page_token": "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List all plans", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.plans.PlansService/ListPlans -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "plans_update",
		Path:        plansconnect.PlansServiceUpdatePlanProcedure,
		Method:      "POST",
		Summary:     "Update a backup plan",
		Description: "Replaces the plan's fields and membership lists in full.",
		Category:    "plans",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":              "string (required)",
				"name":            "string",
				"target_ids":      "array<string>",
				"destination_ids": "array<string>",
				"schedule":        "string",
				"retention":       "RetentionPolicy",
				"enabled":         "bool",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"plan": "Plan"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id, name, target_ids, or destination_ids"},
			{Status: 404, Code: "not_found", Description: "No plan with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Update a plan", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.plans.PlansService/UpdatePlan -H 'Content-Type: application/json' -d '{\"id\":\"plan-1\",\"name\":\"nightly-v2\",\"targetIds\":[\"tgt-1\"],\"destinationIds\":[\"dst-1\"]}'"},
		},
	},
	{
		ID:          "plans_delete",
		Path:        plansconnect.PlansServiceDeletePlanProcedure,
		Method:      "POST",
		Summary:     "Delete a backup plan",
		Description: "Removes a plan and its membership rows by id.",
		Category:    "plans",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"removed": "boolean"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository delete failure"},
		},
		Examples: []module.Example{
			{Name: "Delete a plan", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.plans.PlansService/DeletePlan -H 'Content-Type: application/json' -d '{\"id\":\"plan-1\"}'"},
		},
	},
}
