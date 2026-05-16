package tts

import (
	"context"
	"errors"
	"fmt"
	"io"
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

// synthesizeStream calls the server-streaming SynthesizeStream RPC and
// appends each frame's audio bytes to --out (or stdout) as they arrive.
// Streaming-capable providers emit incremental frames; non-streaming
// providers emit a single is_final=true frame with the full audio.
func (h *handlers) synthesizeStream(ctx cliapp.RunContext) error {
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
	stream, err := h.client.SynthesizeStream(context.Background(), req)
	if err != nil {
		return cliapp.WrapAPIError("synthesize-stream", err, nil)
	}
	defer stream.Close()

	outPath := ctx.Flag("out")
	var sink io.Writer = os.Stdout
	var outFile *os.File
	if outPath != "" {
		f, err := os.Create(outPath)
		if err != nil {
			return fmt.Errorf("open %s: %w", outPath, err)
		}
		defer f.Close()
		outFile = f
		sink = f
	}

	var (
		totalBytes int
		finalTier  string
		finalID    string
		finalModel string
		finalMs    float64
		contentTy  string
	)
	for {
		ok := stream.Receive()
		if !ok {
			if err := stream.Err(); err != nil && !errors.Is(err, io.EOF) {
				return cliapp.WrapAPIError("synthesize-stream receive", err, nil)
			}
			break
		}
		frame := stream.Msg()
		if frame == nil {
			continue
		}
		if len(frame.Audio) > 0 {
			n, werr := sink.Write(frame.Audio)
			if werr != nil {
				return fmt.Errorf("write audio frame: %w", werr)
			}
			totalBytes += n
		}
		if frame.IsFinal {
			finalTier = frame.ProviderTier
			finalID = frame.ProviderId
			finalModel = frame.ModelId
			finalMs = frame.LatencyMs
			contentTy = frame.ContentType
		}
	}
	if outFile != nil {
		_ = outFile.Sync()
	}

	return cliapp.RenderProtoMutation(ctx, &ttsv1.AudioFrame{}, cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Synthesized %d bytes (%s) via %s/%s/%s in %.0fms.",
				totalBytes, contentTy, finalTier, finalID, finalModel, finalMs),
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
