package flows

import (
	"flow-verifier/internal/module"

	flowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows/flows_v1connect"
)

// Endpoints describes the flows Connect-RPC surface for the codegen
// pipeline that emits .vrooli/endpoints.json.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "flows.list",
		Path:        flowsconnect.FlowsServiceListFlowsProcedure,
		Method:      "POST",
		Summary:     "List discovered flows",
		Description: "Walks the configured root and returns one entry per flow/flow.json with id, contract path, language, and schema version.",
		Category:    "flows",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier flows list"},
	},
	{
		ID:          "flows.get",
		Path:        flowsconnect.FlowsServiceGetFlowProcedure,
		Method:      "POST",
		Summary:     "Flow detail",
		Description: "Returns the typed flow.json projection consumed by the UI's flow-detail page.",
		Category:    "flows",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier flows show"},
	},
	{
		ID:          "flows.create",
		Path:        flowsconnect.FlowsServiceCreateFlowProcedure,
		Method:      "POST",
		Summary:     "Scaffold a new flow",
		Description: "Creates a fresh flow directory with hand-authored transition + test sidecars. Replaces the legacy in-process CLI command.",
		Category:    "flows",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier flows new"},
	},
	{
		ID:          "flows.validate",
		Path:        flowsconnect.FlowsServiceValidateFlowProcedure,
		Method:      "POST",
		Summary:     "Validate every flow under root",
		Description: "Compiles every flow.json and returns the first compilation/contract error encountered, or empty on success.",
		Category:    "flows",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier flows validate"},
	},
	{
		ID:          "flows.explain",
		Path:        flowsconnect.FlowsServiceExplainFlowProcedure,
		Method:      "POST",
		Summary:     "Human-readable explain report",
		Description: "Renders the markdown explanation consumed by the UI Overview tab and CLI explain output.",
		Category:    "flows",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier flows explain"},
	},
	{
		ID:          "flows.codegen",
		Path:        flowsconnect.FlowsServiceCodegenFlowProcedure,
		Method:      "POST",
		Summary:     "Emit codegen artifacts",
		Description: "Runs the navigation kind's codegen (routes.generated.ts) and optionally writes the output to disk.",
		Category:    "flows",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier flows codegen"},
	},
	{
		ID:          "flows.reconcile",
		Path:        flowsconnect.FlowsServiceReconcileFlowProcedure,
		Method:      "POST",
		Summary:     "Reconcile spec against source",
		Description: "Walks the scenario's ui/src tree and asserts every route declared in code matches a spec route (and vice-versa).",
		Category:    "flows",
		CLIMapping:  &module.CLIMapping{Command: "flow-verifier flows reconcile"},
	},
}
