package workspace

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/workspace/workspace_v1connect"
)

type workspaceTestClient struct {
	workspaceconnect.WorkspaceServiceClient
}

func (workspaceTestClient) GetLayout(context.Context, *connect.Request[workspacev1.GetLayoutRequest]) (*connect.Response[workspacev1.GetLayoutResponse], error) {
	return connect.NewResponse(&workspacev1.GetLayoutResponse{}), nil
}

func (workspaceTestClient) SaveLayout(context.Context, *connect.Request[workspacev1.SaveLayoutRequest]) (*connect.Response[workspacev1.SaveLayoutResponse], error) {
	return connect.NewResponse(&workspacev1.SaveLayoutResponse{}), nil
}

func (workspaceTestClient) UpdatePane(context.Context, *connect.Request[workspacev1.UpdatePaneRequest]) (*connect.Response[workspacev1.UpdatePaneResponse], error) {
	return connect.NewResponse(&workspacev1.UpdatePaneResponse{}), nil
}

func (workspaceTestClient) DeletePane(context.Context, *connect.Request[workspacev1.DeletePaneRequest]) (*connect.Response[workspacev1.DeletePaneResponse], error) {
	return connect.NewResponse(&workspacev1.DeletePaneResponse{}), nil
}

func (workspaceTestClient) CreateGroup(context.Context, *connect.Request[workspacev1.CreateGroupRequest]) (*connect.Response[workspacev1.CreateGroupResponse], error) {
	return connect.NewResponse(&workspacev1.CreateGroupResponse{}), nil
}

func (workspaceTestClient) UpdateGroup(context.Context, *connect.Request[workspacev1.UpdateGroupRequest]) (*connect.Response[workspacev1.UpdateGroupResponse], error) {
	return connect.NewResponse(&workspacev1.UpdateGroupResponse{}), nil
}

func (workspaceTestClient) DeleteGroup(context.Context, *connect.Request[workspacev1.DeleteGroupRequest]) (*connect.Response[workspacev1.DeleteGroupResponse], error) {
	return connect.NewResponse(&workspacev1.DeleteGroupResponse{}), nil
}

func TestHandlersRenderSuccessfulResponses(t *testing.T) {
	body := t.TempDir() + "/body.json"
	if err := os.WriteFile(body, []byte(`{"name":"pane","header_color":"blue","theme_id":"dark","font_size":14,"sort_order":2,"group_id":"g1","supports_messages_view":true,"is_collapsed":true,"color":"red"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &handlers{client: workspaceTestClient{}}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "body-file"}}, Positionals: []cliapp.Positional{{Name: "session-id"}, {Name: "group-id"}}}
	ctx := func(pos map[string]string) cliapp.RunContext {
		return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: schema, Flags: map[string]string{"body-file": body}, Positionals: pos, JSON: true})
	}
	for _, call := range []func(cliapp.RunContext) error{h.layoutGet, h.layoutSave, h.paneUpdate, h.paneDelete, h.groupCreate, h.groupUpdate, h.groupDelete} {
		if err := call(ctx(map[string]string{"session-id": "s1", "group-id": "g1"})); err != nil {
			t.Fatal(err)
		}
	}
}
