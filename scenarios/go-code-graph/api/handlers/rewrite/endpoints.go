// Package rewrite is the handler-layer home for the rewrite RPCs. It
// owns no Connect mount of its own — the GoCodeGraphService Connect
// handler is mounted by handlers/graph because the proto declaration
// is a single service. This package contributes:
//
//   - Endpoints: the machine-readable descriptors for RewritePlan /
//     RewriteApply that the codegen + registry walk.
//   - Schema(): the rewrite-domain DDL (rewrite_plans +
//     rewrite_operation_log) re-exported from internal/rewrite so the
//     registry's AllSchemas walk picks it up uniformly with other
//     domains.
//
// The handler logic lives in handlers/graph/handler.go where the
// connect-handler struct holds references to BOTH *intgraph.Service
// AND *intrewrite.Service. See the doc comment on handlers/graph/module.go.
package rewrite

import (
	"go-code-graph/internal/module"
	intrewrite "go-code-graph/internal/rewrite"

	"github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
)

// Endpoints is the machine-readable description of the two rewrite
// Connect procedures. Paths reference the generated Procedure
// constants so renaming an RPC in graph.proto breaks this file at
// compile time. The complementary parity test in registry_test.go
// walks the proto FileDescriptor and asserts every rpc has exactly
// one matching entry across all handler packages.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "rewrite_plan",
		Path:        graph_v1connect.GoCodeGraphServiceRewritePlanProcedure,
		Method:      "POST",
		Summary:     "Plan a set of rewrite operations",
		Description: "Validates and normalizes a list of FileMove/ImportRewrite operations and returns a plan_id. No filesystem changes.",
		Category:    "rewrite",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"module_path": "string (required)",
				"operations":  "array<Operation> (required, ≥1)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"plan_id":               "string",
				"normalized_operations": "array<Operation>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Empty operations list or malformed op"},
			{Status: 500, Code: "internal", Description: "Plan persistence failure"},
		},
	},
	{
		ID:          "rewrite_apply",
		Path:        graph_v1connect.GoCodeGraphServiceRewriteApplyProcedure,
		Method:      "POST",
		Summary:     "Apply a previously-planned rewrite",
		Description: "Executes the operations associated with plan_id. Caller must set apply=true; apply=false is rejected with InvalidArgument so dry-run callers do not accidentally mutate. X-Dry-Run: true header threads through to a synthetic-OK response without invoking the executor.",
		Category:    "rewrite",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"module_path": "string (required)",
				"plan_id":     "string (required)",
				"apply":       "boolean (must be true)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"plan_id": "string",
				"results": "array<OperationResult>",
				"dry_run": "boolean",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unknown plan_id, apply=false, or malformed input"},
			{Status: 412, Code: "failed_precondition", Description: "Plan was created for a different module_path"},
			{Status: 500, Code: "internal", Description: "Executor or persistence failure"},
		},
	},
}

// Schema re-exports the rewrite domain's DDL (rewrite_plans +
// rewrite_operation_log) for EnsureSchemas. The actual SQL lives in
// internal/rewrite/schema.sql per the domain-owns-its-tables convention.
func Schema() string { return intrewrite.Schema() }
