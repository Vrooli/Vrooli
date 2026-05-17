// Stream-config administration handlers (Get/Update) backed by the
// JSON-singleton stream-config store. Also exposes the projection
// the TranscribeStream handler uses to derive selector StreamConfig.
package stt

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"

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
	if d.VadSilenceMs != 0 {
		cfg.VADSilenceMs = int(d.VadSilenceMs)
	}
	if d.OverlapWindowMs != 0 {
		cfg.OverlapWindowMs = int(d.OverlapWindowMs)
	}
	if d.OverlapCommitRuns != 0 {
		cfg.OverlapCommitRuns = int(d.OverlapCommitRuns)
	}
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
	return d, nil
}

func (h *connectHandler) GetStreamConfig(ctx context.Context, _ *connect.Request[sttv1.GetStreamConfigRequest]) (*connect.Response[sttv1.GetStreamConfigResponse], error) {
	d, err := h.loadStreamCfg(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&sttv1.GetStreamConfigResponse{Config: d.toProto()}), nil
}

func (h *connectHandler) UpdateStreamConfig(ctx context.Context, req *connect.Request[sttv1.UpdateStreamConfigRequest]) (*connect.Response[sttv1.UpdateStreamConfigResponse], error) {
	d, err := h.loadStreamCfg(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	m := req.Msg
	if m.GetHasFlushIntervalMs() {
		d.FlushIntervalMs = m.GetFlushIntervalMs()
	}
	if m.GetHasMinDeltaBytes() {
		d.MinDeltaBytes = m.GetMinDeltaBytes()
	}
	if m.GetHasOverlapBytes() {
		d.OverlapBytes = m.GetOverlapBytes()
	}
	if m.GetHasPersistentMode() {
		d.PersistentMode = m.GetPersistentMode()
	}
	if m.GetHasWakeWordEnabled() {
		d.WakeWordEnabled = m.GetWakeWordEnabled()
	}
	if m.GetHasWakeWordThreshold() {
		d.WakeWordThreshold = m.GetWakeWordThreshold()
	}
	if m.GetHasSegmentSilenceMs() {
		d.SegmentSilenceMs = m.GetSegmentSilenceMs()
	}
	if m.GetHasStreamingMode() {
		d.StreamingMode = m.GetStreamingMode()
	}
	if m.GetHasStrategyPreference() {
		d.StrategyPreference = m.GetStrategyPreference()
	}
	if m.GetHasVadSilenceMs() {
		d.VadSilenceMs = m.GetVadSilenceMs()
	}
	if m.GetHasOverlapWindowMs() {
		d.OverlapWindowMs = m.GetOverlapWindowMs()
	}
	if m.GetHasOverlapCommitRuns() {
		d.OverlapCommitRuns = m.GetOverlapCommitRuns()
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
