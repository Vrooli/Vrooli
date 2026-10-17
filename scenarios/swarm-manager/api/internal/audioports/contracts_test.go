package audioports

import (
	"reflect"
	"testing"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// -----------------------------------------------------------------------------
// Enum toProto / fromProto
// -----------------------------------------------------------------------------

func TestAudioFormatToProto(t *testing.T) {
	cases := []struct {
		in   AudioFormat
		want commonv1.AudioFormat
	}{
		{AudioFormatUnspecified, commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED},
		{AudioFormatWAV, commonv1.AudioFormat_AUDIO_FORMAT_WAV},
		{AudioFormatAAC, commonv1.AudioFormat_AUDIO_FORMAT_AAC},
		{AudioFormat(-1), commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED}, // out of range low
		{AudioFormat(99), commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED}, // out of range high
	}
	for _, c := range cases {
		if got := c.in.toProto(); got != c.want {
			t.Errorf("AudioFormat(%d).toProto() = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestResponseFormatRoundTrip(t *testing.T) {
	for _, v := range []ResponseFormat{
		ResponseFormatUnspecified, ResponseFormatMP3, ResponseFormatWAV, ResponseFormatOPUS, ResponseFormatFLAC,
	} {
		if got := responseFormatFromProto(v.toProto()); got != v {
			t.Errorf("ResponseFormat round-trip %d = %d", v, got)
		}
	}
	// out of range clamps to unspecified
	if got := ResponseFormat(99).toProto(); got != commonv1.ResponseFormat_RESPONSE_FORMAT_UNSPECIFIED {
		t.Errorf("out-of-range ResponseFormat = %v", got)
	}
	if got := ResponseFormat(-1).toProto(); got != commonv1.ResponseFormat_RESPONSE_FORMAT_UNSPECIFIED {
		t.Errorf("negative ResponseFormat = %v", got)
	}
}

func TestProviderTierFromProto(t *testing.T) {
	cases := map[commonv1.ProviderTier]ProviderTier{
		commonv1.ProviderTier_PROVIDER_TIER_UNSPECIFIED: ProviderTierUnspecified,
		commonv1.ProviderTier_PROVIDER_TIER_LOCAL:       ProviderTierLocal,
		commonv1.ProviderTier_PROVIDER_TIER_BYOK:        ProviderTierBYOK,
		commonv1.ProviderTier_PROVIDER_TIER_VROOLI:      ProviderTierVrooli,
	}
	for in, want := range cases {
		if got := providerTierFromProto(in); got != want {
			t.Errorf("providerTierFromProto(%v) = %d, want %d", in, got, want)
		}
	}
}

func TestSpeakerModeRoundTrip(t *testing.T) {
	for _, v := range []SpeakerMode{SpeakerModeUnspecified, SpeakerModeOff, SpeakerModeFilter, SpeakerModeAdvisory} {
		if got := speakerModeFromProto(v.toProto()); got != v {
			t.Errorf("SpeakerMode round-trip %d = %d", v, got)
		}
	}
	if got := SpeakerMode(99).toProto(); got != sttv1.SpeakerMode_SPEAKER_MODE_UNSPECIFIED {
		t.Errorf("out-of-range SpeakerMode = %v", got)
	}
}

func TestRejectBehaviorRoundTrip(t *testing.T) {
	for _, v := range []RejectBehavior{RejectBehaviorUnspecified, RejectBehaviorDrop, RejectBehaviorShowMuted} {
		if got := rejectBehaviorFromProto(v.toProto()); got != v {
			t.Errorf("RejectBehavior round-trip %d = %d", v, got)
		}
	}
	if got := RejectBehavior(99).toProto(); got != sttv1.RejectBehavior_REJECT_BEHAVIOR_UNSPECIFIED {
		t.Errorf("out-of-range RejectBehavior = %v", got)
	}
}

func TestStreamingModeRoundTrip(t *testing.T) {
	for _, v := range []StreamingMode{StreamingModeUnspecified, StreamingModeAuto, StreamingModeOff} {
		if got := streamingModeFromProto(v.toProto()); got != v {
			t.Errorf("StreamingMode round-trip %d = %d", v, got)
		}
	}
	if got := StreamingMode(99).toProto(); got != sttv1.StreamingMode_STREAMING_MODE_UNSPECIFIED {
		t.Errorf("out-of-range StreamingMode = %v", got)
	}
}

func TestStrategyPreferenceRoundTrip(t *testing.T) {
	for _, v := range []StrategyPreference{
		StrategyPreferenceUnspecified, StrategyPreferenceAuto, StrategyPreferenceVAD,
		StrategyPreferenceOverlap, StrategyPreferencePassthrough,
	} {
		if got := strategyPreferenceFromProto(v.toProto()); got != v {
			t.Errorf("StrategyPreference round-trip %d = %d", v, got)
		}
	}
	if got := StrategyPreference(99).toProto(); got != sttv1.StrategyPreference_STRATEGY_PREFERENCE_UNSPECIFIED {
		t.Errorf("out-of-range StrategyPreference = %v", got)
	}
}

func TestSummarizeLevelRoundTrip(t *testing.T) {
	for _, v := range []SummarizeLevel{
		SummarizeLevelUnspecified, SummarizeLevelLight, SummarizeLevelModerate, SummarizeLevelHeavy,
	} {
		if got := summarizeLevelFromProto(v.toProto()); got != v {
			t.Errorf("SummarizeLevel round-trip %d = %d", v, got)
		}
	}
	if got := SummarizeLevel(99).toProto(); got != summv1.SummarizeLevel_SUMMARIZE_LEVEL_UNSPECIFIED {
		t.Errorf("out-of-range SummarizeLevel = %v", got)
	}
}

func TestSpeakerCapabilityFromString(t *testing.T) {
	cases := map[string]SpeakerCapability{
		"available":     SpeakerCapabilityAvailable,
		"degraded":      SpeakerCapabilityDegraded,
		"unavailable":   SpeakerCapabilityUnavailable,
		"uninitialized": SpeakerCapabilityUninitialized,
		"":              SpeakerCapabilityUninitialized,
		"garbage":       SpeakerCapabilityUnspecified,
	}
	for in, want := range cases {
		if got := speakerCapabilityFromString(in); got != want {
			t.Errorf("speakerCapabilityFromString(%q) = %d, want %d", in, got, want)
		}
	}
}

// -----------------------------------------------------------------------------
// Struct round-trips (domain -> proto -> domain)
// -----------------------------------------------------------------------------

func TestStreamConfigRoundTrip(t *testing.T) {
	orig := StreamConfig{
		FlushIntervalMs:    100,
		MinDeltaBytes:      200,
		OverlapBytes:       300,
		PersistentMode:     true,
		WakeWordEnabled:    true,
		WakeWordThreshold:  0.42,
		SegmentSilenceMs:   400,
		StreamingMode:      StreamingModeAuto,
		StrategyPreference: StrategyPreferenceVAD,
		VadSilenceMs:       500,
		OverlapWindowMs:    600,
		OverlapCommitRuns:  7,
	}
	got := streamConfigFromProto(streamConfigToProto(orig))
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("StreamConfig round-trip mismatch:\n got=%+v\nwant=%+v", got, orig)
	}
}

func TestStreamConfigFromProtoNil(t *testing.T) {
	if got := streamConfigFromProto(nil); !reflect.DeepEqual(got, StreamConfig{}) {
		t.Errorf("streamConfigFromProto(nil) = %+v, want zero", got)
	}
}

func TestWakeWordTemplateRoundTrip(t *testing.T) {
	orig := &WakeWordTemplate{
		Label:     "hey vrooli",
		Threshold: 0.8,
		Samples: []WakeWordSample{
			{Audio: []byte{1, 2, 3}, Format: AudioFormatWAV, SampleRateHz: 16000},
			{Audio: []byte{4, 5}, Format: AudioFormatMP3, SampleRateHz: 8000},
		},
	}
	// NOTE: wakeWordTemplateToProto intentionally does not emit UpdatedAt
	// (the audio-tools server owns that timestamp), so the round-trip is over
	// Label/Threshold/Samples only.
	got := wakeWordTemplateFromProto(wakeWordTemplateToProto(orig))
	if got == nil {
		t.Fatal("round-trip returned nil")
	}
	if got.Label != orig.Label || got.Threshold != orig.Threshold {
		t.Errorf("scalar mismatch: got=%+v want=%+v", got, orig)
	}
	if !reflect.DeepEqual(got.Samples, orig.Samples) {
		t.Errorf("samples mismatch: got=%+v want=%+v", got.Samples, orig.Samples)
	}
	// The UpdatedAt direction (proto -> domain) is covered separately.
	upTmpl := wakeWordTemplateFromProto(&sttv1.WakeWordTemplate{
		Label:     "x",
		UpdatedAt: timestamppb.New(time.Unix(1700000000, 0).UTC()),
	})
	if upTmpl == nil || upTmpl.UpdatedAt.Unix() != 1700000000 {
		t.Errorf("UpdatedAt fromProto not mapped: %+v", upTmpl)
	}
}

func TestWakeWordTemplateNil(t *testing.T) {
	if got := wakeWordTemplateFromProto(nil); got != nil {
		t.Errorf("wakeWordTemplateFromProto(nil) = %+v, want nil", got)
	}
	if got := wakeWordTemplateToProto(nil); got != nil {
		t.Errorf("wakeWordTemplateToProto(nil) = %+v, want nil", got)
	}
}

func TestWakeWordConfigFromProto(t *testing.T) {
	if got := wakeWordConfigFromProto(nil); !reflect.DeepEqual(got, WakeWordConfig{}) {
		t.Errorf("wakeWordConfigFromProto(nil) = %+v, want zero", got)
	}
	p := &sttv1.WakeWordConfig{
		Configured: true,
		Template:   &sttv1.WakeWordTemplate{Label: "x", Threshold: 0.5},
	}
	got := wakeWordConfigFromProto(p)
	if !got.Configured {
		t.Error("Configured not propagated")
	}
	if got.Template == nil || got.Template.Label != "x" {
		t.Errorf("Template not propagated: %+v", got.Template)
	}
}

func TestSpeakerConfigRoundTrip(t *testing.T) {
	orig := SpeakerConfig{
		Enabled:                     true,
		ProfileIDs:                  []string{"a", "b"},
		Threshold:                   0.6,
		Mode:                        SpeakerModeFilter,
		RejectBehavior:              RejectBehaviorShowMuted,
		FallbackWithoutVerification: true,
		ExtractionEnabled:           true,
	}
	got := speakerConfigFromProto(speakerConfigToProto(orig))
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("SpeakerConfig round-trip mismatch:\n got=%+v\nwant=%+v", got, orig)
	}
}

func TestSpeakerConfigFromProtoNil(t *testing.T) {
	if got := speakerConfigFromProto(nil); !reflect.DeepEqual(got, SpeakerConfig{}) {
		t.Errorf("speakerConfigFromProto(nil) = %+v, want zero", got)
	}
}

func TestSpeakerProfileFromProto(t *testing.T) {
	if got := speakerProfileFromProto(nil); !reflect.DeepEqual(got, SpeakerProfile{}) {
		t.Errorf("speakerProfileFromProto(nil) = %+v, want zero", got)
	}
	created := time.Unix(1700000001, 0).UTC()
	updated := time.Unix(1700000002, 0).UTC()
	p := &sttv1.SpeakerProfile{
		Id:                     "id1",
		DisplayName:            "Alice",
		ModelName:              "ecapa",
		EmbeddingDim:           192,
		SampleRate:             16000,
		EnrollmentAudioSeconds: 12.5,
		Notes:                  "n",
		CreatedAt:              timestamppb.New(created),
		UpdatedAt:              timestamppb.New(updated),
	}
	got := speakerProfileFromProto(p)
	if got.ID != "id1" || got.DisplayName != "Alice" || got.ModelName != "ecapa" ||
		got.EmbeddingDim != 192 || got.SampleRate != 16000 || got.EnrollmentAudioSeconds != 12.5 || got.Notes != "n" {
		t.Errorf("scalar fields mismatch: %+v", got)
	}
	if !got.CreatedAt.Equal(created) || !got.UpdatedAt.Equal(updated) {
		t.Errorf("timestamps mismatch: %+v", got)
	}
}

func TestSpeakerStatusFromProto(t *testing.T) {
	if got := speakerStatusFromProto(nil); !reflect.DeepEqual(got, SpeakerStatus{}) {
		t.Errorf("speakerStatusFromProto(nil) = %+v, want zero", got)
	}
	checked := time.Unix(1700000003, 0).UTC()
	p := &sttv1.SpeakerStatus{
		Config:            &sttv1.SpeakerConfig{Enabled: true, Mode: sttv1.SpeakerMode_SPEAKER_MODE_FILTER},
		Capability:        "degraded",
		CapabilityLabel:   "Degraded",
		ResourceReady:     true,
		ProfileConfigured: true,
		ProfileExists:     true,
		ProfileCount:      2,
		Profiles: []*sttv1.SpeakerProfile{
			{Id: "p1"},
			{Id: "p2"},
		},
		Info: &sttv1.SpeakerResourceInfo{
			Backend:      "torch",
			Model:        "ecapa",
			Device:       "cpu",
			SampleRate:   16000,
			Version:      "1.0",
			EmbeddingDim: 192,
		},
		CheckedAt: timestamppb.New(checked),
	}
	got := speakerStatusFromProto(p)
	if got.Capability != SpeakerCapabilityDegraded {
		t.Errorf("Capability = %d, want degraded", got.Capability)
	}
	if got.CapabilityLabel != "Degraded" || !got.ResourceReady || !got.ProfileConfigured ||
		!got.ProfileExists || got.ProfileCount != 2 {
		t.Errorf("scalar fields mismatch: %+v", got)
	}
	if !got.Config.Enabled || got.Config.Mode != SpeakerModeFilter {
		t.Errorf("Config not propagated: %+v", got.Config)
	}
	if len(got.Profiles) != 2 || got.Profiles[0].ID != "p1" || got.Profiles[1].ID != "p2" {
		t.Errorf("Profiles not propagated: %+v", got.Profiles)
	}
	if got.Info == nil || got.Info.Backend != "torch" || got.Info.EmbeddingDim != 192 {
		t.Errorf("Info not propagated: %+v", got.Info)
	}
	if !got.CheckedAt.Equal(checked) {
		t.Errorf("CheckedAt mismatch: %v", got.CheckedAt)
	}
}

func TestSpeakerEnrollmentFromProto(t *testing.T) {
	if got := speakerEnrollmentFromProto(nil); !reflect.DeepEqual(got, SpeakerEnrollment{}) {
		t.Errorf("speakerEnrollmentFromProto(nil) = %+v, want zero", got)
	}
	created := time.Unix(1700000004, 0).UTC()
	p := &sttv1.SpeakerEnrollment{
		ProfileId:              "pid",
		DisplayName:            "Bob",
		EmbeddingDim:           256,
		SampleRate:             24000,
		EnrollmentAudioSeconds: 9.0,
		ModelName:              "m",
		CreatedAt:              timestamppb.New(created),
	}
	got := speakerEnrollmentFromProto(p)
	if got.ProfileID != "pid" || got.DisplayName != "Bob" || got.EmbeddingDim != 256 ||
		got.SampleRate != 24000 || got.EnrollmentAudioSeconds != 9.0 || got.ModelName != "m" {
		t.Errorf("scalar fields mismatch: %+v", got)
	}
	if !got.CreatedAt.Equal(created) {
		t.Errorf("CreatedAt mismatch: %v", got.CreatedAt)
	}
}

func TestSummarizeConfigRoundTrip(t *testing.T) {
	orig := SummarizeConfig{
		Enabled:        true,
		CharThreshold:  1000,
		Level:          SummarizeLevelModerate,
		Model:          "gemma",
		TimeoutSeconds: 30,
	}
	got := summarizeConfigFromProto(summarizeConfigToProto(orig))
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("SummarizeConfig round-trip mismatch:\n got=%+v\nwant=%+v", got, orig)
	}
}

func TestSummarizeConfigFromProtoNil(t *testing.T) {
	if got := summarizeConfigFromProto(nil); !reflect.DeepEqual(got, SummarizeConfig{}) {
		t.Errorf("summarizeConfigFromProto(nil) = %+v, want zero", got)
	}
}

func TestSummarizeModelFromProto(t *testing.T) {
	if got := summarizeModelFromProto(nil); !reflect.DeepEqual(got, SummarizeModel{}) {
		t.Errorf("summarizeModelFromProto(nil) = %+v, want zero", got)
	}
	p := &summv1.SummarizeModel{
		Id:              "id",
		DisplayName:     "Gemma",
		Installed:       true,
		Recommended:     true,
		DefaultEligible: true,
		Reasoning:       true,
		StatusLabel:     "ready",
		PullCommand:     "ollama pull gemma",
		SizeBytes:       12345,
		ParameterSize:   "2b",
		SourceUrl:       "http://x",
		Notes:           "note",
	}
	got := summarizeModelFromProto(p)
	want := SummarizeModel{
		ID:              "id",
		DisplayName:     "Gemma",
		Installed:       true,
		Recommended:     true,
		DefaultEligible: true,
		Reasoning:       true,
		StatusLabel:     "ready",
		PullCommand:     "ollama pull gemma",
		SizeBytes:       12345,
		ParameterSize:   "2b",
		SourceURL:       "http://x",
		Notes:           "note",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("summarizeModelFromProto mismatch:\n got=%+v\nwant=%+v", got, want)
	}
}

func TestTTSConfigRoundTrip(t *testing.T) {
	orig := TTSConfig{
		AutoEnabled:           true,
		DefaultVoice:          "af_heart",
		DefaultSpeed:          1.25,
		DefaultResponseFormat: ResponseFormatOPUS,
	}
	got := ttsConfigFromProto(ttsConfigToProto(orig))
	if !reflect.DeepEqual(got, orig) {
		t.Errorf("TTSConfig round-trip mismatch:\n got=%+v\nwant=%+v", got, orig)
	}
}

func TestTTSConfigFromProtoNil(t *testing.T) {
	if got := ttsConfigFromProto(nil); !reflect.DeepEqual(got, TTSConfig{}) {
		t.Errorf("ttsConfigFromProto(nil) = %+v, want zero", got)
	}
}

// Guard against an unused-import on ttsv1 if the file evolves.
var _ = ttsv1.Config{}
