package summarize

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"audio-tools/cli/domains/internal/providerlabels"
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

func summarizeLevelFromFlag(s string) summv1.SummarizeLevel {
	switch s {
	case "light":
		return summv1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT
	case "moderate", "":
		return summv1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE
	case "heavy":
		return summv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY
	default:
		return summv1.SummarizeLevel_SUMMARIZE_LEVEL_UNSPECIFIED
	}
}

func (h *handlers) text(ctx cliapp.RunContext) error {
	resp, err := h.client.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{
		Text:  ctx.Flag("text"),
		Level: summarizeLevelFromFlag(ctx.Flag("level")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("summarize", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("[%s/%s, %dms] %s",
				providerlabels.TierLabel(resp.Msg.GetProviderTier()), resp.Msg.ProviderId, int(resp.Msg.LatencyMs), resp.Msg.Text),
		},
	})
}
