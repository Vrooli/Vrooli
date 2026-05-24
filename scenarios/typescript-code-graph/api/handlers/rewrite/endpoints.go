package rewrite

import (
	"typescript-code-graph/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
)

// Endpoints is the machine-readable description of the two rewrite
// procedures. Paths reference the generated Procedure constants so
// renaming an RPC in graph.proto breaks this file at compile time.
//
// The complementary global parity test in
// api/internal/modules/registry_test.go asserts every rpc method has
// exactly one matching entry across all modules' Endpoints; the
// rewrite RPCs are listed in handlers/graph/endpoints.go too for the
// Phase-4 timeline, but only one copy survives Phase 5: this file
// owns them now, and graph/endpoints.go drops them in lockstep.
//
// cli_mapping.command MUST match the cli_commands_seed.json entries
// ("rewrite plan" and "rewrite apply"); the gen-endpoints cross-check
// fails the build otherwise.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "rewrite_plan",
		Path:        graph_v1connect.TypeScriptCodeGraphServiceRewritePlanProcedure,
		Method:      "POST",
		Summary:     "Plan a TypeScript rewrite",
		Description: "Validates and normalizes a list of FileMove / ImportRewrite operations and returns a deterministic plan_id. Identical normalized inputs produce identical plan_ids; the (scenario_path, plan_id) composite scopes plans per project.",
		Category:    "rewrite",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario_path": "string (required, absolute path)",
				"operations":    "array<rewrite.v1.Operation> (>=1; each must set exactly one of file_move / import_rewrite)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"plan_id":               "string (sha256-hex of normalized operations)",
				"normalized_operations": "array<rewrite.v1.Operation>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario_path, empty operations, absolute/parent path, or both/neither oneof arm set"},
			{Status: 500, Code: "internal", Description: "Plan store save failed"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "typescript-code-graph rewrite plan",
			Args:    []string{"<ops.json>"},
		},
	},
	{
		ID:          "rewrite_apply",
		Path:        graph_v1connect.TypeScriptCodeGraphServiceRewriteApplyProcedure,
		Method:      "POST",
		Summary:     "Apply a TypeScript rewrite plan",
		Description: "Executes the operations recorded under plan_id against the project at scenario_path. The apply flag must be true; X-Dry-Run: true header short-circuits before the sidecar call and returns synthetic OK results.",
		Category:    "rewrite",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario_path": "string (required, absolute path; must match the plan's scenario)",
				"plan_id":       "string (required, returned by RewritePlan)",
				"apply":         "bool (must be true; false is rejected with InvalidArgument)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"plan_id": "string",
				"results": "array<rewrite.v1.OperationResult> (1:1 with normalized operations)",
				"dry_run": "bool (true when the request rode in with X-Dry-Run: true)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario_path / plan_id, or apply=false"},
			{Status: 412, Code: "failed_precondition", Description: "Plan not found for this scenario (mismatched scenario_path or unknown plan_id)"},
			{Status: 503, Code: "unavailable", Description: "Node sidecar is unhealthy or permanently failed"},
			{Status: 504, Code: "deadline_exceeded", Description: "Sidecar call exceeded its deadline"},
			{Status: 500, Code: "internal", Description: "Unexpected sidecar or plan store failure"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "typescript-code-graph rewrite apply",
			Args:    []string{"<plan_id>"},
		},
	},
}
