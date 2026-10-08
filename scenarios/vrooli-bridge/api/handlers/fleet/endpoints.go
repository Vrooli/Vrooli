package fleet

import (
	"vrooli-bridge/internal/module"

	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/fleet/fleet_v1connect"
)

// Endpoints is the machine-readable description of the fleet module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in fleet.proto breaks this file at compile time.
// The global parity test (TestProtoConnectParity) asserts every rpc has exactly
// one entry here once fleet is listed in AllProtoFiles().
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "fleet_roll_fleet",
		Path:        fleetconnect.FleetServiceRollFleetProcedure,
		Method:      "POST",
		Summary:     "Roll the fleet (or a subset) to a target revision",
		Description: "Pins every registered (or named) node to a target git revision: enumerates the eligible nodes, classifies each (online + protocol-compatible + not-revoked → dispatch; otherwise skip with a reason), dispatches a privileged provisioning op to each eligible node, and records a durable rollout with the per-node ledger. target_revision defaults to the control plane's current commit (pass \"@cp\" to say so explicitly) and is resolved+preflighted ONCE up front, so the whole fleet pins to a single exact commit and an unpushed target fails the roll before any node is touched. Honours X-Dry-Run. Owner-gated.",
		Category:    "fleet",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target_revision": "string (defaults to the control plane's commit; \"@cp\" = same)",
			"node_ids":        "array<string>",
			"timeout_seconds": "int64",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"rollout_id": "string",
			"dry_run":    "bool",
			"status":     "RolloutStatus",
			"results":    "array<NodeRolloutResult>",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "An invalid revision (e.g. a relative ref like HEAD~1)"},
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 412, Code: "failed_precondition", Description: "The resolved commit is not on the clone remote (push it first)"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Roll the fleet to a revision", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.fleet.FleetService/RollFleet -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"target_revision\":\"a1b2c3d\"}'"},
		},
	},
	{
		ID:          "fleet_get_rollout",
		Path:        fleetconnect.FleetServiceGetRolloutProcedure,
		Method:      "POST",
		Summary:     "Get a rollout by id with its per-node ledger",
		Description: "Returns one durable rollout and its full per-node result ledger. Owner-gated.",
		Category:    "fleet",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"rollout": "Rollout", "results": "array<NodeRolloutResult>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No rollout with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get rollout", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.fleet.FleetService/GetRollout -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"rollout-123\"}'"},
		},
	},
	{
		ID:          "fleet_list_rollouts",
		Path:        fleetconnect.FleetServiceListRolloutsProcedure,
		Method:      "POST",
		Summary:     "List rollouts",
		Description: "Returns rollouts newest-first. Owner-gated.",
		Category:    "fleet",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"rollouts": "array<Rollout>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List rollouts", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.fleet.FleetService/ListRollouts -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
