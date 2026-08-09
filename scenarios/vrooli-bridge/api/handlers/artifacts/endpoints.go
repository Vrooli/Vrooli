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
		Description: "Ships a non-git artifact (a built installer, a large fixture) to a node through device-sync-hub's directed delivery — bridge orchestrates, device-sync-hub moves the bytes (bridge stores no blob). Records a durable distribution. An unknown/revoked node is rejected before any delivery. Honours X-Dry-Run. Owner-gated.",
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
			{Name: "Distribute an installer", Curl: "curl http://localhost:${API_PORT}/vrooli.vrooli_bridge.v1.artifacts.ArtifactsService/DistributeArtifact -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' -d '{\"node_id\":\"abc123\",\"name\":\"app-setup.exe\",\"source_ref\":\"blob://builds/app-setup.exe\",\"destination_path\":\"/opt/app/setup.exe\"}'"},
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
}
