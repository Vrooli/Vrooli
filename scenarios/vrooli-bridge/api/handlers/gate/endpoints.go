package gate

import (
	"vrooli-bridge/internal/module"

	gateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate/gate_v1connect"
)

// Endpoints is the machine-readable description of the gate module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in gate.proto breaks this file at compile time.
// The global parity test (TestProtoConnectParity) asserts every rpc has exactly
// one entry here once gate is listed in AllProtoFiles().
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "gate_run_gate",
		Path:        gateconnect.GateServiceRunGateProcedure,
		Method:      "POST",
		Summary:     "Run a cross-OS deployment gate for a scenario",
		Description: "Fans a scenario's validation out across the target OSes: selects one eligible node per OS (registered + online + protocol-compatible + not-revoked + matching OS), dispatches the validation run to each (delegating to the dispatch + runs domains), and records a durable gate with the per-OS ledger. Aggregates per-OS verdicts into one cross-OS result — any failing OS (including one with no eligible node) fails the gate. Honours X-Dry-Run. Owner-gated.",
		Category:    "gate",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"scenario":        "string (required)",
			"target_revision": "string (required)",
			"target_oses":     "array<string> (required)",
			"verb":            "string",
			"args":            "array<string>",
			"timeout_seconds": "int64",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"gate_id": "string",
			"dry_run": "bool",
			"verdict": "GateVerdict",
			"results": "array<OSResult>",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario / target_revision / target_oses"},
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Run a cross-OS gate", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.gate.GateService/RunGate -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"scenario\":\"web-search\",\"target_revision\":\"a1b2c3d\",\"target_oses\":[\"linux\",\"darwin\",\"windows\"]}'"},
		},
	},
	{
		ID:          "gate_get_gate",
		Path:        gateconnect.GateServiceGetGateProcedure,
		Method:      "POST",
		Summary:     "Get a gate by id with its per-OS ledger",
		Description: "Returns one durable gate and its per-OS result ledger, recomputing the live cross-OS verdict from the current state of each OS's validation run. Owner-gated.",
		Category:    "gate",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"gate": "Gate", "results": "array<OSResult>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No gate with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get gate", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.gate.GateService/GetGate -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"gate-123\"}'"},
		},
	},
	{
		ID:          "gate_wait_gate",
		Path:        gateconnect.GateServiceWaitGateProcedure,
		Method:      "POST",
		Summary:     "Block once until a gate is terminal, then return its verdict",
		Description: "Blocks server-side until every target OS's validation run is terminal (or the timeout elapses), then returns the final aggregate cross-OS verdict + per-OS ledger. timed_out is true when the deadline elapsed with targets still running. No polling — one call, one return. Owner-gated.",
		Category:    "gate",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"id":              "string (required)",
			"timeout_seconds": "int64",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"gate":      "Gate",
			"results":   "array<OSResult>",
			"timed_out": "bool",
		}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No gate with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Wait for a gate", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.gate.GateService/WaitGate -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"gate-123\"}'"},
		},
	},
	{
		ID:          "gate_list_gates",
		Path:        gateconnect.GateServiceListGatesProcedure,
		Method:      "POST",
		Summary:     "List gates",
		Description: "Returns gates newest-first. Owner-gated.",
		Category:    "gate",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"gates": "array<Gate>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List gates", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.gate.GateService/ListGates -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
