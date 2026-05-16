package voice

import (
	"context"
	"fmt"
	"os"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client sttconnect.STTServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: sttconnect.NewSTTServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) transcribe(ctx cliapp.RunContext) error {
	path := ctx.Flag("file")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	format := ctx.Flag("format")
	if format == "" {
		format = "wav"
	}
	req := connect.NewRequest(&sttv1.TranscribeRequest{
		Audio:    data,
		Format:   format,
		Language: ctx.Flag("language"),
	})
	resp, err := h.client.Transcribe(context.Background(), req)
	if err != nil {
		return cliapp.WrapAPIError("transcribe", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no transcription")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Transcribed via %s/%s (%.0fms): %s",
				resp.Msg.ProviderTier, resp.Msg.ProviderId, resp.Msg.LatencyMs, resp.Msg.Text),
		},
	})
}
