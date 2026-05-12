package flows

import "flow-verifier/internal/module"

// Endpoints describes the flows HTTP surface. Real request/response
// schemas land when Phase C wires the in-process flows domain into
// these handlers; the paths and methods are stable from day one.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "flows.list",
		Path:        "/api/v1/flows",
		Method:      "GET",
		Summary:     "List discovered flows",
		Description: "Walks the configured root and returns one entry per flow/flow.json with id, path, language, and schema version.",
		Category:    "flows",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier flows list"},
	},
	{
		ID:          "flows.get",
		Path:        "/api/v1/flows/{id}",
		Method:      "GET",
		Summary:     "Flow detail",
		Description: "Returns the typed flow plus the latest verification status.",
		Category:    "flows",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier flows explain"},
	},
}
