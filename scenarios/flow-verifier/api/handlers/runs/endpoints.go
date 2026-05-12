package runs

import "flow-verifier/internal/module"

// Endpoints describes the runs HTTP surface. The shape is locked now;
// real handlers + persistence land in Phase E.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "runs.list",
		Path:        "/api/v1/runs",
		Method:      "GET",
		Summary:     "List verification runs",
		Description: "Returns persisted verification history with optional flowId filter and a configurable limit.",
		Category:    "runs",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier runs list"},
	},
	{
		ID:          "runs.get",
		Path:        "/api/v1/runs/{id}",
		Method:      "GET",
		Summary:     "Single run detail",
		Description: "Returns one verification run including the counterexample blob on failure.",
		Category:    "runs",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier runs show"},
	},
}
