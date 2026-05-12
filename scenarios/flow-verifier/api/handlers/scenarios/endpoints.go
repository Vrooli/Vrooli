package scenarios

import "flow-verifier/internal/module"

// Endpoints describes the scenarios HTTP surface. Scenarios are the
// primary aggregate in the inventory UI — flow-level listing rolls up
// through here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "scenarios.list",
		Path:        "/api/v1/scenarios",
		Method:      "GET",
		Summary:     "List discovered scenarios",
		Description: "Walks <vrooli-root>/scenarios/* for .vrooli/service.json descriptors and returns one row per scenario with display name, description, path, and flow count.",
		Category:    "scenarios",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier scenarios list"},
	},
	{
		ID:          "scenarios.get",
		Path:        "/api/v1/scenarios/{id}",
		Method:      "GET",
		Summary:     "Scenario detail",
		Description: "Returns the scenario descriptor plus the flow summaries discovered inside it.",
		Category:    "scenarios",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier scenarios get"},
	},
}
