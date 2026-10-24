// Package build exposes durable desktop build status to CLI operators.
package build

import (
	"context"
	"fmt"
	"strings"

	"scenario-to-desktop/cli/internal/support"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

type buildRPC interface {
	GetBuild(context.Context, *connect.Request[domainv1.BuildStatusRequest]) (*connect.Response[sharedv1.BuildStatusResponse], error)
}

type Commands struct{ rpc buildRPC }

func New(deps support.Dependencies) *Commands {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(deps.ScenarioApp())
	return &Commands{rpc: domainconnect.NewBuildServiceClient(httpClient, baseURL)}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	args := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "build", Required: true, Description: "Build ID"}}}
	return cliapp.SubcommandGroup{Name: "build", Description: "Inspect desktop build status", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "get", Description: "Show durable desktop build status", Args: args}).WithPrimitive(c.getPrimitive()),
	}}
}

func (c *Commands) getPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*sharedv1.BuildStatusResponse, error) {
		request := &domainv1.BuildStatusRequest{BuildId: strings.TrimSpace(ctx.Positional("build"))}
		response, err := c.rpc.GetBuild(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("get desktop build", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *sharedv1.BuildStatusResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Desktop build retrieved"}, Results: []string{fmt.Sprintf("Status: %s", response.GetStatus())}}
	})
}
