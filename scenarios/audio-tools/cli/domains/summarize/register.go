package summarize

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	summconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize/summarize_v1connect"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := summconnect.NewSummarizeServiceClient(httpClient, baseURL)
	return cliapp.SubcommandGroup{
		Name:        "summarize",
		Description: "Text summarization via the audio-tools summarize chain",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "text",
				Description: "Summarize text from --text or stdin",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "text", Required: true, Description: "Text to summarize"},
						{Name: "level", Description: "light|moderate|heavy (default moderate)"},
					},
				},
				RunCtx: func(ctx cliapp.RunContext) error {
					level := ctx.Flag("level")
					if level == "" {
						level = "moderate"
					}
					resp, err := client.Summarize(context.Background(), connect.NewRequest(&summv1.SummarizeRequest{
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
				},
			},
		},
	}
}
