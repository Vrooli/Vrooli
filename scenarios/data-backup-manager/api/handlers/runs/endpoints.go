package runs

import (
	"data-backup-manager/internal/module"

	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/runs/runs_v1connect"
)

// Endpoints describes the runs module's Connect-RPC surface. Paths reference
// generated *Procedure constants so a proto rename breaks this at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "runs_preflight",
		Path:        runsconnect.RunsServicePreflightRunProcedure,
		Method:      "POST",
		Summary:     "Check backup plan readiness",
		Description: "Performs read-only destination, credential, source-adapter, and source-path checks before any backup fan-out.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"plan_id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"ready": "bool", "incidents": "array<FailureCause>"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing plan_id"},
			{Status: 404, Code: "not_found", Description: "No plan with that id"},
		},
		Examples: []module.Example{{Name: "Preflight a plan", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.runs.RunsService/PreflightRun -H 'Content-Type: application/json' -d '{\"planId\":\"plan-1\"}'"}},
	},
	{
		ID:          "runs_trigger",
		Path:        runsconnect.RunsServiceTriggerRunProcedure,
		Method:      "POST",
		Summary:     "Trigger a backup run for a plan",
		Description: "Executes a plan now: per target × destination, capture → cap-check → snapshot → retention → record outcome. Returns the closed run with per-target outcomes.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"plan_id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "Run"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing plan_id"},
			{Status: 404, Code: "not_found", Description: "No plan with that id"},
			{Status: 500, Code: "internal", Description: "Run persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Trigger a run", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.runs.RunsService/TriggerRun -H 'Content-Type: application/json' -d '{\"planId\":\"plan-1\"}'"},
		},
	},
	{
		ID:          "runs_get",
		Path:        runsconnect.RunsServiceGetRunProcedure,
		Method:      "POST",
		Summary:     "Get a run by id",
		Description: "Returns a run with its per-target outcomes.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run": "Run"}},
		Errors:      []module.ErrorDesc{{Status: 404, Code: "not_found", Description: "No run with that id"}},
		Examples:    []module.Example{{Name: "Get a run", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.runs.RunsService/GetRun -H 'Content-Type: application/json' -d '{\"id\":\"run-1\"}'"}},
	},
	{
		ID:          "runs_list",
		Path:        runsconnect.RunsServiceListRunsProcedure,
		Method:      "POST",
		Summary:     "List runs",
		Description: "Lists runs newest-first, optionally filtered by plan id.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"plan_id": "string (optional)", "page_size": "int32", "page_token": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"runs": "array<Run>", "next_page_token": "string"}},
		Examples:    []module.Example{{Name: "List runs", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.runs.RunsService/ListRuns -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "runs_target_status",
		Path:        runsconnect.RunsServiceListTargetStatusProcedure,
		Method:      "POST",
		Summary:     "List last-success per target",
		Description: "Returns the last-success and last-run rollup per target, the catalog's recovery-readiness view.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"owner": "string (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"statuses": "array<TargetStatus>"}},
		Examples:    []module.Example{{Name: "Target status", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.runs.RunsService/ListTargetStatus -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "runs_browse_snapshot",
		Path:        runsconnect.RunsServiceBrowseSnapshotProcedure,
		Method:      "POST",
		Summary:     "Browse a snapshot's contents",
		Description: "Lists entries within a snapshot in a destination repository.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"destination_id": "string (required)", "snapshot_id": "string (required)", "path": "string (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"entries": "array<SnapshotEntry>"}},
		Examples:    []module.Example{{Name: "Browse snapshot", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.runs.RunsService/BrowseSnapshot -H 'Content-Type: application/json' -d '{\"destinationId\":\"dst-1\",\"snapshotId\":\"snap-1\"}'"}},
	},
	{
		ID:          "runs_stats",
		Path:        runsconnect.RunsServiceGetRunStatsProcedure,
		Method:      "POST",
		Summary:     "Aggregate run performance metrics",
		Description: "Reports counts, success rate, p50/p95 wall-clock duration, logical bytes per run, and throughput over the recent run-history window, optionally scoped to one plan.",
		Category:    "runs",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"plan_id": "string (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"stats": "RunStats"}},
		Examples:    []module.Example{{Name: "Run stats", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.runs.RunsService/GetRunStats -H 'Content-Type: application/json' -d '{}'"}},
	},
}
