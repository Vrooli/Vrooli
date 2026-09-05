package relay

import (
	"vrooli-bridge/internal/module"

	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID: "relay_call", Path: relayconnect.RelayServiceCallProcedure, Method: "POST",
		Summary:     "Run an admitted command through a node channel",
		Description: "Owner-gated: applies the same manifest and node-scope admission as durable dispatch, sends a signed typed relay frame, and returns the bounded node response.",
		Category:    "relay",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"node_id": "string", "scenario": "string", "command": "string", "args": "array<string>", "timeout_seconds": "int64", "max_response_bytes": "uint64"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"correlation_id": "string", "outcome": "RelayCallOutcome", "data": "bytes", "reason": "string", "exit_code": "int32", "total_bytes": "uint64"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid command or relay request"},
			{Status: 401, Code: "unauthenticated", Description: "Owner authentication required"},
			{Status: 403, Code: "permission_denied", Description: "Command is outside the node's manifest scopes or contains an unsafe token"},
			{Status: 412, Code: "failed_precondition", Description: "Node is revoked, offline, incompatible, or unsupported"},
			{Status: 429, Code: "resource_exhausted", Description: "Relay response exceeds the bounded byte limit"},
		},
	},
}
