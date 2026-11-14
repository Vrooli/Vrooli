package workspace

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/workspace/workspace_v1connect"
)

type handlers struct {
	client workspaceconnect.WorkspaceServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: workspaceconnect.NewWorkspaceServiceClient(httpClient, baseURL)}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListWorkspaces(context.Background(), connect.NewRequest(&workspacev1.ListWorkspacesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list workspaces", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no workspace response")
	}
	results := make([]string, 0, len(resp.Msg.Workspaces))
	for _, item := range resp.Msg.Workspaces {
		if item == nil {
			continue
		}
		results = append(results, fmt.Sprintf("%s — %s [revision=%d]", item.Id, item.Name, item.Revision))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d workspace(s).", len(resp.Msg.Workspaces))}, ResultsHeading: "Workspaces", Results: results})
}
