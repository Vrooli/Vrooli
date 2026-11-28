// Package projects mounts the Content Desk Projects Connect surface.
package projects

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"content-desk/internal/module"
	internalprojects "content-desk/internal/projects"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/projects"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/projects/projects_v1connect"
)

type handler struct{ repo internalprojects.Repository }

var _ projectsconnect.ProjectsServiceHandler = handler{}

func (h handler) ListProjects(ctx context.Context, _ *connect.Request[projectsv1.ListProjectsRequest]) (*connect.Response[projectsv1.ListProjectsResponse], error) {
	items, err := h.repo.ListActive(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	response := &projectsv1.ListProjectsResponse{}
	for _, p := range items {
		response.Projects = append(response.Projects, projectMessage(p))
	}
	return connect.NewResponse(response), nil
}

func (h handler) CreateProject(ctx context.Context, request *connect.Request[projectsv1.CreateProjectRequest]) (*connect.Response[projectsv1.CreateProjectResponse], error) {
	p, err := h.repo.Create(ctx, request.Msg.GetName())
	if err != nil {
		if err == internalprojects.ErrInvalidName {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create project: %w", err))
	}
	return connect.NewResponse(&projectsv1.CreateProjectResponse{Project: projectMessage(p)}), nil
}

func projectMessage(p internalprojects.Project) *projectsv1.Project {
	return &projectsv1.Project{Id: p.ID, Name: p.Name, Status: p.Status}
}

func Module(db *database.RoutedDB) module.Module {
	path, h := projectsconnect.NewProjectsServiceHandler(handler{repo: internalprojects.NewSQLiteRepository(db)})
	return module.Module{Name: "projects", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h}) }, Endpoints: Endpoints}
}

func Schema() string { return internalprojects.Schema() }

var Endpoints = []module.EndpointDescriptor{
	{ID: "projects_list", Path: projectsconnect.ProjectsServiceListProjectsProcedure, Method: "POST", Summary: "List active editorial projects", Category: "projects"},
	{ID: "projects_create", Path: projectsconnect.ProjectsServiceCreateProjectProcedure, Method: "POST", Summary: "Create an active editorial project", Category: "projects"},
}
