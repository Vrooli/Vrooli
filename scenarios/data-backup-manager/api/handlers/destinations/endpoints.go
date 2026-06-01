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
		Description: "Provisions a new kopia repository (filesystem or S3). Encryption is always on; the repository passphrase is generated and stored in vault. A filesystem destination must not point under the protected root.",
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
		CLIMapping: &module.CLIMapping{
			Command: "data-backup-manager destinations create",
			Args:    []string{"--name", "<name>", "--backend", "<backend>", "--location", "<location>"},
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
		CLIMapping: &module.CLIMapping{
			Command: "data-backup-manager destinations get",
			Args:    []string{"<id>"},
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
		CLIMapping: &module.CLIMapping{
			Command: "data-backup-manager destinations list",
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
		CLIMapping: &module.CLIMapping{
			Command: "data-backup-manager destinations update",
			Args:    []string{"<id>", "--cap-bytes", "<bytes>"},
		},
	},
	{
		ID:          "destinations_delete",
		Path:        destinationsconnect.DestinationsServiceDeleteDestinationProcedure,
		Method:      "POST",
		Summary:     "Delete a destination",
		Description: "Removes the destination catalog row. When delete_repository is explicitly set, also removes local resource-kopia metadata and Vault secret refs for the repository.",
		Category:    "destinations",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":                "string (required)",
				"delete_repository": "bool (also remove local resource-kopia metadata and Vault secret refs)",
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
		CLIMapping: &module.CLIMapping{
			Command: "data-backup-manager destinations delete",
			Args:    []string{"<id>"},
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
		CLIMapping: &module.CLIMapping{
			Command: "data-backup-manager destinations usage",
			Args:    []string{"<id>"},
		},
	},
}
