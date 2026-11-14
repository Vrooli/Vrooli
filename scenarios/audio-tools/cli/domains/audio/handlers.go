package audio

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	audiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	audioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio/audio_v1connect"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client audioconnect.AudioProcessingServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: audioconnect.NewAudioProcessingServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) transcode(ctx cliapp.RunContext) error {
	in := ctx.Flag("input")
	data, err := os.ReadFile(in)
	if err != nil {
		return fmt.Errorf("read %s: %w", in, err)
	}
	resp, err := h.client.Transcode(context.Background(), connect.NewRequest(&audiov1.TranscodeRequest{
		Audio:        data,
		OutputFormat: commonv1.AudioFormat_AUDIO_FORMAT_WAV,
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
}
