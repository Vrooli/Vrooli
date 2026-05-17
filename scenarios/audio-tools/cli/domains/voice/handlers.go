package voice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/vrooli/cli-core/cliapp"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
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

func audioFormatFromFlag(s string) commonv1.AudioFormat {
	switch s {
	case "", "wav":
		return commonv1.AudioFormat_AUDIO_FORMAT_WAV
	case "mp3":
		return commonv1.AudioFormat_AUDIO_FORMAT_MP3
	case "flac":
		return commonv1.AudioFormat_AUDIO_FORMAT_FLAC
	case "ogg":
		return commonv1.AudioFormat_AUDIO_FORMAT_OGG
	case "webm":
		return commonv1.AudioFormat_AUDIO_FORMAT_WEBM
	case "opus":
		return commonv1.AudioFormat_AUDIO_FORMAT_OPUS
	default:
		return commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED
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

func streamingModeLabel(m sttv1.StreamingMode) string {
	switch m {
	case sttv1.StreamingMode_STREAMING_MODE_OFF:
		return "off"
	case sttv1.StreamingMode_STREAMING_MODE_AUTO:
		return "auto"
	default:
		return "auto"
	}
}

func strategyPreferenceLabel(p sttv1.StrategyPreference) string {
	switch p {
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD:
		return "vad"
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_OVERLAP:
		return "overlap"
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_PASSTHROUGH:
		return "passthrough"
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO:
		return "auto"
	default:
		return "auto"
	}
}

func streamingModeFromFlag(s string) (sttv1.StreamingMode, error) {
	switch s {
	case "auto":
		return sttv1.StreamingMode_STREAMING_MODE_AUTO, nil
	case "off":
		return sttv1.StreamingMode_STREAMING_MODE_OFF, nil
	default:
		return sttv1.StreamingMode_STREAMING_MODE_UNSPECIFIED, fmt.Errorf("--streaming-mode must be auto|off: %q", s)
	}
}

func strategyPreferenceFromFlag(s string) (sttv1.StrategyPreference, error) {
	switch s {
	case "auto":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO, nil
	case "vad":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD, nil
	case "overlap":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_OVERLAP, nil
	case "passthrough":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_PASSTHROUGH, nil
	default:
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_UNSPECIFIED, fmt.Errorf("--strategy-preference must be auto|vad|overlap|passthrough: %q", s)
	}
}

func (h *handlers) transcribe(ctx cliapp.RunContext) error {
	path := ctx.Flag("file")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	req := connect.NewRequest(&sttv1.TranscribeRequest{
		Audio:    data,
		Format:   audioFormatFromFlag(ctx.Flag("format")),
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
				providerTierLabel(resp.Msg.GetProviderTier()), resp.Msg.ProviderId, resp.Msg.LatencyMs, resp.Msg.Text),
		},
	})
}

// transcribeStream feeds the audio file chunk-by-chunk through the
// bidi-streaming TranscribeStream RPC and prints one human-readable
// line per emitted event (partial, segment, wake_word, speaker_rejection,
// error, done). Exits non-zero on a chain error.
func (h *handlers) transcribeStream(ctx cliapp.RunContext) error {
	path := ctx.Flag("file")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	chunkBytes := 32 * 1024
	if cb := ctx.Flag("chunk-bytes"); cb != "" {
		v, err := strconv.Atoi(cb)
		if err != nil || v <= 0 {
			return fmt.Errorf("--chunk-bytes must be a positive integer: %q", cb)
		}
		chunkBytes = v
	}

	stream := h.client.TranscribeStream(context.Background())
	if err := stream.Send(&sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_Start{
			Start: &sttv1.StreamStart{Language: ctx.Flag("language")},
		},
	}); err != nil {
		return cliapp.WrapAPIError("transcribe-stream start", err, nil)
	}

	// Push chunks then end. Errors break out and let the receive loop
	// drain whatever events the server has already emitted before
	// returning.
	go func() {
		for off := 0; off < len(data); off += chunkBytes {
			end := off + chunkBytes
			if end > len(data) {
				end = len(data)
			}
			if err := stream.Send(&sttv1.TranscribeStreamRequest{
				Payload: &sttv1.TranscribeStreamRequest_AudioChunk{AudioChunk: data[off:end]},
			}); err != nil {
				return
			}
		}
		_ = stream.Send(&sttv1.TranscribeStreamRequest{
			Payload: &sttv1.TranscribeStreamRequest_End{End: &sttv1.StreamEnd{}},
		})
		_ = stream.CloseRequest()
	}()

	var lastErr error
	for {
		ev, err := stream.Receive()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return cliapp.WrapAPIError("transcribe-stream receive", err, nil)
		}
		switch e := ev.GetEvent().(type) {
		case *sttv1.TranscribeStreamEvent_Partial:
			fmt.Fprintf(ctx.Stdout(), "partial: %s\n", e.Partial.Text)
		case *sttv1.TranscribeStreamEvent_Segment:
			fmt.Fprintf(ctx.Stdout(), "segment [%s/%s %.0fms]: %s\n",
				providerTierLabel(e.Segment.ProviderTier), e.Segment.ModelId, e.Segment.LatencyMs, e.Segment.Text)
		case *sttv1.TranscribeStreamEvent_WakeWord:
			fmt.Fprintf(ctx.Stdout(), "wake-word: score=%.3f sample=%s\n", e.WakeWord.Score, e.WakeWord.SampleId)
		case *sttv1.TranscribeStreamEvent_SpeakerRejection:
			fmt.Fprintf(ctx.Stdout(), "speaker-rejection: %s (fallback=%v)\n", e.SpeakerRejection.Reason, e.SpeakerRejection.FallbackUsed)
		case *sttv1.TranscribeStreamEvent_Error:
			fmt.Fprintf(ctx.Stderr(), "error: %s — %s\n", e.Error.Code, e.Error.Message)
			lastErr = fmt.Errorf("%s: %s", e.Error.Code, e.Error.Message)
		case *sttv1.TranscribeStreamEvent_Done:
			fallback := ""
			if e.Done.FellBackToUnary {
				fallback = " (fellback=unary)"
			}
			fmt.Fprintf(ctx.Stdout(), "done [%s/%s %.0fms%s]: %s\n",
				providerTierLabel(e.Done.ProviderTier), e.Done.ModelId, e.Done.LatencyMs, fallback, e.Done.FinalText)
		}
	}
	return lastErr
}

// streamConfigGet prints the resolved StreamConfig, including the five
// streaming-pipeline operator levers documented in
// docs/reference/configuration.md. The display format is
// human-friendly per feedback_cli_default_human_output.
func (h *handlers) streamConfigGet(ctx cliapp.RunContext) error {
	resp, err := h.client.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get-stream-config", err, nil)
	}
	cfg := resp.Msg.GetConfig()
	if cfg == nil {
		return fmt.Errorf("server returned no stream config")
	}
	fmt.Fprintf(ctx.Stdout(), "Streaming STT pipeline:\n")
	fmt.Fprintf(ctx.Stdout(), "  streaming_mode       = %s\n", streamingModeLabel(cfg.GetStreamingMode()))
	fmt.Fprintf(ctx.Stdout(), "  strategy_preference  = %s\n", strategyPreferenceLabel(cfg.GetStrategyPreference()))
	fmt.Fprintf(ctx.Stdout(), "  vad_silence_ms       = %d\n", intOrDefault(cfg.GetVadSilenceMs(), 700))
	fmt.Fprintf(ctx.Stdout(), "  overlap_window_ms    = %d\n", intOrDefault(cfg.GetOverlapWindowMs(), 2000))
	fmt.Fprintf(ctx.Stdout(), "  overlap_commit_runs  = %d\n", intOrDefault(cfg.GetOverlapCommitRuns(), 2))
	fmt.Fprintf(ctx.Stdout(), "Legacy partial-window fields (still used by browser WS):\n")
	fmt.Fprintf(ctx.Stdout(), "  flush_interval_ms    = %d\n", cfg.GetFlushIntervalMs())
	fmt.Fprintf(ctx.Stdout(), "  min_delta_bytes      = %d\n", cfg.GetMinDeltaBytes())
	fmt.Fprintf(ctx.Stdout(), "  overlap_bytes        = %d\n", cfg.GetOverlapBytes())
	fmt.Fprintf(ctx.Stdout(), "  segment_silence_ms   = %d\n", cfg.GetSegmentSilenceMs())
	return nil
}

// streamConfigSet mutates the persisted StreamConfig. Each provided
// flag is added to the FieldMask and its corresponding value populated
// in the request's Config payload; omitted flags leave the persisted
// value untouched. The server enforces the documented ranges and
// returns InvalidArgument on a forbidden value.
func (h *handlers) streamConfigSet(ctx cliapp.RunContext) error {
	mask := &fieldmaskpb.FieldMask{}
	cfg := &sttv1.StreamConfig{}
	if v := ctx.Flag("streaming-mode"); v != "" {
		m, err := streamingModeFromFlag(v)
		if err != nil {
			return err
		}
		cfg.StreamingMode = m
		mask.Paths = append(mask.Paths, "streaming_mode")
	}
	if v := ctx.Flag("strategy-preference"); v != "" {
		p, err := strategyPreferenceFromFlag(v)
		if err != nil {
			return err
		}
		cfg.StrategyPreference = p
		mask.Paths = append(mask.Paths, "strategy_preference")
	}
	if v := ctx.Flag("vad-silence-ms"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--vad-silence-ms must be integer: %q", v)
		}
		cfg.VadSilenceMs = int32(n)
		mask.Paths = append(mask.Paths, "vad_silence_ms")
	}
	if v := ctx.Flag("overlap-window-ms"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--overlap-window-ms must be integer: %q", v)
		}
		cfg.OverlapWindowMs = int32(n)
		mask.Paths = append(mask.Paths, "overlap_window_ms")
	}
	if v := ctx.Flag("overlap-commit-runs"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("--overlap-commit-runs must be integer: %q", v)
		}
		cfg.OverlapCommitRuns = int32(n)
		mask.Paths = append(mask.Paths, "overlap_commit_runs")
	}
	if len(mask.Paths) == 0 {
		return fmt.Errorf("at least one of --streaming-mode, --strategy-preference, --vad-silence-ms, --overlap-window-ms, --overlap-commit-runs must be set")
	}
	resp, err := h.client.UpdateStreamConfig(context.Background(), connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		UpdateMask: mask,
		Config:     cfg,
	}))
	if err != nil {
		return cliapp.WrapAPIError("update-stream-config", err, nil)
	}
	out := resp.Msg.GetConfig()
	fmt.Fprintf(ctx.Stdout(), "Updated. Resolved streaming STT pipeline:\n")
	fmt.Fprintf(ctx.Stdout(), "  streaming_mode       = %s\n", streamingModeLabel(out.GetStreamingMode()))
	fmt.Fprintf(ctx.Stdout(), "  strategy_preference  = %s\n", strategyPreferenceLabel(out.GetStrategyPreference()))
	fmt.Fprintf(ctx.Stdout(), "  vad_silence_ms       = %d\n", intOrDefault(out.GetVadSilenceMs(), 700))
	fmt.Fprintf(ctx.Stdout(), "  overlap_window_ms    = %d\n", intOrDefault(out.GetOverlapWindowMs(), 2000))
	fmt.Fprintf(ctx.Stdout(), "  overlap_commit_runs  = %d\n", intOrDefault(out.GetOverlapCommitRuns(), 2))
	return nil
}

func intOrDefault(v, def int32) int32 {
	if v == 0 {
		return def
	}
	return v
}
