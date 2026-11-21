package provision

import (
	"vrooli-bridge/internal/module"

	provisionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision/provision_v1connect"
)

// Endpoints is the machine-readable description of the provision module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in provision.proto breaks this file at compile
// time. The global parity test (TestProtoConnectParity) asserts every rpc has
// exactly one entry here once provision is listed in AllProtoFiles().
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "provision_sync_to_revision",
		Path:        provisionconnect.ProvisionServiceSyncToRevisionProcedure,
		Method:      "POST",
		Summary:     "Sync a node to a project revision (privileged)",
		Description: "Brings a node to a target git revision (`git fetch`@R + idempotent `vrooli setup`) via the node's separate privileged helper, creating a durable, audited provisioning op. Validates and rejects an unknown/revoked/offline node before any op is created. target_revision defaults to the control plane's current commit (pass \"@cp\" to say so explicitly) and is preflighted against the clone remote, so an unpushed commit is rejected with push-first guidance; rollback_revision accepts \"@cp\" too. Honours X-Dry-Run. Owner-gated.",
		Category:    "provision",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"node_id":           "string (required)",
			"target_revision":   "string (defaults to the control plane's commit; \"@cp\" = same)",
			"rollback_revision": "string (\"@cp\" accepted)",
			"timeout_seconds":   "int64",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"op_id":   "string",
			"dry_run": "bool",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing node_id, or an invalid revision (e.g. a relative ref like HEAD~1)"},
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "Unknown node"},
			{Status: 409, Code: "failed_precondition", Description: "Node revoked or offline"},
			{Status: 412, Code: "failed_precondition", Description: "The resolved commit is not on the clone remote (push it first)"},
			{Status: 503, Code: "unavailable", Description: "Provisioning command could not be delivered"},
		},
		Examples: []module.Example{
			{Name: "Sync a node to a revision", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.provision.ProvisionService/SyncToRevision -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"node_id\":\"abc123\",\"target_revision\":\"a1b2c3d\"}'"},
		},
	},
	{
		ID:          "provision_get_provisioning_op",
		Path:        provisionconnect.ProvisionServiceGetProvisioningOpProcedure,
		Method:      "POST",
		Summary:     "Get a provisioning op by id with its event history",
		Description: "Returns one durable provisioning op and its full persisted event history (logs/status/version/exit). Re-attaching after a client disconnect is just calling this again. Owner-gated.",
		Category:    "provision",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"op": "ProvisioningOp", "events": "array<ProvisionEvent>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No op with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get provisioning op", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.provision.ProvisionService/GetProvisioningOp -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"op-123\"}'"},
		},
	},
	{
		ID:          "provision_list_provisioning_ops",
		Path:        provisionconnect.ProvisionServiceListProvisioningOpsProcedure,
		Method:      "POST",
		Summary:     "List provisioning ops",
		Description: "Returns provisioning ops newest-first, optionally filtered by node. Owner-gated.",
		Category:    "provision",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"node_id": "string", "limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"ops": "array<ProvisioningOp>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List provisioning ops", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.provision.ProvisionService/ListProvisioningOps -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "provision_wait_provisioning_op",
		Path:        provisionconnect.ProvisionServiceWaitProvisioningOpProcedure,
		Method:      "POST",
		Summary:     "Block-once wait for a provisioning op to finish",
		Description: "Blocks server-side and returns EXACTLY ONCE when the op reaches a terminal status (no polling). Returns timed_out=true if the wait deadline elapses first. Owner-gated.",
		Category:    "provision",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)", "timeout_seconds": "int64"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"op": "ProvisioningOp", "timed_out": "bool"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No op with that id"},
			{Status: 499, Code: "canceled", Description: "Client cancelled the wait"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Wait for provisioning op", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.provision.ProvisionService/WaitProvisioningOp -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"op-123\",\"timeout_seconds\":1800}'"},
		},
	},
	{
		ID:          "provision_get_node_version",
		Path:        provisionconnect.ProvisionServiceGetNodeVersionProcedure,
		Method:      "POST",
		Summary:     "Get a node's current project revision",
		Description: "Returns a node's current recorded project revision (the last VERSION a provisioning op reported). has_version=false when the node has never been provisioned. Owner-gated.",
		Category:    "provision",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"node_id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"version": "NodeVersion", "has_version": "bool"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get node version", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.provision.ProvisionService/GetNodeVersion -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"node_id\":\"abc123\"}'"},
		},
	},
	{
		ID:          "provision_report_provision_event",
		Path:        provisionconnect.ProvisionServiceReportProvisionEventProcedure,
		Method:      "POST",
		Summary:     "Ingest a provision event from the node-agent (node-facing)",
		Description: "Node-facing: the node-agent's privileged helper streams ProvisionEvents back here, signed with its per-node Ed25519 credential. A node may only report against its own ops. A VERSION event records the node's current revision; a terminal EXIT flips the op terminal and wakes block-once waiters. Not an operator verb — omitted from the CLI manifest.",
		Category:    "provision",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"event": "ProvisionEvent"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"accepted": "bool"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing op_id"},
			{Status: 401, Code: "unauthenticated", Description: "Node mutual-auth signature required"},
			{Status: 403, Code: "permission_denied", Description: "A node may only report its own ops"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Report provision event (node-agent)", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.provision.ProvisionService/ReportProvisionEvent -H 'X-Bridge-Node: <node>' -H 'X-Bridge-Timestamp: <ts>' -H 'X-Bridge-Signature: <sig>' -H 'Content-Type: application/json' -d '{\"event\":{\"op_id\":\"op-123\",\"kind\":\"PROVISION_EVENT_KIND_LOG\",\"log_chunk\":\"...\"}}'"},
		},
	},
}
