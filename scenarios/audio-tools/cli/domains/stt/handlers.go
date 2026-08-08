package stt

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/types/known/fieldmaskpb"

	"github.com/vrooli/cli-core/cliapp"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	sttconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt/stt_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client sttconnect.STTServiceClient
	stream sttconnect.STTServiceClient
	admin  sttconnect.STTAdminServiceClient
}

type streamingHTTPClient struct {
	core   *cliapp.ScenarioApp
	client *http.Client
}

func (c streamingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if c.core != nil {
		if c.core.HTTPClient != nil {
			c.core.HTTPClient.ApplyRequestHeaders(req)
		}
		if c.core.APIClient != nil {
			for k, v := range c.core.APIClient.AuthHeaders() {
				req.Header.Set(k, v)
			}
		}
	}
	client := c.client
	if client == nil {
		return nil, errors.New("streaming HTTP client is not configured")
	}
	// The CLI deliberately connects to the operator-selected scenario API base;
	// the shared cli-core preflight validates that base before commands run.
	// #nosec G704 -- this is the configured scenario endpoint, not a request URL derived from input.
	return client.Do(req)
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: sttconnect.NewSTTServiceClient(httpClient, baseURL),
		stream: NewStreamingSTTClient(core, baseURL),
		admin:  sttconnect.NewSTTAdminServiceClient(httpClient, baseURL),
	}
}

func applyBoolFlag(ctx cliapp.RunContext, flag, path string, mask *fieldmaskpb.FieldMask, set func(bool)) error {
	raw := ctx.Flag(flag)
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fmt.Errorf("--%s must be true|false: %q", flag, raw)
	}
	set(value)
	mask.Paths = append(mask.Paths, path)
	return nil
}

// NewStreamingSTTClient builds the HTTP/2-capable STT client used by commands
// that send a long-lived request stream. Keeping this transport seam here lets
// validation and normal STT commands share the same auth, timeout, and clear
// missing-client behavior.
func NewStreamingSTTClient(core *cliapp.ScenarioApp, baseURL string) sttconnect.STTServiceClient {
	var timeout time.Duration
	if core != nil {
		if core.HTTPClient != nil {
			timeout = core.HTTPClient.Timeout()
		}
	}
	client := &http.Client{Timeout: timeout}
	if parsed, err := url.Parse(baseURL); err == nil && parsed.Scheme == "http" {
		client.Transport = &http2.Transport{
			AllowHTTP: true,
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, network, addr)
			},
		}
	}
	return sttconnect.NewSTTServiceClient(streamingHTTPClient{core: core, client: client}, baseURL, connect.WithGRPC())
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

	streamClient := h.stream
	if streamClient == nil {
		streamClient = h.client
	}
	stream := streamClient.TranscribeStream(context.Background())
	sessionID, err := streamSessionIdentity()
	if err != nil {
		return err
	}
	resumeToken, err := streamSessionIdentity()
	if err != nil {
		return err
	}
	if err := stream.Send(&sttv1.TranscribeStreamRequest{
		Payload: &sttv1.TranscribeStreamRequest_Start{
			Start: &sttv1.StreamStart{Language: ctx.Flag("language"), ProtocolVersion: 2, SessionId: sessionID, ResumeToken: resumeToken},
		},
	}); err != nil {
		return cliapp.WrapAPIError("transcribe-stream start", err, nil)
	}

	// Push chunks then end. Errors break out and let the receive loop
	// drain whatever events the server has already emitted before
	// returning.
	go func() {
		var sequence uint64
		for off := 0; off < len(data); off += chunkBytes {
			end := off + chunkBytes
			if end > len(data) {
				end = len(data)
			}
			chunk := data[off:end]
			digest := sha256.Sum256(chunk)
			if err := stream.Send(&sttv1.TranscribeStreamRequest{
				Payload: &sttv1.TranscribeStreamRequest_AudioChunk{AudioChunk: &sttv1.StreamAudioChunk{
					Audio: chunk, Sequence: sequence, StartSample: int64(off / 2), EndSample: int64(end / 2), Sha256: digest[:],
				}},
			}); err != nil {
				return
			}
			sequence++
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
				fallback = " (streaming degraded: buffered mode)"
			}
			fmt.Fprintf(ctx.Stdout(), "done [%s/%s %.0fms%s]: %s\n",
				providerTierLabel(e.Done.ProviderTier), e.Done.ModelId, e.Done.LatencyMs, fallback, e.Done.FinalText)
		}
	}
	return lastErr
}

func streamSessionIdentity() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate stream identity: %w", err)
	}
	return hex.EncodeToString(value[:]), nil
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
	fmt.Fprintf(ctx.Stdout(), "  overlap_max_stall_rejects = %d%s\n", cfg.GetOverlapMaxStallRejects(), stallRejectsHint(cfg.GetOverlapMaxStallRejects()))
	fmt.Fprintf(ctx.Stdout(), "Egress gate (post-recognition quality):\n")
	fmt.Fprintf(ctx.Stdout(), "  hallucination_filter = %t\n", cfg.GetHallucinationFilterEnabled())
	fmt.Fprintf(ctx.Stdout(), "  vad_filter           = %t\n", cfg.GetVadFilterEnabled())
	fmt.Fprintf(ctx.Stdout(), "  no_speech_threshold  = %.2f\n", floatOrDefault(cfg.GetNoSpeechThreshold(), 0.6))
	fmt.Fprintf(ctx.Stdout(), "  logprob_threshold    = %.2f\n", floatOrDefault(cfg.GetLogprobThreshold(), -1.0))
	fmt.Fprintf(ctx.Stdout(), "Ingress gate (pre-recognition audio):\n")
	fmt.Fprintf(ctx.Stdout(), "  denoise              = %t\n", cfg.GetDenoiseEnabled())
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
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return fmt.Errorf("--vad-silence-ms must be integer: %q", v)
		}
		cfg.VadSilenceMs = int32(n)
		mask.Paths = append(mask.Paths, "vad_silence_ms")
	}
	if v := ctx.Flag("overlap-window-ms"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return fmt.Errorf("--overlap-window-ms must be integer: %q", v)
		}
		cfg.OverlapWindowMs = int32(n)
		mask.Paths = append(mask.Paths, "overlap_window_ms")
	}
	if v := ctx.Flag("overlap-commit-runs"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return fmt.Errorf("--overlap-commit-runs must be integer: %q", v)
		}
		cfg.OverlapCommitRuns = int32(n)
		mask.Paths = append(mask.Paths, "overlap_commit_runs")
	}
	if v := ctx.Flag("overlap-max-stall-rejects"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return fmt.Errorf("--overlap-max-stall-rejects must be integer: %q", v)
		}
		cfg.OverlapMaxStallRejects = int32(n)
		mask.Paths = append(mask.Paths, "overlap_max_stall_rejects")
	}
	if err := applyBoolFlag(ctx, "hallucination-filter", "hallucination_filter_enabled", mask, func(value bool) { cfg.HallucinationFilterEnabled = value }); err != nil {
		return err
	}
	if err := applyBoolFlag(ctx, "vad-filter", "vad_filter_enabled", mask, func(value bool) { cfg.VadFilterEnabled = value }); err != nil {
		return err
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
	if err := applyBoolFlag(ctx, "denoise", "denoise_enabled", mask, func(value bool) { cfg.DenoiseEnabled = value }); err != nil {
		return err
	}
	if len(mask.Paths) == 0 {
		return fmt.Errorf("at least one streaming/egress lever flag must be set (e.g. --streaming-mode, --strategy-preference, --vad-silence-ms, --overlap-window-ms, --overlap-commit-runs, --overlap-max-stall-rejects, --hallucination-filter, --vad-filter, --no-speech-threshold, --logprob-threshold)")
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
	fmt.Fprintf(ctx.Stdout(), "  overlap_max_stall_rejects = %d%s\n", out.GetOverlapMaxStallRejects(), stallRejectsHint(out.GetOverlapMaxStallRejects()))
	fmt.Fprintf(ctx.Stdout(), "  hallucination_filter = %t\n", out.GetHallucinationFilterEnabled())
	fmt.Fprintf(ctx.Stdout(), "  vad_filter           = %t\n", out.GetVadFilterEnabled())
	fmt.Fprintf(ctx.Stdout(), "  no_speech_threshold  = %.2f\n", floatOrDefault(out.GetNoSpeechThreshold(), 0.6))
	fmt.Fprintf(ctx.Stdout(), "  logprob_threshold    = %.2f\n", floatOrDefault(out.GetLogprobThreshold(), -1.0))
	fmt.Fprintf(ctx.Stdout(), "  denoise              = %t\n", out.GetDenoiseEnabled())
	return nil
}

// stallRejectsHint annotates the overlap_max_stall_rejects value so the
// operator can tell the disabled sentinel (0) apart from an active count.
func stallRejectsHint(v int32) string {
	if v == 0 {
		return " (disabled — only the max_window_ms net applies)"
	}
	return ""
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

// ----- speaker verification ----------------------------------------

func speakerModeFromFlag(s string) (sttv1.SpeakerMode, error) {
	switch s {
	case "off":
		return sttv1.SpeakerMode_SPEAKER_MODE_OFF, nil
	case "filter":
		return sttv1.SpeakerMode_SPEAKER_MODE_FILTER, nil
	case "advisory":
		return sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY, nil
	default:
		return sttv1.SpeakerMode_SPEAKER_MODE_UNSPECIFIED, fmt.Errorf("--mode must be off|filter|advisory: %q", s)
	}
}

func speakerModeLabel(m sttv1.SpeakerMode) string {
	switch m {
	case sttv1.SpeakerMode_SPEAKER_MODE_OFF:
		return "off"
	case sttv1.SpeakerMode_SPEAKER_MODE_FILTER:
		return "filter"
	case sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY:
		return "advisory"
	default:
		return "unspecified"
	}
}

func rejectBehaviorFromFlag(s string) (sttv1.RejectBehavior, error) {
	switch s {
	case "drop":
		return sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP, nil
	case "show-muted":
		return sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED, nil
	default:
		return sttv1.RejectBehavior_REJECT_BEHAVIOR_UNSPECIFIED, fmt.Errorf("--reject-behavior must be drop|show-muted: %q", s)
	}
}

func rejectBehaviorLabel(r sttv1.RejectBehavior) string {
	switch r {
	case sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP:
		return "drop"
	case sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED:
		return "show-muted"
	default:
		return "unspecified"
	}
}

func boolPtr(b bool) *bool { return &b }

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// speakerStatus prints the live speaker-verification config, resource
// capability, and enrolled profiles. Surfaces the Whisper-only caveat so a
// user enabling it on the streaming engine isn't silently unprotected.
func (h *handlers) speakerStatus(ctx cliapp.RunContext) error {
	resp, err := h.admin.GetSpeakerStatus(context.Background(), connect.NewRequest(&sttv1.GetSpeakerStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get-speaker-status", err, nil)
	}
	st := resp.Msg.GetStatus()
	if st == nil {
		return fmt.Errorf("server returned no speaker status")
	}
	if ctx.JSON() {
		return cliapp.PrintProtoJSON(ctx.Stdout(), st)
	}
	cfg := st.GetConfig()
	fmt.Fprintf(ctx.Stdout(), "Speaker verification:\n")
	fmt.Fprintf(ctx.Stdout(), "  capability      = %s (%s)\n", st.GetCapability(), st.GetCapabilityLabel())
	fmt.Fprintf(ctx.Stdout(), "  resource_ready  = %t\n", st.GetResourceReady())
	fmt.Fprintf(ctx.Stdout(), "  enabled         = %t\n", cfg.GetEnabled())
	fmt.Fprintf(ctx.Stdout(), "  mode            = %s\n", speakerModeLabel(cfg.GetMode()))
	fmt.Fprintf(ctx.Stdout(), "  threshold       = %.2f\n", cfg.GetThreshold())
	fmt.Fprintf(ctx.Stdout(), "  reject_behavior = %s\n", rejectBehaviorLabel(cfg.GetRejectBehavior()))
	fmt.Fprintf(ctx.Stdout(), "  active_profiles = %v\n", cfg.GetProfileIds())
	fmt.Fprintf(ctx.Stdout(), "  extraction      = %t\n", cfg.GetExtractionEnabled())
	fmt.Fprintf(ctx.Stdout(), "  enrolled        = %d profile(s)\n", st.GetProfileCount())
	for _, p := range st.GetProfiles() {
		active := ""
		for _, id := range cfg.GetProfileIds() {
			if id == p.GetId() {
				active = " [active]"
				break
			}
		}
		fmt.Fprintf(ctx.Stdout(), "    - %s (%s, %d clip(s), %.1fs voiced)%s\n", p.GetId(), p.GetDisplayName(), p.GetClipCount(), p.GetTotalVoicedSeconds(), active)
	}
	if cfg.GetEnabled() && cfg.GetMode() != sttv1.SpeakerMode_SPEAKER_MODE_OFF {
		fmt.Fprintf(ctx.Stdout(), "  note: speaker isolation only protects the Whisper VAD path (per-segment\n")
		fmt.Fprintf(ctx.Stdout(), "        PCM). The Kyutai streaming engine emits no per-segment audio, so it\n")
		fmt.Fprintf(ctx.Stdout(), "        is NOT gated by speaker verification.\n")
	}
	return nil
}

// speakerConfig mutates the speaker-verification config. Only provided flags
// are changed. --profiles replaces the active binding list; --bind-profile
// appends one (reads the current list first).
func (h *handlers) speakerConfig(ctx cliapp.RunContext) error {
	mask := &fieldmaskpb.FieldMask{}
	cfg := &sttv1.SpeakerConfig{}
	if v := ctx.Flag("mode"); v != "" {
		m, err := speakerModeFromFlag(v)
		if err != nil {
			return err
		}
		cfg.Mode = m
		mask.Paths = append(mask.Paths, "mode")
	}
	if v := ctx.Flag("threshold"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("--threshold must be a number: %q", v)
		}
		cfg.Threshold = f
		mask.Paths = append(mask.Paths, "threshold")
	}
	if err := applyBoolFlag(ctx, "enabled", "enabled", mask, func(value bool) { cfg.Enabled = value }); err != nil {
		return err
	}
	if v := ctx.Flag("reject-behavior"); v != "" {
		r, err := rejectBehaviorFromFlag(v)
		if err != nil {
			return err
		}
		cfg.RejectBehavior = r
		mask.Paths = append(mask.Paths, "reject_behavior")
	}
	if err := applyBoolFlag(ctx, "fallback", "fallback_without_verification", mask, func(value bool) { cfg.FallbackWithoutVerification = value }); err != nil {
		return err
	}
	if err := applyBoolFlag(ctx, "extraction-enabled", "extraction_enabled", mask, func(value bool) { cfg.ExtractionEnabled = value }); err != nil {
		return err
	}
	if v := ctx.Flag("min-decision-seconds"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("--min-decision-seconds must be a number: %q", v)
		}
		cfg.MinDecisionSeconds = f
		mask.Paths = append(mask.Paths, "min_decision_seconds")
	}
	if v := ctx.Flag("score-smoothing"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("--score-smoothing must be a number: %q", v)
		}
		cfg.ScoreSmoothing = f
		mask.Paths = append(mask.Paths, "score_smoothing")
	}
	profilesFlag := ctx.Flag("profiles")
	bindFlag := ctx.Flag("bind-profile")
	if profilesFlag != "" && bindFlag != "" {
		return fmt.Errorf("use either --profiles (replace) or --bind-profile (append), not both")
	}
	if profilesFlag != "" {
		cfg.ProfileIds = splitCSV(profilesFlag)
		mask.Paths = append(mask.Paths, "profile_ids")
	}
	if bindFlag != "" {
		cur, err := h.admin.GetSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.GetSpeakerConfigRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("get-speaker-config", err, nil)
		}
		ids := append([]string{}, cur.Msg.GetConfig().GetProfileIds()...)
		already := false
		for _, id := range ids {
			if id == bindFlag {
				already = true
				break
			}
		}
		if !already {
			ids = append(ids, bindFlag)
		}
		cfg.ProfileIds = ids
		mask.Paths = append(mask.Paths, "profile_ids")
	}
	if len(mask.Paths) == 0 {
		return fmt.Errorf("at least one flag must be set (--mode, --threshold, --enabled, --profiles, --bind-profile, --reject-behavior, --fallback, --extraction-enabled, --min-decision-seconds, --score-smoothing)")
	}
	resp, err := h.admin.UpdateSpeakerConfig(context.Background(), connect.NewRequest(&sttv1.UpdateSpeakerConfigRequest{
		UpdateMask: mask,
		Config:     cfg,
	}))
	if err != nil {
		return cliapp.WrapAPIError("update-speaker-config", err, nil)
	}
	out := resp.Msg.GetConfig()
	fmt.Fprintf(ctx.Stdout(), "Updated. Resolved speaker config:\n")
	fmt.Fprintf(ctx.Stdout(), "  enabled         = %t\n", out.GetEnabled())
	fmt.Fprintf(ctx.Stdout(), "  mode            = %s\n", speakerModeLabel(out.GetMode()))
	fmt.Fprintf(ctx.Stdout(), "  threshold       = %.2f\n", out.GetThreshold())
	fmt.Fprintf(ctx.Stdout(), "  reject_behavior = %s\n", rejectBehaviorLabel(out.GetRejectBehavior()))
	fmt.Fprintf(ctx.Stdout(), "  active_profiles = %v\n", out.GetProfileIds())
	fmt.Fprintf(ctx.Stdout(), "  fallback        = %t\n", out.GetFallbackWithoutVerification())
	fmt.Fprintf(ctx.Stdout(), "  extraction      = %t\n", out.GetExtractionEnabled())
	fmt.Fprintf(ctx.Stdout(), "  min_decision_s  = %.1f\n", out.GetMinDecisionSeconds())
	fmt.Fprintf(ctx.Stdout(), "  score_smoothing = %.2f\n", out.GetScoreSmoothing())
	return nil
}

// speakerEnroll appends one or more enrollment clips to a voice profile. --file
// accepts a comma-separated list to enroll several clips (e.g. different devices
// or speaking styles) in one call — each becomes its own append against the same
// profile, strengthening the identity. Pass --activate to bind+enable in one
// step. A profile is one identity; each clip is one condition (--label).
func (h *handlers) speakerEnroll(ctx cliapp.RunContext) error {
	paths := splitCSV(ctx.Flag("file"))
	if len(paths) == 0 {
		return fmt.Errorf("--file is required (comma-separated for multiple clips)")
	}

	var activate *bool
	if v := ctx.Flag("activate"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("--activate must be true|false: %q", v)
		}
		activate = boolPtr(b)
	}

	format := audioFormatFromFlag(ctx.Flag("format"))
	label := ctx.Flag("label")
	name := ctx.Flag("name")
	notes := ctx.Flag("notes")
	profileID := ctx.Flag("profile")

	fmt.Fprintf(ctx.Stdout(), "Enrolling %d clip(s):\n", len(paths))
	var lastResp *sttv1.EnrollSpeakerProfileResponse
	for i, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		req := &sttv1.EnrollSpeakerProfileRequest{
			Audio:       data,
			Format:      format,
			ProfileId:   profileID,
			DisplayName: name,
			Notes:       notes,
			Label:       label,
		}
		// Activate once, on the first clip; later clips just append.
		if i == 0 && activate != nil {
			req.AddToActive = activate
			req.Enable = activate
		}
		resp, err := h.admin.EnrollSpeakerProfile(context.Background(), connect.NewRequest(req))
		if err != nil {
			return cliapp.WrapAPIError("enroll-speaker-profile", err, nil)
		}
		en := resp.Msg.GetEnrollment()
		// Subsequent clips land in the same profile (server may have generated id).
		profileID = en.GetProfileId()
		lastResp = resp.Msg
		fmt.Fprintf(ctx.Stdout(), "  [%d] %s -> clip %s (label=%q, %.1fs voiced) — %d clip(s) total\n",
			i+1, path, en.GetClipId(), en.GetLabel(), en.GetVoicedSeconds(), en.GetClipCount())
	}

	if lastResp != nil {
		en := lastResp.GetEnrollment()
		fmt.Fprintf(ctx.Stdout(), "Profile %s: %d clip(s), %.1fs total voiced; model %s (dim %d)\n",
			en.GetProfileId(), en.GetClipCount(), en.GetTotalVoicedSeconds(), en.GetModelName(), en.GetEmbeddingDim())
		if cfg := lastResp.GetConfig(); cfg != nil {
			fmt.Fprintf(ctx.Stdout(), "  active_now = %v (enabled=%t)\n", cfg.GetProfileIds(), cfg.GetEnabled())
		}
	}
	return nil
}

// speakerClips lists the enrollment clips of a profile.
func (h *handlers) speakerClips(ctx cliapp.RunContext) error {
	profile := ctx.Flag("profile")
	if profile == "" {
		return fmt.Errorf("--profile is required")
	}
	resp, err := h.admin.ListSpeakerProfileClips(context.Background(), connect.NewRequest(&sttv1.ListSpeakerProfileClipsRequest{
		ProfileId: profile,
	}))
	if err != nil {
		return cliapp.WrapAPIError("list-speaker-profile-clips", err, nil)
	}
	clips := resp.Msg.GetClips()
	fmt.Fprintf(ctx.Stdout(), "Profile %s: %d clip(s)\n", resp.Msg.GetProfileId(), resp.Msg.GetCount())
	for _, c := range clips {
		fmt.Fprintf(ctx.Stdout(), "  - %s (label=%q, %.1fs voiced)\n", c.GetClipId(), c.GetLabel(), c.GetVoicedSeconds())
	}
	return nil
}

// speakerDeleteClip removes one clip from a profile and recomputes its centroid.
// Deleting the last clip deletes the profile.
func (h *handlers) speakerDeleteClip(ctx cliapp.RunContext) error {
	profile := ctx.Flag("profile")
	clip := ctx.Flag("clip")
	if profile == "" || clip == "" {
		return fmt.Errorf("--profile and --clip are required")
	}
	resp, err := h.admin.DeleteSpeakerProfileClip(context.Background(), connect.NewRequest(&sttv1.DeleteSpeakerProfileClipRequest{
		ProfileId: profile,
		ClipId:    clip,
	}))
	if err != nil {
		return cliapp.WrapAPIError("delete-speaker-profile-clip", err, nil)
	}
	m := resp.Msg
	if m.GetDeletedProfile() {
		fmt.Fprintf(ctx.Stdout(), "Deleted clip %s — that was the last clip, so profile %s was removed.\n", m.GetClipId(), m.GetProfileId())
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "Deleted clip %s from profile %s — %d clip(s) remain (%.1fs total voiced).\n",
		m.GetClipId(), m.GetProfileId(), m.GetClipCount(), m.GetTotalVoicedSeconds())
	return nil
}
