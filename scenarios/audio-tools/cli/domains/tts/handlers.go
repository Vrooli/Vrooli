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

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
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

func responseFormatFromFlag(s string) commonv1.ResponseFormat {
	switch s {
	case "", "mp3":
		return commonv1.ResponseFormat_RESPONSE_FORMAT_MP3
	case "wav":
		return commonv1.ResponseFormat_RESPONSE_FORMAT_WAV
	case "opus":
		return commonv1.ResponseFormat_RESPONSE_FORMAT_OPUS
	case "flac":
		return commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC
	default:
		return commonv1.ResponseFormat_RESPONSE_FORMAT_UNSPECIFIED
	}
}

// responseFormatLabel renders a common.ResponseFormat as its lower-case
// wire name for human output. Display inverse of responseFormatFromFlag.
func responseFormatLabel(f commonv1.ResponseFormat) string {
	switch f {
	case commonv1.ResponseFormat_RESPONSE_FORMAT_MP3:
		return "mp3"
	case commonv1.ResponseFormat_RESPONSE_FORMAT_WAV:
		return "wav"
	case commonv1.ResponseFormat_RESPONSE_FORMAT_OPUS:
		return "opus"
	case commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC:
		return "flac"
	default:
		return "unspecified"
	}
}

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

func (h *handlers) synthesize(ctx cliapp.RunContext) error {
	speed := 1.0
	if s := ctx.Flag("speed"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			speed = v
		}
	}
	voice := ctx.Flag("voice")
	if voice == "" {
		voice = "voice.neutral.default"
	}
	req := connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text:           ctx.Flag("text"),
		Voice:          voice,
		Speed:          speed,
		ResponseFormat: responseFormatFromFlag(ctx.Flag("format")),
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
				providerTierLabel(resp.Msg.GetProviderTier()), resp.Msg.ProviderId, resp.Msg.LatencyMs),
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
	voice := ctx.Flag("voice")
	if voice == "" {
		voice = "voice.neutral.default"
	}
	req := connect.NewRequest(&ttsv1.SynthesizeRequest{
		Text:           ctx.Flag("text"),
		Voice:          voice,
		Speed:          speed,
		ResponseFormat: responseFormatFromFlag(ctx.Flag("format")),
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
			finalTier = providerTierLabel(frame.GetProviderTier())
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

// formats prints the TTS egress capability matrix: the output containers
// the API can produce for a --format. Human-friendly output per
// feedback_cli_default_human_output.
func (h *handlers) formats(ctx cliapp.RunContext) error {
	resp, err := h.client.GetSupportedFormats(context.Background(), connect.NewRequest(&ttsv1.GetSupportedFormatsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("tts-formats", err, nil)
	}
	msg := resp.Msg
	fmt.Fprintf(ctx.Stdout(), "TTS egress — producible output formats:\n")
	for _, f := range msg.GetEmittedFormats() {
		fmt.Fprintf(ctx.Stdout(), "  - %s\n", responseFormatLabel(f))
	}
	fmt.Fprintf(ctx.Stdout(), "ffmpeg encode backend = %s\n", availabilityLabel(msg.GetFfmpegAvailable()))
	return nil
}

func availabilityLabel(ok bool) string {
	if ok {
		return "available"
	}
	return "unavailable"
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
