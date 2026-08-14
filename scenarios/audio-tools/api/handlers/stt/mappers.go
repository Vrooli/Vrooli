// Pure proto<->domain mappers and default-config builders for the
// stt package. These functions hold no I/O and depend on no handler
// state — they are safe to call from any goroutine.
package stt

import (
	"audio-tools/internal/protomap"
	sttpkg "audio-tools/internal/stt"
	sttpipeline "audio-tools/internal/stt/pipeline"

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
	// Session-decision tuning (warm-up window + EMA smoothing). Zero means the
	// session verifier applies its built-in default.
	MinDecisionSeconds float64 `json:"min_decision_seconds"`
	ScoreSmoothing     float64 `json:"score_smoothing"`
}

func defaultSpeakerCfg() speakerCfgDoc {
	return speakerCfgDoc{
		Enabled: false, ProfileIDs: []string{}, Threshold: 0.5,
		Mode: "off", RejectBehavior: "drop",
		FallbackWithoutVerification: false,
		ExtractionEnabled:           false,
		MinDecisionSeconds:          sttpipeline.DefaultMinDecisionSeconds,
		ScoreSmoothing:              sttpipeline.DefaultScoreSmoothing,
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
		MinDecisionSeconds:          d.MinDecisionSeconds,
		ScoreSmoothing:              d.ScoreSmoothing,
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
	// OverlapMaxStallRejects is the OverlapAgree stall-fallback policy. It
	// is a *int32 (not a plain int32) because 0 is MEANINGFUL here — it
	// disables the fallback — so an absent field (a doc written before the
	// field existed) must NOT read as a deliberate "disabled". Presence
	// tracking: nil backfills to the default (0); an explicit 0 stays
	// disabled. Same rationale as the egress *bool toggles above.
	OverlapMaxStallRejects *int32 `json:"overlap_max_stall_rejects,omitempty"`

	// Egress-gate levers. The two filter toggles are *bool so an absent
	// field (a doc written before the field existed) backfills to its
	// default-true value rather than reading as a deliberate "off".
	HallucinationFilterEnabled *bool   `json:"hallucination_filter_enabled,omitempty"`
	VADFilterEnabled           *bool   `json:"vad_filter_enabled,omitempty"`
	NoSpeechThreshold          float64 `json:"no_speech_threshold"`
	LogProbThreshold           float64 `json:"logprob_threshold"`

	// DenoiseEnabled toggles the pre-recognition ingress denoise stage. A plain
	// bool (not *bool) because the default is OFF — an absent field reads as
	// false, which is exactly the intended default, so no presence tracking is
	// needed (unlike the default-true egress filters above).
	DenoiseEnabled bool `json:"denoise_enabled"`
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
		OverlapMaxStallRejects:     int32OrDefault(d.OverlapMaxStallRejects, 0),
		HallucinationFilterEnabled: boolOrTrue(d.HallucinationFilterEnabled),
		VadFilterEnabled:           boolOrTrue(d.VADFilterEnabled),
		NoSpeechThreshold:          d.NoSpeechThreshold,
		LogprobThreshold:           d.LogProbThreshold,
		DenoiseEnabled:             d.DenoiseEnabled,
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

// int32Ptr boxes an int32 for presence-tracked config fields.
func int32Ptr(v int32) *int32 { return &v }

// int32OrDefault dereferences a presence-tracked int32, substituting def
// when the pointer is absent (nil). An explicit value — including 0 — is
// honored verbatim.
func int32OrDefault(p *int32, def int32) int32 {
	if p == nil {
		return def
	}
	return *p
}

func defaultStreamCfg() streamCfgDoc {
	return streamCfgDoc{
		FlushIntervalMs: 250, MinDeltaBytes: 16384, OverlapBytes: 2048,
		PersistentMode: false, WakeWordEnabled: false, WakeWordThreshold: 0.6,
		SegmentSilenceMs:           800,
		StreamingMode:              "auto",
		StrategyPreference:         "auto",
		EngineID:                   "whisper-local",
		VadSilenceMs:               sttpkg.DefaultVADSilenceMs,
		OverlapWindowMs:            2000,
		OverlapCommitRuns:          2,
		OverlapMaxStallRejects:     int32Ptr(0),
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
