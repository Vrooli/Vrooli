package tts

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client ttsconnect.TTSServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: ttsconnect.NewTTSServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) synthesize(ctx cliapp.RunContext) error {
	speed := 1.0
	if s := ctx.Flag("speed"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			speed = v
		}
	}
	format := ctx.Flag("format")
	if format == "" {
		format = "mp3"
	}
	voice := ctx.Flag("voice")
	if voice == "" {
		voice = "voice.neutral.default"
	}
	req := connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text:           ctx.Flag("text"),
		Voice:          voice,
		Speed:          speed,
		ResponseFormat: format,
	})
	resp, err := h.client.Synthesize(context.Background(), req)
	if err != nil {
		return cliapp.WrapAPIError("synthesize", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no synthesis response")
	}
	out := ctx.Flag("out")
	if out == "" {
		_, _ = os.Stdout.Write(resp.Msg.Audio)
	} else {
		if err := os.WriteFile(out, resp.Msg.Audio, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", out, err)
		}
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Synthesized %d bytes (%s) via %s/%s in %.0fms.",
				len(resp.Msg.Audio), resp.Msg.ContentType,
				resp.Msg.ProviderTier, resp.Msg.ProviderId, resp.Msg.LatencyMs),
		},
	})
}

func (h *handlers) voices(ctx cliapp.RunContext) error {
	resp, err := h.client.ListVoices(context.Background(), connect.NewRequest(&ttsv1.ListVoicesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list voices", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no voices")
	}
	results := make([]string, 0, len(resp.Msg.Voices))
	for _, v := range resp.Msg.Voices {
		results = append(results, fmt.Sprintf("%s — %s", v.Id, v.Name))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Found %d canonical voice(s).", len(resp.Msg.Voices))},
		ResultsHeading: "Voices",
		Results:        results,
	})
}
