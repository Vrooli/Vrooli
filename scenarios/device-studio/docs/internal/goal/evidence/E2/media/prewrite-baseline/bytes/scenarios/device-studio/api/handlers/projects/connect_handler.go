package projects

import (
	"connectrpc.com/connect"
	"context"
	domain "device-studio/internal/projects"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	"log"
)

type Deps struct {
	Service domain.Service
	Logger  *log.Logger
}
type handler struct{ deps Deps }

func NewConnectHandler(d Deps) *handler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &handler{deps: d}
}
func (h *handler) ListProjects(ctx context.Context, req *connect.Request[projectsv1.ListProjectsRequest]) (*connect.Response[projectsv1.ListProjectsResponse], error) {
	rows, next, err := h.deps.Service.List(ctx, req.Msg.PageSize, req.Msg.PageToken)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	out := &projectsv1.ListProjectsResponse{Projects: make([]*projectsv1.Project, 0, len(rows)), NextPageToken: next}
	for _, p := range rows {
		out.Projects = append(out.Projects, toProto(p))
	}
	return connect.NewResponse(out), nil
}
func (h *handler) CreateProject(ctx context.Context, req *connect.Request[projectsv1.CreateProjectRequest]) (*connect.Response[projectsv1.CreateProjectResponse], error) {
	p, err := h.deps.Service.Create(ctx, domain.CreateInput{Name: req.Msg.Name})
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&projectsv1.CreateProjectResponse{Project: toProto(p)}), nil
}
func (h *handler) GetProject(ctx context.Context, req *connect.Request[projectsv1.GetProjectRequest]) (*connect.Response[projectsv1.GetProjectResponse], error) {
	p, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&projectsv1.GetProjectResponse{Project: toProto(p)}), nil
}
func (h *handler) RenameProject(ctx context.Context, req *connect.Request[projectsv1.RenameProjectRequest]) (*connect.Response[projectsv1.RenameProjectResponse], error) {
	p, err := h.deps.Service.Rename(ctx, domain.RenameInput{ID: req.Msg.Id, Name: req.Msg.Name, ExpectedRevision: req.Msg.ExpectedRevision})
	if err != nil {
		return nil, toConnectError(err, h.deps.Logger)
	}
	return connect.NewResponse(&projectsv1.RenameProjectResponse{Project: toProto(p)}), nil
}
