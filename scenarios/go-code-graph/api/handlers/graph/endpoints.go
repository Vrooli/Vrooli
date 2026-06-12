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
		Description: "Loads the Go module rooted at module_path and returns the normalized graph + warnings + extraction time + graph hash.",
		Category:    "graph",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"module_path":    "string (required, absolute or scenario-relative module root)",
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
			{Status: 400, Code: "invalid_argument", Description: "Missing or invalid module_path; no/multiple go.mod"},
			{Status: 404, Code: "not_found", Description: "module_path unreadable"},
			{Status: 501, Code: "unimplemented", Description: "go.work multi-module workspaces are not supported in v1"},
			{Status: 500, Code: "internal", Description: "Loader failure"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "go-code-graph graph extract",
			Args:    []string{"<path>", "[--include-vendor]"},
		},
	},
	{
		ID:          "graph_list_fixtures",
		Path:        graph_v1connect.GoCodeGraphServiceListFixturesProcedure,
		Method:      "POST",
		Summary:     "List golden determinism fixtures",
		Description: "Enumerates the fixture directories under bas/fixtures so the Fixture Validator UI can offer them without reading repo files in the browser.",
		Category:    "graph",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"fixtures": "array<{name:string, path:string, has_expected:bool}>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Fixtures directory unreadable"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "go-code-graph graph list-fixtures",
		},
	},
	{
		ID:          "graph_validate_fixture",
		Path:        graph_v1connect.GoCodeGraphServiceValidateFixtureProcedure,
		Method:      "POST",
		Summary:     "Validate a fixture against its golden graph",
		Description: "Re-runs Extract against a named fixture server-side and byte-compares the canonical JSON against expected-graph.json, returning pass/fail plus a line diff.",
		Category:    "graph",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"name": "string (required, fixture directory name)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"passed":         "boolean",
				"diff":           "string (line diff; empty on pass)",
				"expected_bytes": "int64",
				"actual_bytes":   "int64",
				"graph_hash":     "string (hex sha256 of canonical graph)",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing or path-escaping fixture name"},
			{Status: 404, Code: "not_found", Description: "Fixture not found"},
			{Status: 412, Code: "failed_precondition", Description: "Fixture has no expected-graph.json baseline"},
			{Status: 500, Code: "internal", Description: "Extraction or comparison failure"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "go-code-graph graph validate-fixture",
			Args:    []string{"<name>"},
		},
	},
}
