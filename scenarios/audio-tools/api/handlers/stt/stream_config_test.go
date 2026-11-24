package stt

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestValidateStreamingLeversBoundaries(t *testing.T) {
	tests := []struct {
		name string
		doc  streamCfgDoc
		want string
	}{
		{name: "empty values use defaults", doc: streamCfgDoc{}},
		{name: "supported mode and strategy", doc: streamCfgDoc{StreamingMode: "off", StrategyPreference: "passthrough"}},
		{name: "timing lower boundaries", doc: streamCfgDoc{VadSilenceMs: 200, OverlapWindowMs: 1000, OverlapCommitRuns: 2}},
		{name: "timing upper boundaries", doc: streamCfgDoc{VadSilenceMs: 3000, OverlapWindowMs: 5000, OverlapCommitRuns: 4}},
		{name: "stall fallback explicit disabled", doc: streamCfgDoc{OverlapMaxStallRejects: int32Ptr(0)}},
		{name: "stall fallback upper boundary", doc: streamCfgDoc{OverlapMaxStallRejects: int32Ptr(10)}},
		{name: "threshold boundaries", doc: streamCfgDoc{NoSpeechThreshold: 1, LogProbThreshold: -10}},
		{name: "zero thresholds select defaults", doc: streamCfgDoc{NoSpeechThreshold: 0, LogProbThreshold: 0}},
		{name: "unsupported mode", doc: streamCfgDoc{StreamingMode: "enabled"}, want: `streaming_mode must be "auto" or "off", got "enabled"`},
		{name: "unsupported strategy", doc: streamCfgDoc{StrategyPreference: "native"}, want: "strategy_preference must be one of auto|vad|overlap|passthrough, got \"native\""},
		{name: "VAD timing below range", doc: streamCfgDoc{VadSilenceMs: 199}, want: "vad_silence_ms must be in [200, 3000], got 199"},
		{name: "VAD timing above range", doc: streamCfgDoc{VadSilenceMs: 3001}, want: "vad_silence_ms must be in [200, 3000], got 3001"},
		{name: "overlap window below range", doc: streamCfgDoc{OverlapWindowMs: 999}, want: "overlap_window_ms must be in [1000, 5000], got 999"},
		{name: "overlap window above range", doc: streamCfgDoc{OverlapWindowMs: 5001}, want: "overlap_window_ms must be in [1000, 5000], got 5001"},
		{name: "commit runs below range", doc: streamCfgDoc{OverlapCommitRuns: 1}, want: "overlap_commit_runs must be in [2, 4], got 1"},
		{name: "commit runs above range", doc: streamCfgDoc{OverlapCommitRuns: 5}, want: "overlap_commit_runs must be in [2, 4], got 5"},
		{name: "stall fallback below range", doc: streamCfgDoc{OverlapMaxStallRejects: int32Ptr(-1)}, want: "overlap_max_stall_rejects must be in [0, 10] (0 disables), got -1"},
		{name: "stall fallback above range", doc: streamCfgDoc{OverlapMaxStallRejects: int32Ptr(11)}, want: "overlap_max_stall_rejects must be in [0, 10] (0 disables), got 11"},
		{name: "negative no-speech threshold", doc: streamCfgDoc{NoSpeechThreshold: -0.01}, want: "no_speech_threshold must be in (0, 1], got -0.010000"},
		{name: "no-speech threshold above range", doc: streamCfgDoc{NoSpeechThreshold: 1.01}, want: "no_speech_threshold must be in (0, 1], got 1.010000"},
		{name: "log-probability threshold below range", doc: streamCfgDoc{LogProbThreshold: -10.01}, want: "logprob_threshold must be in [-10, 0), got -10.010000"},
		{name: "log-probability threshold above range", doc: streamCfgDoc{LogProbThreshold: 0.01}, want: "logprob_threshold must be in [-10, 0), got 0.010000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStreamingLevers(tt.doc)
			if tt.want == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.want)
		})
	}
}

func TestBackfillStreamCfgDefaultsPreservesPresenceTrackedValues(t *testing.T) {
	got := backfillStreamCfgDefaults(streamCfgDoc{
		OverlapMaxStallRejects:     int32Ptr(0),
		HallucinationFilterEnabled: boolPtr(false),
		VADFilterEnabled:           boolPtr(false),
		DenoiseEnabled:             false,
		StreamingMode:              "off",
	})

	require.Equal(t, int32(0), *got.OverlapMaxStallRejects, "explicit zero disables the stall fallback")
	require.False(t, *got.HallucinationFilterEnabled, "explicit false must not be replaced by the default")
	require.False(t, *got.VADFilterEnabled, "explicit false must not be replaced by the default")
	require.False(t, got.DenoiseEnabled, "denoise defaults off without presence tracking")
	require.Equal(t, "off", got.StreamingMode, "explicit non-default strings must be preserved")

	defaults := defaultStreamCfg()
	legacy := backfillStreamCfgDefaults(streamCfgDoc{FlushIntervalMs: 333})
	require.Equal(t, int32(333), legacy.FlushIntervalMs, "existing non-zero values must be preserved")
	require.Equal(t, defaults.VadSilenceMs, legacy.VadSilenceMs, "legacy missing numeric fields must receive defaults")
	require.Equal(t, defaults.OverlapMaxStallRejects, legacy.OverlapMaxStallRejects, "legacy missing presence fields must receive defaults")
	require.Equal(t, defaults.HallucinationFilterEnabled, legacy.HallucinationFilterEnabled)
}

func TestValidateStreamConfigUpdateMaskErrors(t *testing.T) {
	require.EqualError(t, validateStreamConfigUpdateMask(nil), "update_mask required")
	require.EqualError(t, validateStreamConfigUpdateMask(&fieldmaskpb.FieldMask{}), "update_mask required")
	require.NoError(t, validateStreamConfigUpdateMask(&fieldmaskpb.FieldMask{Paths: []string{"denoise_enabled"}}))
	require.EqualError(t,
		validateStreamConfigUpdateMask(&fieldmaskpb.FieldMask{Paths: []string{"bogus"}}),
		"unknown update_mask paths: [bogus]",
	)
}
