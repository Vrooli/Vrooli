package shortcuts

import (
	shortcutsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shortcuts/shortcuts_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the shortcuts module's public surface. Connect-RPC
// method paths reference generated *Procedure constants so adding or
// renaming an RPC in shortcuts.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "shortcuts_get_effective",
		Path:        shortcutsconnect.ShortcutsServiceGetEffectiveProcedure,
		Method:      "POST",
		Summary:     "Get effective shortcuts",
		Description: "Returns the resolved shortcut list — the highest-priority scope (parent > workspace > service) wins, falling back to built-in defaults if no profiles exist.",
		Category:    "shortcuts",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"shortcuts": "[]Shortcut"},
		},
	},
	{
		ID:          "shortcuts_list_profiles",
		Path:        shortcutsconnect.ShortcutsServiceListProfilesProcedure,
		Method:      "POST",
		Summary:     "List shortcut profiles",
		Description: "Returns every stored shortcut profile across all scopes.",
		Category:    "shortcuts",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"profiles": "[]Profile"},
		},
	},
	{
		ID:          "shortcuts_upsert_profile",
		Path:        shortcutsconnect.ShortcutsServiceUpsertProfileProcedure,
		Method:      "POST",
		Summary:     "Create or update a shortcut profile",
		Description: "Idempotent upsert keyed by id; updated_at is only bumped when content changes.",
		Category:    "shortcuts",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"id":        "string",
				"scope":     "string (service|workspace|parent)",
				"name":      "string",
				"shortcuts": "[]Shortcut",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"profile": "Profile"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Unknown scope, blank id/name, or shortcut entry missing label/command"},
		},
	},
	{
		ID:          "shortcuts_delete_profile",
		Path:        shortcutsconnect.ShortcutsServiceDeleteProfileProcedure,
		Method:      "POST",
		Summary:     "Delete a shortcut profile",
		Description: "Idempotent: succeeds whether the id exists or not.",
		Category:    "shortcuts",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"id": "string"},
		},
	},
}
