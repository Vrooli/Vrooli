package restores

import (
	"data-backup-manager/internal/module"

	restoresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores/restores_v1connect"
)

// Endpoints describes the restores module's Connect-RPC surface. Paths
// reference generated *Procedure constants so a proto rename breaks this at
// compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "restores_restore_target",
		Path:        restoresconnect.RestoresServiceRestoreTargetProcedure,
		Method:      "POST",
		Summary:     "Restore a snapshot to a location",
		Description: "Restores a snapshot to a caller-chosen filesystem location, then applies the source-kind restore step. Returns the closed restore record.",
		Category:    "restores",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target_id":      "string (required)",
			"destination_id": "string (required)",
			"snapshot_id":    "string (required)",
			"location":       "string (required)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"restore": "Restore"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing required field"},
			{Status: 404, Code: "not_found", Description: "Unknown target or destination"},
			{Status: 500, Code: "internal", Description: "Engine or persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Restore a snapshot", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.restores.RestoresService/RestoreTarget -H 'Content-Type: application/json' -d '{\"targetId\":\"t-1\",\"destinationId\":\"dst-1\",\"snapshotId\":\"snap-1\",\"location\":\"/restore/dest\"}'"},
		},
	},
	{
		ID:          "restores_verify_target",
		Path:        restoresconnect.RestoresServiceVerifyTargetProcedure,
		Method:      "POST",
		Summary:     "Verify a snapshot's restorability",
		Description: "Test-restores a snapshot to a scratch directory, checksums the result, records last-verified-at on success, and cleans up scratch. The verify gate must pass before committed runtime data is removed.",
		Category:    "restores",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target_id":      "string (required)",
			"destination_id": "string (required)",
			"snapshot_id":    "string (required)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"restore": "Restore"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing required field"},
			{Status: 404, Code: "not_found", Description: "Unknown target or destination"},
			{Status: 500, Code: "internal", Description: "Engine or persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Verify a snapshot", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.restores.RestoresService/VerifyTarget -H 'Content-Type: application/json' -d '{\"targetId\":\"t-1\",\"destinationId\":\"dst-1\",\"snapshotId\":\"snap-1\"}'"},
		},
	},
	{
		ID:          "restores_get",
		Path:        restoresconnect.RestoresServiceGetRestoreProcedure,
		Method:      "POST",
		Summary:     "Get a restore by id",
		Description: "Returns a single restore/verify record by id.",
		Category:    "restores",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"restore": "Restore"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No restore with that id"}},
		Examples:    []module.Example{{Name: "Get a restore", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.restores.RestoresService/GetRestore -H 'Content-Type: application/json' -d '{\"id\":\"restore-1\"}'"}},
	},
	{
		ID:          "restores_list",
		Path:        restoresconnect.RestoresServiceListRestoresProcedure,
		Method:      "POST",
		Summary:     "List restores",
		Description: "Lists restore/verify records newest-first, optionally filtered by target id.",
		Category:    "restores",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target_id":  "string (optional)",
			"page_size":  "int32",
			"page_token": "string",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"restores":        "array<Restore>",
			"next_page_token": "string",
		}},
		Examples: []module.Example{{Name: "List restores", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.restores.RestoresService/ListRestores -H 'Content-Type: application/json' -d '{}'"}},
	},
}
