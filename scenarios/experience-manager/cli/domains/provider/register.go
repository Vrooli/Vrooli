// Package provider exposes the shared ScenarioValidationService CLI surface.
package provider

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "provider"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"ScenarioValidationService.ValidateScenario": h.validate,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("provider: load from manifest: %w", err)
	}
	return group, nil
}

type handlers struct {
	core   *cliapp.ScenarioApp
	client scenariovalidationconnect.ScenarioValidationServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: scenariovalidationconnect.NewScenarioValidationServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) validate(ctx cliapp.RunContext) error {
	resp, err := h.client.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: ctx.Positional("scenario"),
		Path:     ctx.Flag("path"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("validate provider scenario", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no provider validation response")
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary: []string{fmt.Sprintf("%s: %s", resp.Msg.Scenario, resp.Msg.Status.String())},
	})
}
