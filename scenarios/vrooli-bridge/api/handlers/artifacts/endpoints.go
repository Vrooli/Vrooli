package artifacts

import (
	"vrooli-bridge/internal/module"

	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts/artifacts_v1connect"
)

// Endpoints is the machine-readable description of the artifacts module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so renaming an RPC in artifacts.proto breaks this file at compile
// time. The global parity test (TestProtoConnectParity) asserts every rpc has
// exactly one entry here once artifacts is listed in AllProtoFiles().
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "artifacts_distribute_artifact",
		Path:        artifactsconnect.ArtifactsServiceDistributeArtifactProcedure,
		Method:      "POST",
		Summary:     "Distribute a non-git artifact to a node (via device-sync-hub)",
		Description: "Streams a non-git artifact (a built installer, a large fixture) to a node through device-sync-hub's authenticated directed delivery; Bridge stores no blob. Records a durable distribution. An unknown/revoked node is rejected before any delivery. Honours X-Dry-Run. Owner-gated.",
		Category:    "artifacts",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"node_id":          "string (required)",
			"name":             "string",
			"source_ref":       "string (required)",
			"destination_path": "string (required)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"distribution_id": "string",
			"dry_run":         "bool",
			"status":          "DeliveryStatus",
			"delivery_ref":    "string",
		}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing node_id/source_ref/destination_path"},
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "Unknown node"},
			{Status: 409, Code: "failed_precondition", Description: "Node revoked"},
			{Status: 503, Code: "unavailable", Description: "device-sync-hub directed delivery failed"},
		},
		Examples: []module.Example{
			{Name: "Distribute an installer", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.artifacts.ArtifactsService/DistributeArtifact -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"node_id\":\"abc123\",\"name\":\"app-setup.exe\",\"source_ref\":\"file:///tmp/app-setup.exe\",\"destination_path\":\"/opt/app/setup.exe\"}'"},
		},
	},
	{
		ID:          "artifacts_get_distribution",
		Path:        artifactsconnect.ArtifactsServiceGetDistributionProcedure,
		Method:      "POST",
		Summary:     "Get an artifact distribution by id",
		Description: "Returns one durable distribution (reference + status + metadata). Owner-gated.",
		Category:    "artifacts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"distribution": "Distribution"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "No distribution with that id"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get distribution", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.artifacts.ArtifactsService/GetDistribution -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"id\":\"dist-123\"}'"},
		},
	},
	{
		ID:          "artifacts_list_distributions",
		Path:        artifactsconnect.ArtifactsServiceListDistributionsProcedure,
		Method:      "POST",
		Summary:     "List artifact distributions",
		Description: "Returns distributions newest-first, optionally filtered by node. Owner-gated.",
		Category:    "artifacts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"node_id": "string", "limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"distributions": "array<Distribution>"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List distributions", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.artifacts.ArtifactsService/ListDistributions -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "artifacts_upload_run_artifact",
		Path:        artifactsconnect.ArtifactsServiceUploadRunArtifactProcedure,
		Method:      "POST",
		Summary:     "Upload bounded evidence for a dispatched run",
		Description: "Node-facing authenticated upload of a bounded produced artifact. The node may upload only evidence for its own run; the bytes are retained for owner retrieval.",
		Category:    "artifacts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"run_id": "string (required)", "name": "string (required)", "media_type": "string", "data": "bytes (bounded)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"artifact_ref": "string", "size_bytes": "int64"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Invalid or oversized artifact"},
			{Status: 401, Code: "unauthenticated", Description: "Node credential required"},
			{Status: 403, Code: "permission_denied", Description: "Node does not own the run"},
			{Status: 404, Code: "not_found", Description: "Unknown run"},
		},
	},
	{
		ID:          "artifacts_get_run_artifact",
		Path:        artifactsconnect.ArtifactsServiceGetRunArtifactProcedure,
		Method:      "POST",
		Summary:     "Retrieve bounded evidence from a dispatched run",
		Description: "Owner-gated retrieval of a node-produced artifact. The response is suitable for saving plan evidence without a direct node connection.",
		Category:    "artifacts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"run_id": "string (required)", "name": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run_id": "string", "name": "string", "media_type": "string", "data": "bytes", "artifact_ref": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 401, Code: "unauthenticated", Description: "Owner token required"},
			{Status: 404, Code: "not_found", Description: "Produced artifact not found"},
		},
		Examples: []module.Example{
			{Name: "Retrieve screenshot evidence", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.artifacts.ArtifactsService/GetRunArtifact -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"run_id\":\"run-123\",\"name\":\"screenshot.png\"}'"},
		},
	},
	{
		ID:          "artifacts_start_closure_shipment",
		Path:        artifactsconnect.ArtifactsServiceStartClosureShipmentProcedure,
		Method:      "POST",
		Summary:     "Ship one selected dependency closure to a Bridge node",
		Description: "Owner-gated closure shipment bound to an owner-issued onboarding handoff. Bridge resolves the same owner's durable selection receipt and current Machine/Node generation, verifies exact archive inventory, persists intent, and advances one authenticated target state transition at a time. Retries reattach to the same receipt.",
		Category:    "artifacts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"idempotency_key": "string", "target_id": "registered Bridge node and onboarding handoff node ID", "source_ref": "tar or tar.gz file/http source", "destination": "relative target checkout path", "closure": "canonical api-core/coreset dependency closure", "inventory_digest": "sha256 of exact file inventory", "prior_revision": "expected target revision", "new_revision": "decimal onboarding desired_revision", "selection_receipt_ref": "vrooli-onboarding handoff reference", "selection_digest": "onboarding selection digest", "deployment_id": "handoff deployment fence", "machine_id": "handoff machine fence", "node_id": "handoff node fence", "enrollment_generation": "handoff enrollment fence", "desired_revision": "handoff desired revision", "files": "exact path/hash/mode/size manifest"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"shipment_id": "durable Bridge shipment id", "state": "prepared or observed target state", "closure_digest": "BR1 canonical graph digest", "inventory_digest": "source file inventory digest"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Invalid closure, manifest, or source archive"}, {Status: 401, Code: "unauthenticated", Description: "Owner token required"}, {Status: 404, Code: "not_found", Description: "Unknown target node"}, {Status: 409, Code: "failed_precondition", Description: "Revoked/stale target, digest mismatch, or policy rejection"}, {Status: 503, Code: "unavailable", Description: "Target channel is not connected or shipment coordinator is unavailable"}},
	},
	{
		ID:          "artifacts_get_closure_shipment",
		Path:        artifactsconnect.ArtifactsServiceGetClosureShipmentProcedure,
		Method:      "POST",
		Summary:     "Read a durable closure shipment receipt",
		Description: "Owner-gated receipt lookup by shipment ID or idempotency key, including destination-observed file inventory and active revision.",
		Category:    "artifacts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "shipment id or idempotency key (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"receipt": "durable target-bound closure shipment receipt"}},
		Errors:      []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Owner token required"}, {Status: 404, Code: "not_found", Description: "No closure shipment with that ID"}},
	},
	{
		ID:          "artifacts_reconcile_closure_shipment",
		Path:        artifactsconnect.ArtifactsServiceReconcileClosureShipmentProcedure,
		Method:      "POST",
		Summary:     "Reconcile a closure shipment with its target",
		Description: "Owner-gated request for the bound target to re-report its durable shipment receipt after an ambiguous response. Read GetClosureShipment before advancing.",
		Category:    "artifacts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "shipment ID (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"shipment_id": "durable shipment ID", "action": "reconcile", "queued": "bool", "observed_state": "last persisted Bridge state"}},
		Errors:      []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Owner token required"}, {Status: 404, Code: "not_found", Description: "No closure shipment with that ID"}, {Status: 409, Code: "failed_precondition", Description: "Target binding is stale, revoked, or unavailable"}},
	},
	{
		ID:          "artifacts_rollback_closure_shipment",
		Path:        artifactsconnect.ArtifactsServiceRollbackClosureShipmentProcedure,
		Method:      "POST",
		Summary:     "Restore a closure shipment's prior target revision",
		Description: "Owner-gated target request that compare-and-swaps the active revision back to the shipment's prior revision; the authenticated target receipt is the result authority.",
		Category:    "artifacts",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "shipment ID (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"shipment_id": "durable shipment ID", "action": "rollback", "queued": "bool", "observed_state": "last persisted Bridge state"}},
		Errors:      []module.ErrorDesc{{Status: 401, Code: "unauthenticated", Description: "Owner token required"}, {Status: 404, Code: "not_found", Description: "No closure shipment with that ID"}, {Status: 409, Code: "failed_precondition", Description: "No rollbackable state or target binding is stale/revoked"}},
	},
}
