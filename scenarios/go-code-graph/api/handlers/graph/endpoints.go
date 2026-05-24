package graph

import (
	"go-code-graph/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph/graph_v1connect"
)

// Endpoints is the machine-readable description of every Connect
// procedure this module mounts. Paths reference the generated Procedure
// constants so renaming an RPC in graph.proto breaks this file at
// compile time. The complementary global parity test in
// api/internal/modules/registry_test.go (if present) walks the proto
// FileDescriptor and asserts every rpc has exactly one matching entry
// here.
//
// Endpoints holds the graph-domain Connect procedure descriptors. The
// rewrite_plan / rewrite_apply descriptors live in handlers/rewrite
// because they belong to that domain even though all three RPCs ride
// the single GoCodeGraphService Connect mount that this package owns.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "graph_extract",
		Path:        graph_v1connect.GoCodeGraphServiceExtractProcedure,
		Method:      "POST",
		Summary:     "Extract a deterministic Go graph",
		Description: "Loads the Go module rooted at scenario_path and returns the normalized graph + warnings + extraction time + graph hash.",
		Category:    "graph",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario_path":  "string (required, absolute or scenario-relative module root)",
				"include_vendor": "boolean (default false)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"graph":         "common.v1.CodeGraph",
				"warnings":      "array<common.v1.CodeGraphWarning>",
				"extraction_ms": "int64",
				"graph_hash":    "string (hex sha256 of canonical graph)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing or invalid scenario_path; no/multiple go.mod"},
			{Status: 404, Code: "not_found", Description: "scenario_path unreadable"},
			{Status: 501, Code: "unimplemented", Description: "go.work multi-module workspaces are not supported in v1"},
			{Status: 500, Code: "internal", Description: "Loader failure"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "go-code-graph graph extract",
			Args:    []string{"<path>", "[--include-vendor]"},
		},
	},
}
