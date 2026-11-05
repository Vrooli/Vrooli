package scenarios

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"
	scenariosconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios/scenarios_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client scenariosconnect.ScenariosServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{core: core, client: scenariosconnect.NewScenariosServiceClient(httpClient, baseURL)}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListScenarios(context.Background(), connect.NewRequest(&scenariosv1.ListScenariosRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list scenarios", err, nil)
	}
	results := make([]string, 0, len(resp.Msg.Scenarios))
	for _, s := range resp.Msg.Scenarios {
		results = append(results, fmt.Sprintf("%s | %s | flows=%d | %s", s.Id, s.DisplayName, s.FlowCount, s.Path))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d scenario(s) under %s.", len(resp.Msg.Scenarios), resp.Msg.VrooliRoot)},
		ResultsHeading: "Scenarios",
		Results:        results,
		RetrievalHints: []string{"`scenarios show <id>` — list the flows inside one scenario"},
	})
}

func (h *handlers) show(ctx cliapp.RunContext) error {
	id := ctx.Positional("id")
	resp, err := h.client.GetScenario(context.Background(), connect.NewRequest(&scenariosv1.GetScenarioRequest{Id: id}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("show scenario %q", id), err, nil)
	}
	s := resp.Msg.Scenario
	if s == nil || s.Summary == nil {
		return fmt.Errorf("server returned no scenario")
	}
	results := []string{
		fmt.Sprintf("id           = %s", s.Summary.Id),
		fmt.Sprintf("displayName  = %s", s.Summary.DisplayName),
		fmt.Sprintf("description  = %s", s.Summary.Description),
		fmt.Sprintf("path         = %s", s.Summary.Path),
		fmt.Sprintf("flowCount    = %d", s.Summary.FlowCount),
	}
	for _, f := range s.Flows {
		results = append(results, fmt.Sprintf("  - %s (%s)", f.FlowId, f.Language))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Scenario %s", s.Summary.Id)},
		ResultsHeading: "Detail",
		Results:        results,
	})
}
