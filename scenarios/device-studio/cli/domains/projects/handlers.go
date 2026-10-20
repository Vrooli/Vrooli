package projects

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects/projects_v1connect"
	"strconv"
	"strings"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client projectsconnect.ProjectsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: projectsconnect.NewProjectsServiceClient(httpClient, base)}
}
func (h *handlers) listCall(ctx cliapp.OperationContext) (*projectsv1.ListProjectsResponse, error) {
	size := int32(50)
	if raw := strings.TrimSpace(ctx.Flag("page-size")); raw != "" {
		n, e := strconv.ParseInt(raw, 10, 32)
		if e != nil {
			return nil, fmt.Errorf("invalid page-size: %w", e)
		}
		size = int32(n)
	}
	resp, err := h.client.ListProjects(context.Background(), connect.NewRequest(&projectsv1.ListProjectsRequest{PageSize: size, PageToken: ctx.Flag("page-token")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list projects", err, nil)
	}
	return resp.Msg, nil
}
func (h *handlers) listReport(_ cliapp.OperationContext, msg *projectsv1.ListProjectsResponse) cliapp.ListReport {
	rows := make([]string, 0, len(msg.Projects))
	for _, p := range msg.Projects {
		rows = append(rows, format(p))
	}
	hints := []string{}
	if msg.NextPageToken != "" {
		hints = append(hints, fmt.Sprintf("`projects list --page-token %s` — next page", msg.NextPageToken))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d project(s).", len(rows))}, ResultsHeading: "Projects", Results: rows, RetrievalHints: hints}
}
func (h *handlers) createCall(ctx cliapp.OperationContext) (*projectsv1.CreateProjectResponse, error) {
	resp, err := h.client.CreateProject(context.Background(), connect.NewRequest(&projectsv1.CreateProjectRequest{Name: ctx.Flag("name")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create project", err, nil)
	}
	if resp.Msg.Project == nil {
		return nil, fmt.Errorf("server returned no project")
	}
	return resp.Msg, nil
}
func (h *handlers) createReport(_ cliapp.OperationContext, msg *projectsv1.CreateProjectResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created project %s.", msg.Project.Id)}, Changes: []string{format(msg.Project)}, NextCommand: []string{fmt.Sprintf("`projects get %s` — open this project", msg.Project.Id), "`projects list` — show projects"}}
}
func (h *handlers) getCall(ctx cliapp.OperationContext) (*projectsv1.GetProjectResponse, error) {
	id := ctx.Positional("id")
	resp, err := h.client.GetProject(context.Background(), connect.NewRequest(&projectsv1.GetProjectRequest{Id: id}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get project", err, nil)
	}
	if resp.Msg.Project == nil {
		return nil, fmt.Errorf("server returned no project")
	}
	return resp.Msg, nil
}
func (h *handlers) getReport(_ cliapp.OperationContext, msg *projectsv1.GetProjectResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{"Opened project."}, ResultsHeading: "Project", Results: []string{format(msg.Project)}}
}
func (h *handlers) renameCall(ctx cliapp.OperationContext) (*projectsv1.RenameProjectResponse, error) {
	rev, err := strconv.ParseInt(ctx.Flag("expected-revision"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid expected-revision: %w", err)
	}
	resp, err := h.client.RenameProject(context.Background(), connect.NewRequest(&projectsv1.RenameProjectRequest{Id: ctx.Positional("id"), Name: ctx.Flag("name"), ExpectedRevision: rev}))
	if err != nil {
		if conflict, ok := projectConflict(err); ok {
			return nil, conflict
		}
		return nil, cliapp.WrapAPIError("rename project", err, nil)
	}
	return resp.Msg, nil
}

func projectConflict(err error) (error, bool) {
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Code() != connect.CodeAborted {
		return nil, false
	}
	for _, detail := range connectErr.Details() {
		message, detailErr := detail.Value()
		if detailErr != nil {
			continue
		}
		conflict, ok := message.(*projectsv1.ProjectConflict)
		if !ok {
			continue
		}
		return fmt.Errorf("project %s revision conflict: expected revision %d, current revision %d", conflict.ProjectId, conflict.ExpectedRevision, conflict.CurrentRevision), true
	}
	return nil, false
}

func (h *handlers) renameReport(_ cliapp.OperationContext, msg *projectsv1.RenameProjectResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Renamed project %s.", msg.Project.Id)}, Changes: []string{format(msg.Project)}}
}
func format(p *projectsv1.Project) string {
	if p == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%s — %s [revision=%d updated=%s]", p.Id, p.Name, p.Revision, p.UpdatedAt.AsTime().Format("2006-01-02T15:04:05Z07:00"))
}
