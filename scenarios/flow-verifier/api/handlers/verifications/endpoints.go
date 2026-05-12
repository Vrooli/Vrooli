package verifications

import "flow-verifier/internal/module"

// Endpoints describes the verifications HTTP surface. The shape is
// locked now; real handlers land in Phase D.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "verifications.start",
		Path:        "/api/v1/verifications",
		Method:      "POST",
		Summary:     "Start a verification",
		Description: "Kicks off pipeline (discover → compile → artifact → codegen → lint) for one or all flows under the given root and returns a runId.",
		Category:    "verifications",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier verify run"},
	},
	{
		ID:          "verifications.get",
		Path:        "/api/v1/verifications/{runId}",
		Method:      "GET",
		Summary:     "Verification status",
		Description: "Returns status + result for one verification run.",
		Category:    "verifications",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier runs show"},
	},
}
