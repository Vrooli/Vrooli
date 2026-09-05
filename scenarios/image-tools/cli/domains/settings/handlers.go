package settings

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	modelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models/models_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client modelsconnect.ModelsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: modelsconnect.NewModelsServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.ListDefaults(context.Background(), connect.NewRequest(&modelsv1.ListDefaultsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list defaults", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no defaults response")
	}
	results := make([]string, 0, len(resp.Msg.Defaults))
	for _, d := range resp.Msg.Defaults {
		modelID := d.ModelId
		if modelID == "" {
			modelID = "(none)"
		}
		results = append(results, fmt.Sprintf("%-20s %-28s [%s]", d.Operation, modelID, d.Source))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("%d operation default(s).", len(resp.Msg.Defaults))},
		ResultsHeading: "Defaults",
		Results:        results,
		RetrievalHints: []string{
			"`settings set-default <operation> <id>` — pin a default model",
			"`settings clear-default <operation>` — revert to the seed default",
		},
	})
}

func (h *handlers) setDefault(ctx cliapp.RunContext) error {
	op := ctx.Positional("operation")
	id := ctx.Positional("id")
	return h.applyDefault(ctx, op, id)
}

func (h *handlers) clearDefault(ctx cliapp.RunContext) error {
	op := ctx.Positional("operation")
	return h.applyDefault(ctx, op, "")
}

// applyDefault calls SetDefaultModel; an empty modelID clears the pin.
func (h *handlers) applyDefault(ctx cliapp.RunContext, op, modelID string) error {
	resp, err := h.client.SetDefaultModel(context.Background(), connect.NewRequest(&modelsv1.SetDefaultModelRequest{
		Operation: op,
		ModelId:   modelID,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("set default for %q", op), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no set-default response")
	}
	var result string
	if resp.Msg.ModelId == "" {
		result = fmt.Sprintf("Cleared the default pin for %q (reverted to the seed default).", resp.Msg.Operation)
	} else {
		result = fmt.Sprintf("Pinned %s as the default for %q.", resp.Msg.ModelId, resp.Msg.Operation)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{result},
		NextCommand: []string{
			"`settings list` — confirm the effective defaults",
			fmt.Sprintf("`models select %s` — preview which model would run", resp.Msg.Operation),
		},
	})
}
