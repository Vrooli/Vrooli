// Pure proto<->domain mappers and default-config builders for the
// stt package. These functions hold no I/O and depend on no handler
// state — they are safe to call from any goroutine.
package stt

import (
	"audio-tools/internal/protomap"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// speakerCfgDoc is the JSON view of SpeakerConfig.
type speakerCfgDoc struct {
	Enabled                     bool     `json:"enabled"`
	ProfileIDs                  []string `json:"profile_ids"`
	Threshold                   float64  `json:"threshold"`
	Mode                        string   `json:"mode"`
	RejectBehavior              string   `json:"reject_behavior"`
	FallbackWithoutVerification bool     `json:"fallback_without_verification"`
	ExtractionEnabled           bool     `json:"extraction_enabled"`
}

func defaultSpeakerCfg() speakerCfgDoc {
	return speakerCfgDoc{
		Enabled: false, ProfileIDs: []string{}, Threshold: 0.7,
		Mode: "off", RejectBehavior: "drop",
	}
}

func (d speakerCfgDoc) toProto() *sttv1.SpeakerConfig {
	return &sttv1.SpeakerConfig{
		Enabled:                     d.Enabled,
		ProfileIds:                  d.ProfileIDs,
		Threshold:                   d.Threshold,
		Mode:                        protomap.SpeakerModeToProto(d.Mode),
		RejectBehavior:              protomap.RejectBehaviorToProto(d.RejectBehavior),
		FallbackWithoutVerification: d.FallbackWithoutVerification,
		ExtractionEnabled:           d.ExtractionEnabled,
	}
}

// streamCfgDoc is the on-disk representation of StreamConfig.
type streamCfgDoc struct {
	FlushIntervalMs   int32   `json:"flush_interval_ms"`
	MinDeltaBytes     int32   `json:"min_delta_bytes"`
	OverlapBytes      int32   `json:"overlap_bytes"`
	PersistentMode    bool    `json:"persistent_mode"`
	WakeWordEnabled   bool    `json:"wake_word_enabled"`
	WakeWordThreshold float64 `json:"wake_word_threshold"`
	SegmentSilenceMs  int32   `json:"segment_silence_ms"`

	StreamingMode      string `json:"streaming_mode"`
	StrategyPreference string `json:"strategy_preference"`
	EngineID           string `json:"engine_id"`
	VadSilenceMs       int32  `json:"vad_silence_ms"`
	OverlapWindowMs    int32  `json:"overlap_window_ms"`
	OverlapCommitRuns  int32  `json:"overlap_commit_runs"`

	// Egress-gate levers. The two filter toggles are *bool so an absent
	// field (a doc written before the field existed) backfills to its
	// default-true value rather than reading as a deliberate "off".
	HallucinationFilterEnabled *bool   `json:"hallucination_filter_enabled,omitempty"`
	VADFilterEnabled           *bool   `json:"vad_filter_enabled,omitempty"`
	NoSpeechThreshold          float64 `json:"no_speech_threshold"`
	LogProbThreshold           float64 `json:"logprob_threshold"`
}

func (d streamCfgDoc) toProto() *sttv1.StreamConfig {
	return &sttv1.StreamConfig{
		FlushIntervalMs:            d.FlushIntervalMs,
		MinDeltaBytes:              d.MinDeltaBytes,
		OverlapBytes:               d.OverlapBytes,
		PersistentMode:             d.PersistentMode,
		WakeWordEnabled:            d.WakeWordEnabled,
		WakeWordThreshold:          d.WakeWordThreshold,
		SegmentSilenceMs:           d.SegmentSilenceMs,
		StreamingMode:              protomap.StreamingModeToProto(d.StreamingMode),
		StrategyPreference:         protomap.StrategyPreferenceToProto(d.StrategyPreference),
		EngineId:                   d.EngineID,
		VadSilenceMs:               d.VadSilenceMs,
		OverlapWindowMs:            d.OverlapWindowMs,
		OverlapCommitRuns:          d.OverlapCommitRuns,
		HallucinationFilterEnabled: boolOrTrue(d.HallucinationFilterEnabled),
		VadFilterEnabled:           boolOrTrue(d.VADFilterEnabled),
		NoSpeechThreshold:          d.NoSpeechThreshold,
		LogprobThreshold:           d.LogProbThreshold,
	}
}

// boolOrTrue dereferences a presence-tracked bool, defaulting absent (nil)
// to true — the documented default for both egress filter toggles.
func boolOrTrue(p *bool) bool {
	if p == nil {
		return true
	}
	return *p
}

func boolPtr(b bool) *bool { return &b }

func defaultStreamCfg() streamCfgDoc {
	return streamCfgDoc{
		FlushIntervalMs: 250, MinDeltaBytes: 16384, OverlapBytes: 2048,
		PersistentMode: false, WakeWordEnabled: false, WakeWordThreshold: 0.6,
		SegmentSilenceMs:           800,
		StreamingMode:              "auto",
		StrategyPreference:         "auto",
		EngineID:                   "whisper-local",
		VadSilenceMs:               1200,
		OverlapWindowMs:            2000,
		OverlapCommitRuns:          2,
		HallucinationFilterEnabled: boolPtr(true),
		VADFilterEnabled:           boolPtr(true),
		NoSpeechThreshold:          0.6,
		LogProbThreshold:           -1.0,
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
