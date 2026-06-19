package workspace

import (
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace/workspace_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the workspace module's public surface. Connect-RPC
// method paths reference generated *Procedure constants so adding or
// renaming an RPC in workspace.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "workspace_get_layout",
		Path:        workspaceconnect.WorkspaceServiceGetLayoutProcedure,
		Method:      "POST",
		Summary:     "Get workspace layout",
		Description: "Returns the active pane, ordered panes, and tab groups.",
		Category:    "workspace",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"active_pane": "string",
				"panes":       "[]Pane",
				"groups":      "[]Group",
			},
		},
	},
	{
		ID:          "workspace_save_layout",
		Path:        workspaceconnect.WorkspaceServiceSaveLayoutProcedure,
		Method:      "POST",
		Summary:     "Save workspace layout",
		Description: "Persists pane ordering and the active pane selection.",
		Category:    "workspace",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"active_pane": "string",
				"pane_order":  "[]string",
			},
		},
	},
	{
		ID:          "workspace_update_pane",
		Path:        workspaceconnect.WorkspaceServiceUpdatePaneProcedure,
		Method:      "POST",
		Summary:     "Create or update a pane",
		Description: "Upserts a pane keyed by session_id. Only fields with has_* = true are applied.",
		Category:    "workspace",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"pane": "Pane"},
		},
	},
	{
		ID:          "workspace_delete_pane",
		Path:        workspaceconnect.WorkspaceServiceDeletePaneProcedure,
		Method:      "POST",
		Summary:     "Delete a pane",
		Description: "Idempotent: succeeds whether the session_id exists or not.",
		Category:    "workspace",
	},
	{
		ID:          "workspace_create_group",
		Path:        workspaceconnect.WorkspaceServiceCreateGroupProcedure,
		Method:      "POST",
		Summary:     "Create a tab group",
		Description: "Creates a new tab group with the given name and display color.",
		Category:    "workspace",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"group": "Group"},
		},
	},
	{
		ID:          "workspace_update_group",
		Path:        workspaceconnect.WorkspaceServiceUpdateGroupProcedure,
		Method:      "POST",
		Summary:     "Update a tab group",
		Description: "Updates fields on an existing tab group. Only fields with has_* = true are applied.",
		Category:    "workspace",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"group": "Group"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "Group id does not exist"},
		},
	},
	{
		ID:          "workspace_delete_group",
		Path:        workspaceconnect.WorkspaceServiceDeleteGroupProcedure,
		Method:      "POST",
		Summary:     "Delete a tab group",
		Description: "Idempotent. Panes in the group have their group_id cleared.",
		Category:    "workspace",
	},
}
