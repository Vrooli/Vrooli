package dispatch

import (
	"vrooli-bridge/internal/module"

	dispatchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch/dispatch_v1connect"
)

// Endpoints is the machine-readable description of the dispatch module's public
// surface. The Connect-RPC method path references the generated *Procedure
// constant, so renaming the RPC in dispatch.proto breaks this at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "dispatch_dispatch_job",
		Path:        dispatchconnect.DispatchServiceDispatchJobProcedure,
		Method:      "POST",
		Summary:     "Dispatch a typed job to a node",
		Description: "Validates {scenario, verb, args} against the manifest allowlist + the node's granted scopes, then creates a durable run, audits the dispatch, and pushes the typed job to the node. An undeclared/out-of-scope verb is rejected (and audited) before any run is created. Honours X-Dry-Run. Owner-gated.",
		Category:    "dispatch",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"node_id":         "string (required)",
			"scenario":        "string",
			"verb":            "string (required)",
			"args":            "array<string>",
			"timeout_seconds": "int64",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"run_id":  "string",
			"dry_run": "bool",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing/empty verb"},
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 403, Code: "permission_denied", Description: "Verb not allowlisted or out of the node's scopes"},
			{Status: 404, Code: "not_found", Description: "Unknown node"},
			{Status: 409, Code: "failed_precondition", Description: "Node revoked or offline"},
			{Status: 503, Code: "unavailable", Description: "Job could not be delivered to the node"},
		},
		Examples: []module.Example{
			{Name: "Dispatch a scenario test", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.dispatch.DispatchService/DispatchJob -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"node_id\":\"abc123\",\"scenario\":\"web-search\",\"verb\":\"scenario test\",\"args\":[\"web-search\"]}'"},
		},
	},
}
