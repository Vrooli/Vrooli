package graph

import (
	"typescript-code-graph/internal/module"

	"github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph/graph_v1connect"
)

// Endpoints is the machine-readable description of every Connect
// procedure this module mounts. Paths reference the generated Procedure
// constants so renaming an RPC in graph.proto breaks this file at
// compile time.
//
// Only graph_extract lives here; the two rewrite RPCs' descriptors
// live in handlers/rewrite/endpoints.go (Phase 5 split). The global
// parity test in api/internal/modules/registry_test.go walks every
// module's Endpoints in concert and asserts each rpc method has
// exactly one matching descriptor across all modules.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "graph_extract",
		Path:        graph_v1connect.TypeScriptCodeGraphServiceExtractProcedure,
		Method:      "POST",
		Summary:     "Extract a deterministic TypeScript graph",
		Description: "Loads the TypeScript project rooted at scenario_path via the Node sidecar (ts-morph) and returns the normalized graph + warnings + extraction time + graph hash. Declaration nodes carry their leading_comments[] verbatim — REQ-P0-003.",
		Category:    "graph",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario_path": "string (required, absolute path to the project root containing tsconfig.json)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"graph":              "common.v1.CodeGraph",
				"warnings":           "array<common.v1.CodeGraphWarning>",
				"extraction_ms":      "int64",
				"graph_hash":         "string (hex sha256 of canonical graph)",
				"sidecar_request_id": "string (uuid of the originating sidecar IPC request)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing or non-absolute scenario_path; no/multiple tsconfig.json"},
			{Status: 404, Code: "not_found", Description: "scenario_path unreadable"},
			{Status: 501, Code: "unimplemented", Description: "pnpm/yarn workspaces are not supported in v1"},
			{Status: 503, Code: "unavailable", Description: "Node sidecar is unhealthy or permanently failed"},
			{Status: 504, Code: "deadline_exceeded", Description: "Sidecar call exceeded its deadline"},
			{Status: 500, Code: "internal", Description: "Unexpected sidecar or normalization failure"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "typescript-code-graph graph extract",
			Args:    []string{"<path>"},
		},
	},
}
