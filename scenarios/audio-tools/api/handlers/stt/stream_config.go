// Stream-config administration handlers (Get/Update) backed by the
// JSON-singleton stream-config store. Also exposes the projection
// the TranscribeStream handler uses to derive selector StreamConfig.
package stt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	"audio-tools/internal/protomap"
	sttpkg "audio-tools/internal/stt"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// resolveStreamPipelineConfig loads the persisted StreamConfig and
// projects the streaming-pipeline lever subset into the selector's
// StreamConfig shape, falling back to documented defaults when fields
// are missing.
func (h *connectHandler) resolveStreamPipelineConfig(ctx context.Context) sttpkg.StreamConfig {
	d, err := h.loadStreamCfg(ctx)
	if err != nil {
		return sttpkg.Defaults()
	}
	cfg := sttpkg.Defaults()
	if d.StreamingMode != "" {
		cfg.Mode = sttpkg.StreamingMode(d.StreamingMode)
	}
	if d.StrategyPreference != "" {
		cfg.StrategyPreference = sttpkg.StrategyPreference(d.StrategyPreference)
	}
	if d.EngineID != "" {
		cfg.EngineID = d.EngineID
	}
	if d.VadSilenceMs != 0 {
		cfg.VADSilenceMs = int(d.VadSilenceMs)
	}
	if d.OverlapWindowMs != 0 {
		cfg.OverlapWindowMs = int(d.OverlapWindowMs)
	}
	if d.OverlapCommitRuns != 0 {
		cfg.OverlapCommitRuns = int(d.OverlapCommitRuns)
	}
	// Egress-gate levers. loadStreamCfg has already backfilled the *bool
	// toggles and zero thresholds, so these reads are authoritative.
	cfg.HallucinationFilterEnabled = boolOrTrue(d.HallucinationFilterEnabled)
	cfg.VADFilterEnabled = boolOrTrue(d.VADFilterEnabled)
	if d.NoSpeechThreshold != 0 {
		cfg.NoSpeechThreshold = d.NoSpeechThreshold
	}
	if d.LogProbThreshold != 0 {
		cfg.LogProbThreshold = d.LogProbThreshold
	}
	// Ingress denoise: plain bool, default off — read directly.
	cfg.DenoiseEnabled = d.DenoiseEnabled
	return cfg
}

func validateStreamingLevers(d streamCfgDoc) error {
	switch d.StreamingMode {
	case "", "auto", "off":
	default:
		return fmt.Errorf("streaming_mode must be \"auto\" or \"off\", got %q", d.StreamingMode)
	}
	switch d.StrategyPreference {
	case "", "auto", "vad", "overlap", "passthrough":
	default:
		return fmt.Errorf("strategy_preference must be one of auto|vad|overlap|passthrough, got %q", d.StrategyPreference)
	}
	if d.VadSilenceMs != 0 && (d.VadSilenceMs < 200 || d.VadSilenceMs > 3000) {
		return fmt.Errorf("vad_silence_ms must be in [200, 3000], got %d", d.VadSilenceMs)
	}
	if d.OverlapWindowMs != 0 && (d.OverlapWindowMs < 1000 || d.OverlapWindowMs > 5000) {
		return fmt.Errorf("overlap_window_ms must be in [1000, 5000], got %d", d.OverlapWindowMs)
	}
	if d.OverlapCommitRuns != 0 && (d.OverlapCommitRuns < 2 || d.OverlapCommitRuns > 4) {
		return fmt.Errorf("overlap_commit_runs must be in [2, 4], got %d", d.OverlapCommitRuns)
	}
	// Egress-gate thresholds: 0 means "use default" (backfilled), so only
	// non-zero values are range-checked. no_speech_prob is a probability;
	// avg_logprob is negative (log of a probability).
	if d.NoSpeechThreshold != 0 && (d.NoSpeechThreshold < 0 || d.NoSpeechThreshold > 1) {
		return fmt.Errorf("no_speech_threshold must be in (0, 1], got %f", d.NoSpeechThreshold)
	}
	if d.LogProbThreshold != 0 && (d.LogProbThreshold < -10 || d.LogProbThreshold > 0) {
		return fmt.Errorf("logprob_threshold must be in [-10, 0), got %f", d.LogProbThreshold)
	}
	return nil
}

func (h *connectHandler) loadStreamCfg(ctx context.Context) (streamCfgDoc, error) {
	if h.deps.StreamConfig == nil {
		return defaultStreamCfg(), nil
	}
	raw, ok, err := h.deps.StreamConfig.Get(ctx)
	if err != nil {
		return streamCfgDoc{}, err
	}
	if !ok || raw == "" {
		return defaultStreamCfg(), nil
	}
	var d streamCfgDoc
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return defaultStreamCfg(), nil
	}
	return backfillStreamCfgDefaults(d), nil
}

// backfillStreamCfgDefaults replaces any zero-valued field with its
// defaultStreamCfg() value. Critical for docs persisted before a field
// existed: without this, an old doc returns 0 for new fields, which
// causes server-side resolveStreamPipelineConfig to fall back to
// sttpkg.Defaults() while toProto() ships 0 to the client — the two
// sides then disagree on the timing the user can see (mic-button ring
// vs server-side VAD cut). All zero-as-meaningful fields are bool, so
// they cannot use this backfill pattern; here only numeric/string
// timing levers are restored.
func backfillStreamCfgDefaults(d streamCfgDoc) streamCfgDoc {
	def := defaultStreamCfg()
	if d.FlushIntervalMs == 0 {
		d.FlushIntervalMs = def.FlushIntervalMs
	}
	if d.MinDeltaBytes == 0 {
		d.MinDeltaBytes = def.MinDeltaBytes
	}
	if d.OverlapBytes == 0 {
		d.OverlapBytes = def.OverlapBytes
	}
	if d.WakeWordThreshold == 0 {
		d.WakeWordThreshold = def.WakeWordThreshold
	}
	if d.SegmentSilenceMs == 0 {
		d.SegmentSilenceMs = def.SegmentSilenceMs
	}
	if d.StreamingMode == "" {
		d.StreamingMode = def.StreamingMode
	}
	if d.StrategyPreference == "" {
		d.StrategyPreference = def.StrategyPreference
	}
	if d.EngineID == "" {
		d.EngineID = def.EngineID
	}
	if d.VadSilenceMs == 0 {
		d.VadSilenceMs = def.VadSilenceMs
	}
	if d.OverlapWindowMs == 0 {
		d.OverlapWindowMs = def.OverlapWindowMs
	}
	if d.OverlapCommitRuns == 0 {
		d.OverlapCommitRuns = def.OverlapCommitRuns
	}
	// Egress-gate levers. The *bool toggles backfill via presence (nil =
	// absent → default true); the thresholds backfill zero → default.
	if d.HallucinationFilterEnabled == nil {
		d.HallucinationFilterEnabled = def.HallucinationFilterEnabled
	}
	if d.VADFilterEnabled == nil {
		d.VADFilterEnabled = def.VADFilterEnabled
	}
	if d.NoSpeechThreshold == 0 {
		d.NoSpeechThreshold = def.NoSpeechThreshold
	}
	if d.LogProbThreshold == 0 {
		d.LogProbThreshold = def.LogProbThreshold
	}
	return d
}

func (h *connectHandler) GetStreamConfig(ctx context.Context, _ *connect.Request[sttv1.GetStreamConfigRequest]) (*connect.Response[sttv1.GetStreamConfigResponse], error) {
	d, err := h.loadStreamCfg(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&sttv1.GetStreamConfigResponse{Config: d.toProto()}), nil
}

var streamConfigAllowedPaths = map[string]struct{}{
	"flush_interval_ms":   {},
	"min_delta_bytes":     {},
	"overlap_bytes":       {},
	"persistent_mode":     {},
	"wake_word_enabled":   {},
	"wake_word_threshold": {},
	"segment_silence_ms":  {},
	"streaming_mode":      {},
	"strategy_preference": {},
	"engine_id":           {},
	"vad_silence_ms":      {},
	"overlap_window_ms":   {},
	"overlap_commit_runs": {},

	"hallucination_filter_enabled": {},
	"vad_filter_enabled":           {},
	"no_speech_threshold":          {},
	"logprob_threshold":            {},
	"denoise_enabled":              {},
}

func (h *connectHandler) UpdateStreamConfig(ctx context.Context, req *connect.Request[sttv1.UpdateStreamConfigRequest]) (*connect.Response[sttv1.UpdateStreamConfigResponse], error) {
	d, err := h.loadStreamCfg(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	mask := req.Msg.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask required"))
	}
	if bad := protomap.MaskPathsOutsideAllowed(mask, streamConfigAllowedPaths); len(bad) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown update_mask paths: %v", bad))
	}
	cfg := req.Msg.GetConfig()
	if protomap.MaskHas(mask, "flush_interval_ms") {
		d.FlushIntervalMs = cfg.GetFlushIntervalMs()
	}
	if protomap.MaskHas(mask, "min_delta_bytes") {
		d.MinDeltaBytes = cfg.GetMinDeltaBytes()
	}
	if protomap.MaskHas(mask, "overlap_bytes") {
		d.OverlapBytes = cfg.GetOverlapBytes()
	}
	if protomap.MaskHas(mask, "persistent_mode") {
		d.PersistentMode = cfg.GetPersistentMode()
	}
	if protomap.MaskHas(mask, "wake_word_enabled") {
		d.WakeWordEnabled = cfg.GetWakeWordEnabled()
	}
	if protomap.MaskHas(mask, "wake_word_threshold") {
		d.WakeWordThreshold = cfg.GetWakeWordThreshold()
	}
	if protomap.MaskHas(mask, "segment_silence_ms") {
		d.SegmentSilenceMs = cfg.GetSegmentSilenceMs()
	}
	if protomap.MaskHas(mask, "streaming_mode") {
		d.StreamingMode = protomap.StreamingModeFromProto(cfg.GetStreamingMode())
	}
	if protomap.MaskHas(mask, "strategy_preference") {
		d.StrategyPreference = protomap.StrategyPreferenceFromProto(cfg.GetStrategyPreference())
	}
	if protomap.MaskHas(mask, "engine_id") {
		engineID := cfg.GetEngineId()
		if engineID != "" && h.deps.Registry != nil {
			if _, ok := h.deps.Registry.Engine(engineID); !ok {
				return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown engine_id %q (see ListEngines)", engineID))
			}
		}
		d.EngineID = engineID
	}
	if protomap.MaskHas(mask, "vad_silence_ms") {
		d.VadSilenceMs = cfg.GetVadSilenceMs()
	}
	if protomap.MaskHas(mask, "overlap_window_ms") {
		d.OverlapWindowMs = cfg.GetOverlapWindowMs()
	}
	if protomap.MaskHas(mask, "overlap_commit_runs") {
		d.OverlapCommitRuns = cfg.GetOverlapCommitRuns()
	}
	if protomap.MaskHas(mask, "hallucination_filter_enabled") {
		d.HallucinationFilterEnabled = boolPtr(cfg.GetHallucinationFilterEnabled())
	}
	if protomap.MaskHas(mask, "vad_filter_enabled") {
		d.VADFilterEnabled = boolPtr(cfg.GetVadFilterEnabled())
	}
	if protomap.MaskHas(mask, "no_speech_threshold") {
		d.NoSpeechThreshold = cfg.GetNoSpeechThreshold()
	}
	if protomap.MaskHas(mask, "logprob_threshold") {
		d.LogProbThreshold = cfg.GetLogprobThreshold()
	}
	if protomap.MaskHas(mask, "denoise_enabled") {
		d.DenoiseEnabled = cfg.GetDenoiseEnabled()
	}
	if err := validateStreamingLevers(d); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if h.deps.StreamConfig != nil {
		raw, _ := json.Marshal(d)
		if err := h.deps.StreamConfig.Set(ctx, string(raw)); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&sttv1.UpdateStreamConfigResponse{Config: d.toProto()}), nil
}
