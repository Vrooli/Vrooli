package destinations

import (
	"data-backup-manager/internal/module"

	destinationsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations/destinations_v1connect"
)

// Endpoints is the machine-readable description of the destinations module's
// public surface. Connect-RPC method paths reference the generated *Procedure
// constants, so adding or renaming an RPC in destinations.proto breaks this
// file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "destinations_create",
		Path:        destinationsconnect.DestinationsServiceCreateDestinationProcedure,
		Method:      "POST",
		Summary:     "Create a backup destination",
		Description: "Provisions a new kopia repository (filesystem or S3). Encryption is always on; the repository passphrase and S3/backend credentials are stored in the credential authority. A filesystem destination must not point under the protected root.",
		Category:    "destinations",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"name":         "string (required)",
				"backend_kind": "BackendKind (required, non-unspecified)",
				"location":     "string (required)",
				"cap_bytes":    "int64 (optional, 0 = no cap)",
				"cap_policy":   "CapPolicy (optional, defaults to ALERT_BLOCK)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"destination": "Destination"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing name/location, unspecified backend, or separate-root violation"},
			{Status: 500, Code: "internal", Description: "Repository creation failure"},
		},
		Examples: []module.Example{
			{Name: "Create a filesystem destination", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.destinations.DestinationsService/CreateDestination -H 'Content-Type: application/json' -d '{\"name\":\"local-backup\",\"backendKind\":\"BACKEND_KIND_FILESYSTEM\",\"location\":\"/mnt/backup\"}'"},
		},
	},
	{
		ID:          "destinations_get",
		Path:        destinationsconnect.DestinationsServiceGetDestinationProcedure,
		Method:      "POST",
		Summary:     "Get a destination by id",
		Description: "Returns the destination matching the request id.",
		Category:    "destinations",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"destination": "Destination"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No destination with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "Get a destination", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.destinations.DestinationsService/GetDestination -H 'Content-Type: application/json' -d '{\"id\":\"dst-1\"}'"},
		},
	},
	{
		ID:          "destinations_list",
		Path:        destinationsconnect.DestinationsServiceListDestinationsProcedure,
		Method:      "POST",
		Summary:     "List backup destinations",
		Description: "Lists registered destinations ordered by name with cursor pagination.",
		Category:    "destinations",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"page_size":  "int32",
				"page_token": "string",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"destinations":    "array<Destination>",
				"next_page_token": "string",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List all destinations", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.destinations.DestinationsService/ListDestinations -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "destinations_update",
		Path:        destinationsconnect.DestinationsServiceUpdateDestinationProcedure,
		Method:      "POST",
		Summary:     "Update a destination's cap settings",
		Description: "Updates cap_bytes and/or cap_policy for the given destination. Other fields are immutable after creation.",
		Category:    "destinations",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":         "string (required)",
				"cap_bytes":  "int64 (0 = no cap)",
				"cap_policy": "CapPolicy",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"destination": "Destination"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 404, Code: "not_found", Description: "No destination with that id exists"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Set a 10 GiB cap", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.destinations.DestinationsService/UpdateDestination -H 'Content-Type: application/json' -d '{\"id\":\"dst-1\",\"capBytes\":10737418240}'"},
		},
	},
	{
		ID:          "destinations_delete",
		Path:        destinationsconnect.DestinationsServiceDeleteDestinationProcedure,
		Method:      "POST",
		Summary:     "Delete a destination",
		Description: "Removes the destination catalog row. When delete_repository is explicitly set, also removes local resource-kopia metadata and credential-authority refs for the repository.",
		Category:    "destinations",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":                "string (required)",
				"delete_repository": "bool (also remove local resource-kopia metadata and credential-authority refs)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"removed": "boolean"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 500, Code: "internal", Description: "Repository delete failure"},
		},
		Examples: []module.Example{
			{Name: "Delete a destination", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.destinations.DestinationsService/DeleteDestination -H 'Content-Type: application/json' -d '{\"id\":\"dst-1\"}'"},
		},
	},
	{
		ID:          "destinations_get_usage",
		Path:        destinationsconnect.DestinationsServiceGetDestinationUsageProcedure,
		Method:      "POST",
		Summary:     "Get destination usage",
		Description: "Returns current usage bytes, cap, usage state (WITHIN/NEAR/OVER), and cap policy for the given destination.",
		Category:    "destinations",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string (required)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"usage_bytes": "int64",
				"cap_bytes":   "int64",
				"usage_state": "UsageState",
				"cap_policy":  "CapPolicy",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No destination with that id exists"},
			{Status: 500, Code: "internal", Description: "Engine stats failure"},
		},
		Examples: []module.Example{
			{Name: "Get destination usage", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.destinations.DestinationsService/GetDestinationUsage -H 'Content-Type: application/json' -d '{\"id\":\"dst-1\"}'"},
		},
	},
	{
		ID:          "destinations_analyze",
		Path:        destinationsconnect.DestinationsServiceAnalyzeDestinationProcedure,
		Method:      "POST",
		Summary:     "Analyze destination readiness",
		Description: "Inspects a mounted filesystem location read-only and returns structured readiness checks, device identity, filesystem suitability, content warnings, and a recommended backup subdirectory.",
		Category:    "destinations",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"location":                "string (required)",
				"proposed_subdir":         "string (optional)",
				"selected_target_bytes":   "int64 (optional)",
				"retention_copies":        "int32 (optional)",
				"cross_platform_required": "boolean",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"report": "DestinationReadinessReport"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing location or location is not under a mounted volume"},
			{Status: 500, Code: "internal", Description: "Read-only inspection failure"},
		},
		Examples: []module.Example{
			{Name: "Analyze a mounted USB drive", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.destinations.DestinationsService/AnalyzeDestination -H 'Content-Type: application/json' -d '{\"location\":\"/media/user/USB\"}'"},
		},
	},
	{
		ID:          "destinations_prepare_plan",
		Path:        destinationsconnect.DestinationsServicePlanDestinationPreparationProcedure,
		Method:      "POST",
		Summary:     "Plan destination preparation",
		Description: "Creates a non-mutating preparation plan bound to the observed device identity. The plan includes the exact confirmation phrase and whether the action is supported by this platform.",
		Category:    "destinations",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"location":           "string (required)",
				"action":             "PreparationAction (required)",
				"desired_subdir":     "string",
				"desired_label":      "string",
				"desired_filesystem": "string",
				"expected_identity":  "DestinationDeviceIdentity",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"plan": "DestinationPreparationPlan"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing location/action or invalid subdirectory"},
			{Status: 412, Code: "failed_precondition", Description: "Device identity mismatch or protected-path overlap"},
			{Status: 500, Code: "internal", Description: "Read-only inspection failure"},
		},
		Examples: []module.Example{
			{Name: "Plan a backup subdirectory", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.destinations.DestinationsService/PlanDestinationPreparation -H 'Content-Type: application/json' -d '{\"location\":\"/media/user/USB\",\"action\":\"PREPARATION_ACTION_CREATE_SUBDIR\",\"desiredSubdir\":\"vrooli-backups\"}'"},
		},
	},
	{
		ID:          "destinations_prepare_execute",
		Path:        destinationsconnect.DestinationsServiceExecuteDestinationPreparationProcedure,
		Method:      "POST",
		Summary:     "Execute destination preparation",
		Description: "Runs or dry-runs a preparation plan only after confirmation and identity guards pass. Destructive actions additionally require an explicit data-loss acknowledgement.",
		Category:    "destinations",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"plan":                  "DestinationPreparationPlan (required)",
				"confirmation":          "string",
				"dry_run":               "boolean",
				"acknowledge_data_loss": "boolean",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"dry_run":            "boolean",
				"action":             "PreparationAction",
				"location":           "string",
				"post_action_report": "DestinationReadinessReport",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing plan"},
			{Status: 412, Code: "failed_precondition", Description: "Unsupported action, missing confirmation, data-loss acknowledgement missing, or device identity changed"},
			{Status: 500, Code: "internal", Description: "Preparation execution failure"},
		},
		Examples: []module.Example{
			{Name: "Dry-run a preparation plan", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.destinations.DestinationsService/ExecuteDestinationPreparation -H 'Content-Type: application/json' -d '{\"plan\":{\"id\":\"plan-id\"},\"dryRun\":true}'"},
		},
	},
}
