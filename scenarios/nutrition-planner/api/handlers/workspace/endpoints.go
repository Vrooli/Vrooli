package workspace

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace/workspace_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "workspace_list", Path: connect.WorkspaceServiceListWorkspacesProcedure, Method: "POST", Summary: "List owned workspaces", Category: "workspace"},
	{ID: "workspace_create", Path: connect.WorkspaceServiceCreateWorkspaceProcedure, Method: "POST", Summary: "Create a workspace", Category: "workspace"},
	{ID: "workspace_get", Path: connect.WorkspaceServiceGetWorkspaceProcedure, Method: "POST", Summary: "Get an owned workspace", Category: "workspace"},
}
