package summarize

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client summconnect.SummarizeServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: summconnect.NewSummarizeServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) text(ctx cliapp.RunContext) error {
	level := ctx.Flag("level")
	if level == "" {
		level = "moderate"
	}
	resp, err := h.client.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{
		Text:  ctx.Flag("text"),
		Level: level,
	}))
	if err != nil {
		return cliapp.WrapAPIError("summarize", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("[%s/%s, %dms] %s",
				resp.Msg.ProviderTier, resp.Msg.ProviderId, int(resp.Msg.LatencyMs), resp.Msg.Text),
		},
	})
}
