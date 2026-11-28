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

func (h *handlers) text() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(
		func(ctx cliapp.OperationContext) (*summv1.SummarizeResponse, error) {
			resp, err := h.client.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{
				Text:           ctx.Flag("text"),
				Level:          summarizeLevelFromFlag(ctx.Flag("level")),
				LocalModelRole: ctx.Flag("local-model-role"),
			}))
			if err != nil {
				return nil, cliapp.WrapAPIError("summarize", err, nil)
			}
			return resp.Msg, nil
		},
		func(ctx cliapp.RunContext, resp *summv1.SummarizeResponse) error {
			return ctx.RenderMutation(cliapp.MutationReport{Result: []string{
				fmt.Sprintf("[%s/%s, %dms] %s",
					providerlabels.TierLabel(resp.GetProviderTier()), resp.ProviderId, int(resp.LatencyMs), resp.Text),
			}})
		},
		nil,
	)
}
