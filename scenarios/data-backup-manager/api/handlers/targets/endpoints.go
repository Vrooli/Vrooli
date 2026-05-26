package targets

import (
	"data-backup-manager/internal/module"

	targetsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets/targets_v1connect"
)

// Endpoints is the machine-readable description of the targets module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so adding or renaming an RPC in targets.proto breaks this file at
// compile time. The global parity test (TestProtoConnectParity in
// internal/modules/registry_test.go) asserts every rpc has exactly one entry
// here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "targets_register",
		Path:        targetsconnect.TargetsServiceRegisterTargetProcedure,
		Method:      "POST",
		Summary:     "Register (upsert) a backup target",
		Description: "Idempotently registers a backup source keyed by (owner, name). Re-registering with the same spec is a no-op; a changed spec updates in place. Scenarios call this at their lifecycle.",
		Category:    "targets",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"owner":       "string (required)",
				"name":        "string (required)",
				"source_kind": "SourceKind (required, non-unspecified)",
				"locator":     "string (required)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"target": "Target"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing owner/name/locator or unspecified source kind"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Register a filesystem target", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.targets.TargetsService/RegisterTarget -H 'Content-Type: application/json' -d '{\"owner\":\"prompt-manager\",\"name\":\"store\",\"sourceKind\":\"SOURCE_KIND_FILESYSTEM\",\"locator\":\"store/teams\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "data-backup-manager targets register",
			Args:    []string{"--owner", "<owner>", "--name", "<name>", "--kind", "<kind>", "--locator", "<locator>"},
		},
	},
	{
		ID:          "targets_deregister",
		Path:        targetsconnect.TargetsServiceDeregisterTargetProcedure,
		Method:      "POST",
		Summary:     "Deregister a backup target",
		Description: "Removes a target by (owner, name). The catalog stays reconstructable because owning scenarios re-register on boot.",
		Category:    "targets",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"owner": "string (required)", "name": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"removed": "boolean"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing owner or name"},
			{Status: 500, Code: "internal", Description: "Repository delete failure"},
		},
		Examples: []module.Example{
			{Name: "Deregister a target", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.targets.TargetsService/DeregisterTarget -H 'Content-Type: application/json' -d '{\"owner\":\"prompt-manager\",\"name\":\"store\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "data-backup-manager targets deregister",
			Args:    []string{"--owner", "<owner>", "--name", "<name>"},
		},
	},
	{
		ID:          "targets_get",
		Path:        targetsconnect.TargetsServiceGetTargetProcedure,
		Method:      "POST",
		Summary:     "Get a target by id",
		Description: "Returns the target matching the request id.",
		Category:    "targets",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"target": "Target"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No target with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get a target", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.targets.TargetsService/GetTarget -H 'Content-Type: application/json' -d '{\"id\":\"tgt-1\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "data-backup-manager targets get",
			Args:    []string{"<id>"},
		},
	},
	{
		ID:          "targets_list",
		Path:        targetsconnect.TargetsServiceListTargetsProcedure,
		Method:      "POST",
		Summary:     "List backup targets",
		Description: "Lists registered targets, optionally filtered by owner, with cursor pagination.",
		Category:    "targets",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"owner":      "string (optional filter)",
				"page_size":  "int32",
				"page_token": "string",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"targets":         "array<Target>",
				"next_page_token": "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List all targets", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.targets.TargetsService/ListTargets -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "data-backup-manager targets list",
			Args:    []string{"--owner", "<owner>"},
		},
	},
}
