// Package machines describes the operator-facing fleet surface's Connect-RPC
// contract. The handler implementation lives beside the Bridge adapter in the
// api package; this file is the endpoint registry's view of it.
package machines

import (
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/machines/machines_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the machines surface. Procedure constants come from
// generated code so a schema or service rename fails here at compile time
// instead of drifting silently from the endpoint registry.
var Endpoints = []module.EndpointDescriptor{
	{
		ID: "machines_list", Path: machinesconnect.MachineServiceListProcedure, Method: "POST",
		Summary: "List linked machines", Description: "Returns every linked machine with reachability and grant, plus machines asking to join and the permission presets available.", Category: "machines",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"state": "FleetState", "machines": "[]Machine", "join_requests": "[]JoinRequest", "presets": "[]PermissionPreset"}},
	},
	{
		ID: "machines_issue_code", Path: machinesconnect.MachineServiceIssueCodeProcedure, Method: "POST",
		Summary: "Issue a join code", Description: "Mints a single-use, short-lived code a machine can redeem to join the fleet.", Category: "machines",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"label": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"code": "string", "expires_in_seconds": "int64"}},
		Errors:   []module.ErrorDesc{{Status: 503, Code: "unavailable", Description: "The fleet control plane did not answer"}},
	},
	{
		ID: "machines_decide", Path: machinesconnect.MachineServiceDecideProcedure, Method: "POST",
		Summary: "Answer a join request", Description: "Approves or denies one machine asking to join. Approval requires the confirmation words shown by the joining machine.", Category: "machines",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"request_id": "string", "approve": "bool", "confirmation_words": "[]string", "preset": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"machine": "Machine", "message": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "The confirmation words do not match the derived value"},
			{Status: 503, Code: "unavailable", Description: "The fleet control plane did not answer"},
		},
	},
	{
		ID: "machines_set_grant", Path: machinesconnect.MachineServiceSetGrantProcedure, Method: "POST",
		Summary: "Change what a machine may do", Description: "Applies a permission preset, or explicit scopes, to an already-linked machine.", Category: "machines",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"machine_id": "string", "preset": "string", "scopes": "[]string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"machine": "Machine"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "The machine id or preset is not recognized"},
			{Status: 503, Code: "unavailable", Description: "The fleet control plane did not answer"},
		},
	},
	{
		ID: "machines_forget", Path: machinesconnect.MachineServiceForgetProcedure, Method: "POST",
		Summary: "Forget a machine", Description: "Removes one machine from the fleet.", Category: "machines",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"machine_id": "string"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"forgotten_machine_id": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "The machine id is not a linked machine"},
			{Status: 503, Code: "unavailable", Description: "The fleet control plane did not answer"},
		},
	},
}
