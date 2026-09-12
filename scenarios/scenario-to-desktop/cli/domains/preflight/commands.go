// Package preflight exposes durable preflight job status to CLI operators.
package preflight

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
)

type preflightRPC interface {
	GetPreflightJob(context.Context, *connect.Request[domainv1.GetPreflightJobRequest]) (*connect.Response[domainv1.JobStatusResponse], error)
}

type Commands struct{ rpc preflightRPC }

func New(deps support.Dependencies) *Commands {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(deps.ScenarioApp())
	return &Commands{rpc: domainconnect.NewPreflightServiceClient(httpClient, baseURL)}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	args := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "job", Required: true, Description: "Preflight job ID"}}}
	return cliapp.SubcommandGroup{Name: "preflight", Description: "Inspect desktop preflight jobs", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "get", Description: "Show durable preflight job status", Args: args}).WithPrimitive(c.getPrimitive()),
	}}
}

func (c *Commands) getPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.JobStatusResponse, error) {
		request := &domainv1.GetPreflightJobRequest{JobId: strings.TrimSpace(ctx.Positional("job"))}
		response, err := c.rpc.GetPreflightJob(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("get preflight job", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.JobStatusResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Preflight job retrieved"}, Results: []string{fmt.Sprintf("Status: %s", response.GetStatus())}}
	})
}
