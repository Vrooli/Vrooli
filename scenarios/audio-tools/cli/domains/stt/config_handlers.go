package stt

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (h *handlers) formatsPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(
		func(ctx cliapp.OperationContext) (*sttv1.GetSupportedFormatsResponse, error) {
			resp, err := h.client.GetSupportedFormats(context.Background(), connect.NewRequest(&sttv1.GetSupportedFormatsRequest{}))
			if err != nil {
				return nil, cliapp.WrapAPIError("stt-formats", err, nil)
			}
			if resp == nil || resp.Msg == nil {
				return nil, fmt.Errorf("server returned no format capabilities")
			}
			return resp.Msg, nil
		},
		renderFormats,
		nil,
	)
}

func (h *handlers) formats(ctx cliapp.RunContext) error { return h.formatsPrimitive().Run(ctx) }

func renderFormats(ctx cliapp.RunContext, msg *sttv1.GetSupportedFormatsResponse) error {
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
func (h *handlers) enginesPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(
		func(ctx cliapp.OperationContext) (*sttv1.ListEnginesResponse, error) {
			resp, err := h.client.ListEngines(context.Background(), connect.NewRequest(&sttv1.ListEnginesRequest{}))
			if err != nil {
				return nil, cliapp.WrapAPIError("stt-engines", err, nil)
			}
			if resp == nil || resp.Msg == nil {
				return nil, fmt.Errorf("server returned no STT engines")
			}
			return resp.Msg, nil
		},
		renderEngines,
		nil,
	)
}

func renderEngines(ctx cliapp.RunContext, resp *sttv1.ListEnginesResponse) error {
	list := resp.GetEngines()
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
		fmt.Fprintf(ctx.Stdout(), "  %s %-16s %-34s [%s, %s, %s, %s]\n",
			marker, e.GetId(), e.GetDisplayName(), e.GetKind(), streaming, availabilityLabel(e.GetAvailable()), e.GetVerdict())
		fmt.Fprintf(ctx.Stdout(), "      verdict=%s reason=%s\n", e.GetVerdict(), e.GetReason())
	}
	fmt.Fprintf(ctx.Stdout(), "(* = active; set with `audio-tools stt stream-config-set --engine <id>`)\n")
	return nil
}

// engineImpact reports the shared-resource impact of switching away from an
// engine: which other scenarios still depend on its backing resource and the
// exact (never auto-run) command to stop it. audio-tools never stops a shared
// resource itself — this surfaces the decision for the operator.
func (h *handlers) engineImpactPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(
		func(ctx cliapp.OperationContext) (*sttv1.GetEngineSwitchImpactResponse, error) {
			engineID := ctx.Positional("engine")
			if engineID == "" {
				return nil, fmt.Errorf("provide the engine id to assess (see `audio-tools stt engines`)")
			}
			resp, err := h.admin.GetEngineSwitchImpact(context.Background(), connect.NewRequest(&sttv1.GetEngineSwitchImpactRequest{
				FromEngineId: engineID,
			}))
			if err != nil {
				return nil, cliapp.WrapAPIError("engine-impact", err, nil)
			}
			if resp == nil || resp.Msg == nil {
				return nil, fmt.Errorf("server returned no engine switch impact")
			}
			return resp.Msg, nil
		},
		renderEngineImpact,
		nil,
	)
}

func renderEngineImpact(ctx cliapp.RunContext, msg *sttv1.GetEngineSwitchImpactResponse) error {
	if msg.GetResource() == "" {
		fmt.Fprintf(ctx.Stdout(), "Engine %q has no local backing resource — switching away frees nothing to stop.\n", ctx.Positional("engine"))
		return nil
	}
	fmt.Fprintf(ctx.Stdout(), "Switching away from %q would stop using resource %q.\n", ctx.Positional("engine"), msg.GetResource())
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
func (h *handlers) streamConfigPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(_ cliapp.OperationContext) (*sttv1.GetStreamConfigResponse, error) {
		resp, err := h.admin.GetStreamConfig(context.Background(), connect.NewRequest(&sttv1.GetStreamConfigRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get-stream-config", err, nil)
		}
		if resp == nil || resp.Msg == nil {
			return nil, fmt.Errorf("server returned no stream config")
		}
		if resp.Msg.GetConfig() == nil {
			return nil, fmt.Errorf("server returned no stream config")
		}
		return resp.Msg, nil
	}, renderStreamConfig, renderStreamConfig)
}

func (h *handlers) streamConfigGet(ctx cliapp.RunContext) error {
	return h.streamConfigPrimitive().Run(ctx)
}

func renderStreamConfig(ctx cliapp.RunContext, resp *sttv1.GetStreamConfigResponse) error {
	cfg := resp.GetConfig()
	fmt.Fprintf(ctx.Stdout(), "Streaming STT pipeline:\n")
	fmt.Fprintf(ctx.Stdout(), "  engine_id            = %s\n", stringOrDefault(cfg.GetEngineId(), "auto (resolver)"))
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
func (h *handlers) streamConfigSetPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*sttv1.UpdateStreamConfigResponse, error) {
		resp, err := h.updateStreamConfig(ctx)
		return resp, err
	}, renderStreamConfigUpdate, renderStreamConfigUpdate)
}

// streamConfigUpdateFromFlags builds the sparse config payload and FieldMask
// in CLI flag order, preserving which invalid flag is reported first.
func streamConfigUpdateFromFlags(ctx cliapp.OperationContext) (*sttv1.StreamConfig, *fieldmaskpb.FieldMask, error) {
	mask := &fieldmaskpb.FieldMask{}
	cfg := &sttv1.StreamConfig{}
	if err := applyStreamConfigPipelineFlags(ctx, cfg, mask); err != nil {
		return nil, nil, err
	}
	if err := applyStreamConfigQualityFlags(ctx, cfg, mask); err != nil {
		return nil, nil, err
	}
	if len(mask.Paths) == 0 {
		return nil, nil, fmt.Errorf("at least one streaming/egress lever flag must be set (e.g. --streaming-mode, --strategy-preference, --vad-silence-ms, --overlap-window-ms, --overlap-commit-runs, --overlap-max-stall-rejects, --hallucination-filter, --vad-filter, --no-speech-threshold, --logprob-threshold)")
	}
	return cfg, mask, nil
}

// applyStreamConfigPipelineFlags processes the pipeline flags before the
// quality and ingress flags, matching the CLI's validation and mask order.
func applyStreamConfigPipelineFlags(ctx cliapp.OperationContext, cfg *sttv1.StreamConfig, mask *fieldmaskpb.FieldMask) error {
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
	return nil
}

// applyStreamConfigQualityFlags processes post-recognition quality flags and
// then the ingress denoise flag, preserving their FieldMask order.
func applyStreamConfigQualityFlags(ctx cliapp.OperationContext, cfg *sttv1.StreamConfig, mask *fieldmaskpb.FieldMask) error {
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
	return applyBoolFlag(ctx, "denoise", "denoise_enabled", mask, func(value bool) { cfg.DenoiseEnabled = value })
}

func (h *handlers) updateStreamConfig(ctx cliapp.OperationContext) (*sttv1.UpdateStreamConfigResponse, error) {
	cfg, mask, err := streamConfigUpdateFromFlags(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := h.admin.UpdateStreamConfig(context.Background(), connect.NewRequest(&sttv1.UpdateStreamConfigRequest{
		UpdateMask: mask,
		Config:     cfg,
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("update-stream-config", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.GetConfig() == nil {
		return nil, fmt.Errorf("server returned no stream config")
	}
	return resp.Msg, nil
}

func renderStreamConfigUpdate(ctx cliapp.RunContext, resp *sttv1.UpdateStreamConfigResponse) error {
	out := resp.GetConfig()
	fmt.Fprintf(ctx.Stdout(), "Updated. Resolved streaming STT pipeline:\n")
	fmt.Fprintf(ctx.Stdout(), "  engine_id            = %s\n", stringOrDefault(out.GetEngineId(), "auto (resolver)"))
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
