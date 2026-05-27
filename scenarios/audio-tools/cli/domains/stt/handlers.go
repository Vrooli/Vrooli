package stt

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
	admin  sttconnect.STTAdminServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: sttconnect.NewSTTServiceClient(httpClient, baseURL),
		admin:  sttconnect.NewSTTAdminServiceClient(httpClient, baseURL),
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

// audioFormatLabel renders a common.AudioFormat as its lower-case wire
// name for human output. It is the display inverse of audioFormatFromFlag.
func audioFormatLabel(f commonv1.AudioFormat) string {
	switch f {
	case commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE:
		return "pcm_s16le"
	case commonv1.AudioFormat_AUDIO_FORMAT_WAV:
		return "wav"
	case commonv1.AudioFormat_AUDIO_FORMAT_MP3:
		return "mp3"
	case commonv1.AudioFormat_AUDIO_FORMAT_FLAC:
		return "flac"
	case commonv1.AudioFormat_AUDIO_FORMAT_OGG:
		return "ogg"
	case commonv1.AudioFormat_AUDIO_FORMAT_WEBM:
		return "webm"
	case commonv1.AudioFormat_AUDIO_FORMAT_OPUS:
		return "opus"
	case commonv1.AudioFormat_AUDIO_FORMAT_AAC:
		return "aac"
	default:
		return "unspecified"
	}
}

func availabilityLabel(ok bool) string {
	if ok {
		return "available"
	}
	return "unavailable"
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

// formats prints the STT ingress capability matrix: the input codecs the
// API accepts, whether the local ffmpeg decode backend is present, and the
// fixed canonical PCM target. Human-friendly output per
// feedback_cli_default_human_output.
func (h *handlers) formats(ctx cliapp.RunContext) error {
	resp, err := h.client.GetSupportedFormats(context.Background(), connect.NewRequest(&sttv1.GetSupportedFormatsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("stt-formats", err, nil)
	}
	msg := resp.Msg
	fmt.Fprintf(ctx.Stdout(), "STT ingress — accepted input formats:\n")
	for _, f := range msg.GetAcceptedFormats() {
		fmt.Fprintf(ctx.Stdout(), "  - %s\n", audioFormatLabel(f))
	}
	fmt.Fprintf(ctx.Stdout(), "ffmpeg decode backend = %s\n", availabilityLabel(msg.GetFfmpegAvailable()))
	fmt.Fprintf(ctx.Stdout(), "canonical PCM target  = %d Hz, %d channel(s), s16le\n",
		msg.GetCanonicalSampleRateHz(), msg.GetCanonicalChannels())
	if !msg.GetFfmpegAvailable() {
		fmt.Fprintf(ctx.Stdout(), "note: ffmpeg absent — live non-PCM streams fall back to buffered whole-file decode.\n")
	}
	return nil
}

// engines lists the selectable STT engines (manifest-derived) with each
// engine's runtime availability and which one is active.
func (h *handlers) engines(ctx cliapp.RunContext) error {
	resp, err := h.client.ListEngines(context.Background(), connect.NewRequest(&sttv1.ListEnginesRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("stt-engines", err, nil)
	}
	list := resp.Msg.GetEngines()
	if len(list) == 0 {
		fmt.Fprintf(ctx.Stdout(), "No STT engines declared in the manifest.\n")
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "STT engines (manifest-derived):\n")
	for _, e := range list {
		marker := " "
		if e.GetIsActive() {
			marker = "*"
		}
		streaming := "batch"
		if e.GetNativeStreaming() {
			streaming = "native-streaming"
		}
		fmt.Fprintf(ctx.Stdout(), "  %s %-16s %-34s [%s, %s, %s]\n",
			marker, e.GetId(), e.GetDisplayName(), e.GetKind(), streaming, availabilityLabel(e.GetAvailable()))
	}
	fmt.Fprintf(ctx.Stdout(), "(* = active; set with `audio-tools stt stream-config-set --engine <id>`)\n")
	return nil
}

// engineImpact reports the shared-resource impact of switching away from an
// engine: which other scenarios still depend on its backing resource and the
// exact (never auto-run) command to stop it. audio-tools never stops a shared
// resource itself — this surfaces the decision for the operator.
func (h *handlers) engineImpact(ctx cliapp.RunContext) error {
	engineID := ctx.Positional("engine")
	if engineID == "" {
		return fmt.Errorf("provide the engine id to assess (see `audio-tools stt engines`)")
	}
	resp, err := h.admin.GetEngineSwitchImpact(context.Background(), connect.NewRequest(&sttv1.GetEngineSwitchImpactRequest{
		FromEngineId: engineID,
	}))
	if err != nil {
		return cliapp.WrapAPIError("engine-impact", err, nil)
	}
	msg := resp.Msg
	if msg.GetResource() == "" {
		fmt.Fprintf(ctx.Stdout(), "Engine %q has no local backing resource — switching away frees nothing to stop.\n", engineID)
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "Switching away from %q would stop using resource %q.\n", engineID, msg.GetResource())
	if !msg.GetConsumersKnown() {
		fmt.Fprintf(ctx.Stdout(), "Could not enumerate other scenarios (scenarios directory not located); not safe to stop blindly.\n")
		fmt.Fprintf(ctx.Stdout(), "To stop it manually after confirming nothing else needs it: %s\n", msg.GetStopCommand())
		return nil
	}
	consumers := msg.GetConsumers()
	if len(consumers) == 0 {
		fmt.Fprintf(ctx.Stdout(), "No other scenario depends on %q — safe to stop to reclaim compute/VRAM:\n", msg.GetResource())
		fmt.Fprintf(ctx.Stdout(), "  %s\n", msg.GetStopCommand())
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "%d other scenario(s) still depend on %q — leaving it running is recommended:\n", len(consumers), msg.GetResource())
	for _, c := range consumers {
		req := ""
		if c.GetRequired() {
			req = " (required)"
		}
		fmt.Fprintf(ctx.Stdout(), "  - %s (%s)%s\n", c.GetDisplayName(), c.GetScenario(), req)
	}
	fmt.Fprintf(ctx.Stdout(), "If you still want to stop it: %s\n", msg.GetStopCommand())
	return nil
}

// streamConfigGet prints the resolved StreamConfig, including the five
// streaming-pipeline operator levers documented in
// docs/reference/configuration.md. The display format is
// human-friendly per feedback_cli_default_human_output.
func (h *handlers) streamConfigGet(ctx cliapp.RunContext) error {
	resp, err := h.admin.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get-stream-config", err, nil)
	}
	cfg := resp.Msg.GetConfig()
	if cfg == nil {
		return fmt.Errorf("server returned no stream config")
	}
	fmt.Fprintf(ctx.Stdout(), "Streaming STT pipeline:\n")
	fmt.Fprintf(ctx.Stdout(), "  engine_id            = %s\n", stringOrDefault(cfg.GetEngineId(), "whisper-local"))
	fmt.Fprintf(ctx.Stdout(), "  streaming_mode       = %s\n", streamingModeLabel(cfg.GetStreamingMode()))
	fmt.Fprintf(ctx.Stdout(), "  strategy_preference  = %s\n", strategyPreferenceLabel(cfg.GetStrategyPreference()))
	fmt.Fprintf(ctx.Stdout(), "  vad_silence_ms       = %d\n", intOrDefault(cfg.GetVadSilenceMs(), 700))
	fmt.Fprintf(ctx.Stdout(), "  overlap_window_ms    = %d\n", intOrDefault(cfg.GetOverlapWindowMs(), 2000))
	fmt.Fprintf(ctx.Stdout(), "  overlap_commit_runs  = %d\n", intOrDefault(cfg.GetOverlapCommitRuns(), 2))
	fmt.Fprintf(ctx.Stdout(), "Egress gate (post-recognition quality):\n")
	fmt.Fprintf(ctx.Stdout(), "  hallucination_filter = %t\n", cfg.GetHallucinationFilterEnabled())
	fmt.Fprintf(ctx.Stdout(), "  vad_filter           = %t\n", cfg.GetVadFilterEnabled())
	fmt.Fprintf(ctx.Stdout(), "  no_speech_threshold  = %.2f\n", floatOrDefault(cfg.GetNoSpeechThreshold(), 0.6))
	fmt.Fprintf(ctx.Stdout(), "  logprob_threshold    = %.2f\n", floatOrDefault(cfg.GetLogprobThreshold(), -1.0))
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
	if v := ctx.Flag("engine"); v != "" {
		cfg.EngineId = v
		mask.Paths = append(mask.Paths, "engine_id")
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
	if v := ctx.Flag("hallucination-filter"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("--hallucination-filter must be true|false: %q", v)
		}
		cfg.HallucinationFilterEnabled = b
		mask.Paths = append(mask.Paths, "hallucination_filter_enabled")
	}
	if v := ctx.Flag("vad-filter"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("--vad-filter must be true|false: %q", v)
		}
		cfg.VadFilterEnabled = b
		mask.Paths = append(mask.Paths, "vad_filter_enabled")
	}
	if v := ctx.Flag("no-speech-threshold"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("--no-speech-threshold must be a number: %q", v)
		}
		cfg.NoSpeechThreshold = f
		mask.Paths = append(mask.Paths, "no_speech_threshold")
	}
	if v := ctx.Flag("logprob-threshold"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("--logprob-threshold must be a number: %q", v)
		}
		cfg.LogprobThreshold = f
		mask.Paths = append(mask.Paths, "logprob_threshold")
	}
	if len(mask.Paths) == 0 {
		return fmt.Errorf("at least one streaming/egress lever flag must be set (e.g. --streaming-mode, --strategy-preference, --vad-silence-ms, --overlap-window-ms, --overlap-commit-runs, --hallucination-filter, --vad-filter, --no-speech-threshold, --logprob-threshold)")
	}
	resp, err := h.admin.UpdateStreamConfig(context.Background(), connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		UpdateMask: mask,
		Config:     cfg,
	}))
	if err != nil {
		return cliapp.WrapAPIError("update-stream-config", err, nil)
	}
	out := resp.Msg.GetConfig()
	fmt.Fprintf(ctx.Stdout(), "Updated. Resolved streaming STT pipeline:\n")
	fmt.Fprintf(ctx.Stdout(), "  engine_id            = %s\n", stringOrDefault(out.GetEngineId(), "whisper-local"))
	fmt.Fprintf(ctx.Stdout(), "  streaming_mode       = %s\n", streamingModeLabel(out.GetStreamingMode()))
	fmt.Fprintf(ctx.Stdout(), "  strategy_preference  = %s\n", strategyPreferenceLabel(out.GetStrategyPreference()))
	fmt.Fprintf(ctx.Stdout(), "  vad_silence_ms       = %d\n", intOrDefault(out.GetVadSilenceMs(), 700))
	fmt.Fprintf(ctx.Stdout(), "  overlap_window_ms    = %d\n", intOrDefault(out.GetOverlapWindowMs(), 2000))
	fmt.Fprintf(ctx.Stdout(), "  overlap_commit_runs  = %d\n", intOrDefault(out.GetOverlapCommitRuns(), 2))
	fmt.Fprintf(ctx.Stdout(), "  hallucination_filter = %t\n", out.GetHallucinationFilterEnabled())
	fmt.Fprintf(ctx.Stdout(), "  vad_filter           = %t\n", out.GetVadFilterEnabled())
	fmt.Fprintf(ctx.Stdout(), "  no_speech_threshold  = %.2f\n", floatOrDefault(out.GetNoSpeechThreshold(), 0.6))
	fmt.Fprintf(ctx.Stdout(), "  logprob_threshold    = %.2f\n", floatOrDefault(out.GetLogprobThreshold(), -1.0))
	return nil
}

func stringOrDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func floatOrDefault(v, def float64) float64 {
	if v == 0 {
		return def
	}
	return v
}

func intOrDefault(v, def int32) int32 {
	if v == 0 {
		return def
	}
	return v
}
