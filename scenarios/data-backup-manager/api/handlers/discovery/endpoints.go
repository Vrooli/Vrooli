package discovery

import (
	"data-backup-manager/internal/module"

	discoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/discovery/discovery_v1connect"
)

// Endpoints is the machine-readable description of the discovery module's public
// surface. Connect-RPC method paths reference the generated *Procedure
// constants, so adding or renaming an RPC in discovery.proto breaks this file at
// compile time. The global parity test asserts every rpc has exactly one entry
// here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "discovery_target_suggestions",
		Path:        discoveryconnect.DiscoveryServiceListTargetSuggestionsProcedure,
		Method:      "POST",
		Summary:     "List target suggestions",
		Description: "Scans well-known Vrooli runtime state (~/.vrooli) read-only and returns ranked, non-dismissed sources worth protecting that are not yet registered. Accept one by calling TargetsService.RegisterTarget with its owner/name/source_kind/locator.",
		Category:    "discovery",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"suggestions": "array<TargetSuggestion>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Scan or catalog read failure"},
		},
		Examples: []module.Example{
			{Name: "List target suggestions", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.discovery.DiscoveryService/ListTargetSuggestions -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "discovery_destination_suggestions",
		Path:        discoveryconnect.DiscoveryServiceListDestinationSuggestionsProcedure,
		Method:      "POST",
		Summary:     "List destination suggestions",
		Description: "Enumerates mounted volumes (OS-agnostic) and returns ranked, non-dismissed places to back up to, removable/external drives first, with the separate-root rule applied. Accept one by calling DestinationsService.CreateDestination with a filesystem backend at its location.",
		Category:    "discovery",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"suggestions": "array<DestinationSuggestion>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Volume scan or catalog read failure"},
		},
		Examples: []module.Example{
			{Name: "List destination suggestions", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.discovery.DiscoveryService/ListDestinationSuggestions -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "discovery_dismiss",
		Path:        discoveryconnect.DiscoveryServiceDismissSuggestionProcedure,
		Method:      "POST",
		Summary:     "Dismiss a suggestion",
		Description: "Hides a suggestion permanently by its stable id. Idempotent — re-dismissing is a no-op success. Dismissed suggestions never reappear on rescans.",
		Category:    "discovery",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"dismissed": "boolean"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id"},
			{Status: 500, Code: "internal", Description: "Dismissal write failure"},
		},
		Examples: []module.Example{
			{Name: "Dismiss a suggestion", Curl: "curl http://localhost:${API_PORT}/vrooli.data_backup_manager.v1.discovery.DiscoveryService/DismissSuggestion -H 'Content-Type: application/json' -d '{\"id\":\"b0e3a8fa79aefd10\"}'"},
		},
	},
}
