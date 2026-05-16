package audio

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := audioconnect.NewAudioProcessingServiceClient(httpClient, baseURL)
	return cliapp.SubcommandGroup{
		Name:        "audio",
		Description: "Audio processing (transcode, trim, merge, ...)",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "transcode",
				Description: "Transcode an audio file to WAV (16 kHz mono)",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "input", Required: true, Description: "Input audio path"},
						{Name: "output", Required: true, Description: "Output path"},
					},
				},
				RunCtx: func(ctx cliapp.RunContext) error {
					in := ctx.Flag("input")
					data, err := os.ReadFile(in)
					if err != nil {
						return fmt.Errorf("read %s: %w", in, err)
					}
					resp, err := client.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
						Audio:        data,
						OutputFormat: "wav",
					}))
					if err != nil {
						return cliapp.WrapAPIError("transcode", err, nil)
					}
					if err := os.WriteFile(ctx.Flag("output"), resp.Msg.Audio, 0o644); err != nil {
						return fmt.Errorf("write output: %w", err)
					}
					return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
						Result: []string{fmt.Sprintf("Wrote %d bytes to %s.", len(resp.Msg.Audio), ctx.Flag("output"))},
					})
				},
			},
		},
	}
}
