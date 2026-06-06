package safety

import (
	"data-backup-manager/internal/module"

	safetyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety/safety_v1connect"
)

// Endpoints describes the safety module's Connect-RPC surface. Paths reference
// generated *Procedure constants so a proto rename breaks this at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "safety_ensure_destination",
		Path:        safetyconnect.SafetyServiceEnsureSafetyDestinationProcedure,
		Method:      "POST",
		Summary:     "Ensure the Baseline Modes safety destination",
		Description: "Idempotently provisions (or returns the existing) reserved baseline-safety filesystem destination used only for pre-promote safety snapshots. Returns created=true only when this call created it.",
		Category:    "safety",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"cap_bytes": "int64 (optional; 0 = no cap)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"destination": "SafetyDestination", "created": "bool"}},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Runtime root unresolved or destination create failed"},
		},
		Examples: []module.Example{
			{Name: "Ensure safety destination", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.safety.SafetyService/EnsureSafetyDestination -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "data-backup-manager safety ensure-destination"},
	},
	{
		ID:          "safety_backup_scenario_now",
		Path:        safetyconnect.SafetyServiceBackupScenarioNowProcedure,
		Method:      "POST",
		Summary:     "Back up a scenario's targets now",
		Description: "Backs up every target registered under owner=scenario to the safety destination immediately, by building-or-reusing a manual-only ephemeral per-scenario plan and triggering a run. The run executes asynchronously; poll `runs get <id>`.",
		Category:    "safety",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string (required)", "keep_latest": "int32 (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"run_id": "string", "plan_id": "string", "destination_id": "string", "target_count": "int32", "status": "string"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario"},
			{Status: 412, Code: "failed_precondition", Description: "Scenario has no registered targets"},
			{Status: 500, Code: "internal", Description: "Plan/run orchestration failure"},
		},
		Examples: []module.Example{
			{Name: "Back up a scenario now", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.safety.SafetyService/BackupScenarioNow -H 'Content-Type: application/json' -d '{\"scenario\":\"swarm-manager\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "data-backup-manager safety backup-now", Args: []string{"--scenario", "<scenario>"}},
	},
	{
		ID:          "safety_register_scenario_targets",
		Path:        safetyconnect.SafetyServiceRegisterScenarioTargetsProcedure,
		Method:      "POST",
		Summary:     "Derive and register a scenario's backup targets",
		Description: "Derives a scenario's reliably-conventional backup targets from its .vrooli/service.json + on-disk layout (Postgres `vrooli_<scenario>` when declared; the filesystem data dir when present) and idempotently registers them under owner=scenario, so `safety backup-now` works without a hand-run `targets register`. Redis/Qdrant/SQLite are not derivable and are returned in `skipped`.",
		Category:    "safety",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "registered": "RegisteredTarget[]", "skipped": "SkippedTarget[]"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario"},
			{Status: 500, Code: "internal", Description: "Manifest/data-dir inspection or target registration failed"},
		},
		Examples: []module.Example{
			{Name: "Register a scenario's targets", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.safety.SafetyService/RegisterScenarioTargets -H 'Content-Type: application/json' -d '{\"scenario\":\"swarm-manager\"}'"},
		},
		CLIMapping: &module.CLIMapping{Command: "data-backup-manager safety register-targets", Args: []string{"--scenario", "<scenario>"}},
	},
}
