package runs

import (
	"flow-verifier/internal/module"

	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs/runs_v1connect"
)

// Endpoints describes the runs Connect-RPC surface for the codegen
// pipeline that emits .vrooli/endpoints.json.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "runs.list",
		Path:        runsconnect.RunsServiceListRunsProcedure,
		Method:      "POST",
		Summary:     "List verification runs",
		Description: "Returns persisted verification history with optional flow_id filter and a configurable limit.",
		Category:    "runs",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier runs list"},
	},
	{
		ID:          "runs.get",
		Path:        runsconnect.RunsServiceGetRunProcedure,
		Method:      "POST",
		Summary:     "Single run detail",
		Description: "Returns one verification run including the counterexample blob on failure.",
		Category:    "runs",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier runs show"},
	},
}
