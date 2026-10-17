package workspace

import (
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/workspace/workspace_v1connect"
	"personal-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "workspace_get_profile", Path: v.WorkspaceServiceGetProfileProcedure, Method: "POST", Summary: "Get planning profile", Description: "Reads the local timezone, capacity, reserve, and focus preferences.", Category: "workspace"},
	{ID: "workspace_update_profile", Path: v.WorkspaceServiceUpdateProfileProcedure, Method: "POST", Summary: "Update planning profile", Description: "Updates local planning profile preferences with revision protection.", Category: "workspace"},
	{ID: "workspace_list_availability", Path: v.WorkspaceServiceListAvailabilityProcedure, Method: "POST", Summary: "List availability", Description: "Reads weekly availability windows and date exceptions.", Category: "workspace"},
	{ID: "workspace_replace_availability", Path: v.WorkspaceServiceReplaceAvailabilityProcedure, Method: "POST", Summary: "Replace availability", Description: "Replaces availability with revision protection; protected time remains explicit.", Category: "workspace"},
}
