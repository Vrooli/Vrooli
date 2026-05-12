package artifacts

import "flow-verifier/internal/module"

// Endpoints describes the artifacts HTTP surface. The UI's Artifacts
// panel and the `flow-verifier flows artifacts ...` CLI both target
// these routes; both surfaces share the same contract.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "flows.artifacts.status",
		Path:        "/api/v1/flows/{id}/artifacts",
		Method:      "GET",
		Summary:     "Inspect a flow's generated/ tree",
		Description: "Returns each expected artifact path with its existence/mtime, plus an overall fresh|missing status. Pure inspection — no mutation.",
		Category:    "artifacts",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier artifacts status"},
	},
	{
		ID:          "flows.artifacts.generate",
		Path:        "/api/v1/flows/{id}/artifacts:generate",
		Method:      "POST",
		Summary:     "Generate or regenerate a flow's artifacts",
		Description: "Runs the pipeline in generate mode for one flow. Returns the new artifacts status alongside the recorded run.",
		Category:    "artifacts",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier artifacts generate"},
	},
	{
		ID:          "flows.artifacts.clear",
		Path:        "/api/v1/flows/{id}/artifacts",
		Method:      "DELETE",
		Summary:     "Clear a flow's generated/ tree",
		Description: "Removes every file under <flow-dir>/generated/. Refuses crafted paths that would escape the scenario root.",
		Category:    "artifacts",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier artifacts clear"},
	},
	{
		ID:          "scenarios.artifacts.generate",
		Path:        "/api/v1/scenarios/{id}/artifacts:generate",
		Method:      "POST",
		Summary:     "Generate artifacts for every flow in a scenario",
		Description: "Walks every flow inside the scenario and regenerates each one serially.",
		Category:    "artifacts",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier artifacts generate --scenario"},
	},
	{
		ID:          "scenarios.artifacts.clear",
		Path:        "/api/v1/scenarios/{id}/artifacts",
		Method:      "DELETE",
		Summary:     "Clear artifacts for every flow in a scenario",
		Description: "Walks every flow inside the scenario and clears each one.",
		Category:    "artifacts",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier artifacts clear --scenario"},
	},
}
