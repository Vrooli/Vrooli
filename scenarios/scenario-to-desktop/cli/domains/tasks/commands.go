// Package tasks exposes durable desktop investigation tasks to CLI operators.
package tasks

import (
	"context"
	"fmt"
	"scenario-to-desktop/cli/internal/support"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
)

type taskRPC interface {
	ListTasks(context.Context, *connect.Request[domainv1.ListTasksRequest]) (*connect.Response[domainv1.ListTasksResponse], error)
}

type Commands struct{ rpc taskRPC }

func New(deps support.Dependencies) *Commands {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(deps.ScenarioApp())
	return &Commands{rpc: domainconnect.NewTaskServiceClient(httpClient, baseURL)}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	args := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "pipeline", Required: true, Description: "Pipeline ID"}}}
	return cliapp.SubcommandGroup{Name: "tasks", Description: "Inspect desktop investigation tasks", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "list", Description: "List tasks for a desktop pipeline", Args: args}).WithPrimitive(c.listPrimitive()),
	}}
}

func (c *Commands) listPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.ListTasksResponse, error) {
		request := &domainv1.ListTasksRequest{PipelineId: strings.TrimSpace(ctx.Positional("pipeline"))}
		response, err := c.rpc.ListTasks(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("list desktop tasks", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.ListTasksResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Desktop tasks retrieved"}, Results: []string{fmt.Sprintf("Tasks: %d", len(response.GetTasks()))}}
	})
}
