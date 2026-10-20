package projects

import (
	"device-studio/internal/module"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects/projects_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "projects_list", Path: projectsconnect.ProjectsServiceListProjectsProcedure, Method: "POST", Summary: "List projects", Description: "Returns a bounded page of local projects.", Category: "projects"},
	{ID: "projects_create", Path: projectsconnect.ProjectsServiceCreateProjectProcedure, Method: "POST", Summary: "Create project", Description: "Creates an empty local project.", Category: "projects"},
	{ID: "projects_get", Path: projectsconnect.ProjectsServiceGetProjectProcedure, Method: "POST", Summary: "Open project", Description: "Returns a local project by id.", Category: "projects"},
	{ID: "projects_rename", Path: projectsconnect.ProjectsServiceRenameProjectProcedure, Method: "POST", Summary: "Rename project", Description: "Renames a project only when expected_revision matches; conflicts return ProjectConflict detail.", Category: "projects"},
}
