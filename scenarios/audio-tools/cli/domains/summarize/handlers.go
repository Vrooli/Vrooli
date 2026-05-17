package summarize

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

func providerTierLabel(t commonv1.ProviderTier) string {
	switch t {
	case commonv1.ProviderTier_PROVIDER_TIER_LOCAL:
		return "local"
	case commonv1.ProviderTier_PROVIDER_TIER_BYOK:
		return "byok"
	case commonv1.ProviderTier_PROVIDER_TIER_VROOLI:
		return "vrooli"
	default:
		return "unknown"
	}
}

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

func summarizeLevelFromFlag(s string) ttsv1.SummarizeLevel {
	switch s {
	case "light":
		return ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT
	case "moderate", "":
		return ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE
	case "heavy":
		return ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY
	default:
		return ttsv1.SummarizeLevel_SUMMARIZE_LEVEL_UNSPECIFIED
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
				providerTierLabel(resp.Msg.GetProviderTier()), resp.Msg.ProviderId, int(resp.Msg.LatencyMs), resp.Msg.Text),
		},
	})
}
