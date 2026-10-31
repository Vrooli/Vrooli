// Package state exposes persisted desktop scenario state to CLI operators.
package state

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

type stateRPC interface {
	LoadScenarioState(context.Context, *connect.Request[domainv1.LoadScenarioStateRequest]) (*connect.Response[domainv1.StateResponse], error)
}

type Commands struct{ rpc stateRPC }

func New(deps support.Dependencies) *Commands {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(deps.ScenarioApp())
	return &Commands{rpc: domainconnect.NewStateServiceClient(httpClient, baseURL)}
}

func Register(deps support.Dependencies) cliapp.SubcommandGroup {
	c := New(deps)
	scenarioArgs := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "scenario", Required: true, Description: "Scenario name"}}}
	return cliapp.SubcommandGroup{Name: "state", Description: "Inspect persisted desktop scenario state", NeedsAPI: true, Subcommands: []cliapp.Command{
		(cliapp.Command{Name: "get", Description: "Load persisted desktop scenario state", Args: scenarioArgs}).WithPrimitive(c.getPrimitive()),
	}}
}

func (c *Commands) getPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.StateResponse, error) {
		request := &domainv1.LoadScenarioStateRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}
		response, err := c.rpc.LoadScenarioState(context.Background(), connect.NewRequest(request))
		if err != nil {
			return nil, cliapp.WrapAPIError("load scenario state", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.StateResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Scenario state retrieved"}, Results: []string{fmt.Sprintf("Found: %t", response.GetFound())}}
	})
}
