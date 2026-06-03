package audits

import (
	"data-backup-manager/internal/module"

	auditsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/audits/audits_v1connect"
)

// Endpoints describes the audits module's Connect-RPC surface. Paths reference
// generated *Procedure constants so a proto rename breaks this at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "audits_run_snapshot",
		Path:        auditsconnect.AuditsServiceRunSnapshotAuditProcedure,
		Method:      "POST",
		Summary:     "Run a generic snapshot audit",
		Description: "Restores a snapshot to scratch, captures the live target to scratch (read-only on live), walks both trees, and compares them by generic signals only (counts, bytes, path-list/content hashes, per-SQLite integrity/schema). Asynchronous: returns the requested record; poll GetAudit for the verdict.",
		Category:    "audits",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target_id":             "string (required)",
			"destination_id":        "string (required)",
			"snapshot_id":           "string (required)",
			"include_content_hash":  "bool (clients default true)",
			"include_sqlite_checks": "bool (clients default true)",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"audit": "Audit"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing required field"},
			{Status: 404, Code: "not_found", Description: "Unknown target or destination"},
			{Status: 500, Code: "internal", Description: "Engine or persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Run a snapshot audit", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.audits.AuditsService/RunSnapshotAudit -H 'Content-Type: application/json' -d '{\"targetId\":\"t-1\",\"destinationId\":\"dst-1\",\"snapshotId\":\"snap-1\",\"includeContentHash\":true,\"includeSqliteChecks\":true}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "data-backup-manager audits run", Args: []string{"--target", "<id>", "--destination", "<id>", "--snapshot", "<id>"}},
	},
	{
		ID:          "audits_get",
		Path:        auditsconnect.AuditsServiceGetAuditProcedure,
		Method:      "POST",
		Summary:     "Get an audit by id",
		Description: "Returns a single snapshot audit record (status, inventories, comparison) by id.",
		Category:    "audits",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"audit": "Audit"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No audit with that id"}},
		Examples:    []module.Example{{Name: "Get an audit", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.audits.AuditsService/GetAudit -H 'Content-Type: application/json' -d '{\"id\":\"audit-1\"}'"}},
		CLIMapping:  &module.CLIMapping{Command: "data-backup-manager audits get", Args: []string{"<id>"}},
	},
	{
		ID:          "audits_list",
		Path:        auditsconnect.AuditsServiceListAuditsProcedure,
		Method:      "POST",
		Summary:     "List audits",
		Description: "Lists snapshot audit records newest-first, optionally filtered by target id.",
		Category:    "audits",
		Request: &module.Schema{Type: "object", Properties: map[string]string{
			"target_id":  "string (optional)",
			"page_size":  "int32",
			"page_token": "string",
		}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"audits":          "array<Audit>",
			"next_page_token": "string",
		}},
		Examples:   []module.Example{{Name: "List audits", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.audits.AuditsService/ListAudits -H 'Content-Type: application/json' -d '{}'"}},
		CLIMapping: &module.CLIMapping{Command: "data-backup-manager audits list", Args: []string{"--target", "<id>"}},
	},
}
