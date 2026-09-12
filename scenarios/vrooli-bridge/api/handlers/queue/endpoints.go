package queue

import (
	"vrooli-bridge/internal/module"

	queueconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/queue/queue_v1connect"
)

// Endpoints is the machine-readable description of the queue module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in queue.proto breaks this file at compile time.
// The global parity test (TestProtoConnectParity) asserts every rpc has exactly
// one entry here once queue is listed in AllProtoFiles().
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "queue_list_queue",
		Path:        queueconnect.QueueServiceListQueueProcedure,
		Method:      "POST",
		Summary:     "List the live per-node job queue",
		Description: "Returns the live scheduler view: one NodeQueue per node with any running or queued job (running first, then queued in FIFO order with positions), optionally narrowed to a single node. Read-only; the queue is mutated through dispatch (enqueue) and runs AbortRun (cancel). Owner-gated.",
		Category:    "queue",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"node_id": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"nodes": "array<NodeQueue>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
		},
		Examples: []module.Example{
			{Name: "List the queue", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.queue.QueueService/ListQueue -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{}'"},
			{Name: "List one node's queue", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.queue.QueueService/ListQueue -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"node_id\":\"abc123\"}'"},
		},
	},
}
