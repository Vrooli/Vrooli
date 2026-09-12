package design

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	designv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/design"
	designconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/design/design_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

// handlers bundles the closure over *cliapp.ScenarioApp so each RunCtx-func has
// typed access to the API client.
type handlers struct {
	core   *cliapp.ScenarioApp
	client designconnect.DesignServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: designconnect.NewDesignServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) generate(ctx cliapp.RunContext) error {
	resp, err := h.client.GenerateDesignLanguage(context.Background(), connect.NewRequest(&designv1.GenerateDesignLanguageRequest{
		BrandId: ctx.Flag("brand-id"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("generate design language", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no design response")
	}
	m := resp.Msg
	return cliapp.RenderProtoMutation(ctx, m, cliapp.MutationReport{
		Result: []string{fmt.Sprintf("Rendered DESIGN.md for brand %s.", m.BrandId)},
		// The markdown document is the payload — surface it verbatim so the
		// caller can pipe it straight into a file.
		Changes: []string{m.Markdown},
		NextCommand: []string{
			fmt.Sprintf("`brands get --id %s` — inspect the brand this was rendered from", m.BrandId),
		},
	})
}
