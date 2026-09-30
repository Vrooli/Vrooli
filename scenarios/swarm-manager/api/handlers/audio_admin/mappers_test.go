package audio_admin

import (
	"testing"
	"time"

	"swarm-manager/internal/audioports"

	audiocommonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/audio_common"
)

// -----------------------------------------------------------------------------
// StreamConfig round-trip
// -----------------------------------------------------------------------------

func TestStreamConfigRoundTrip(t *testing.T) {
	in := audioports.StreamConfig{
		FlushIntervalMs:    250,
		MinDeltaBytes:      1024,
		OverlapBytes:       64,
		PersistentMode:     true,
		WakeWordEnabled:    true,
		WakeWordThreshold:  0.75,
		SegmentSilenceMs:   400,
		StreamingMode:      audioports.StreamingModeAuto,
		StrategyPreference: audioports.StrategyPreferenceVAD,
		VadSilenceMs:       300,
		OverlapWindowMs:    150,
		OverlapCommitRuns:  3,
	}
	got := streamConfigFromProto(streamConfigToProto(in))
	if got != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestStreamConfigFromProto_Nil(t *testing.T) {
	if got := streamConfigFromProto(nil); got != (audioports.StreamConfig{}) {
		t.Errorf("nil proto should yield zero value, got %+v", got)
	}
}

// -----------------------------------------------------------------------------
// SpeakerConfig round-trip
// -----------------------------------------------------------------------------

func TestSpeakerConfigRoundTrip(t *testing.T) {
	in := audioports.SpeakerConfig{
		Enabled:                     true,
		ProfileIDs:                  []string{"p1", "p2"},
		Threshold:                   0.8,
		Mode:                        audioports.SpeakerModeFilter,
		RejectBehavior:              audioports.RejectBehaviorShowMuted,
		FallbackWithoutVerification: true,
		ExtractionEnabled:           true,
	}
	got := speakerConfigFromProto(speakerConfigToProto(in))
	if got.Enabled != in.Enabled || got.Threshold != in.Threshold || got.Mode != in.Mode ||
		got.RejectBehavior != in.RejectBehavior || got.FallbackWithoutVerification != in.FallbackWithoutVerification ||
		got.ExtractionEnabled != in.ExtractionEnabled {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
	if len(got.ProfileIDs) != 2 || got.ProfileIDs[0] != "p1" || got.ProfileIDs[1] != "p2" {
		t.Errorf("ProfileIDs mismatch: %+v", got.ProfileIDs)
	}
}

func TestSpeakerConfigFromProto_Nil(t *testing.T) {
	got := speakerConfigFromProto(nil)
	if got.Enabled || got.ProfileIDs != nil {
		t.Errorf("nil proto should yield zero value, got %+v", got)
	}
}

// -----------------------------------------------------------------------------
// TTSConfig round-trip
// -----------------------------------------------------------------------------

func TestTTSConfigRoundTrip(t *testing.T) {
	in := audioports.TTSConfig{
		AutoEnabled:           true,
		DefaultVoice:          "af_heart",
		DefaultSpeed:          1.25,
		DefaultResponseFormat: audioports.ResponseFormatOPUS,
	}
	got := ttsConfigFromProto(ttsConfigToProto(in))
	if got != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestTTSConfigFromProto_Nil(t *testing.T) {
	if got := ttsConfigFromProto(nil); got != (audioports.TTSConfig{}) {
		t.Errorf("nil proto should yield zero value, got %+v", got)
	}
}

// -----------------------------------------------------------------------------
// SummarizeConfig round-trip
// -----------------------------------------------------------------------------

func TestSummarizeConfigRoundTrip(t *testing.T) {
	in := audioports.SummarizeConfig{
		Enabled:        true,
		CharThreshold:  500,
		Level:          audioports.SummarizeLevelHeavy,
		Model:          "qwen2.5",
		TimeoutSeconds: 30,
	}
	got := summarizeConfigFromProto(summarizeConfigToProto(in))
	if got != in {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, in)
	}
}

func TestSummarizeConfigFromProto_Nil(t *testing.T) {
	if got := summarizeConfigFromProto(nil); got != (audioports.SummarizeConfig{}) {
		t.Errorf("nil proto should yield zero value, got %+v", got)
	}
}

// -----------------------------------------------------------------------------
// WakeWordTemplate round-trip
// -----------------------------------------------------------------------------

func TestWakeWordTemplateRoundTrip(t *testing.T) {
	updated := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	in := &audioports.WakeWordTemplate{
		Label:     "computer",
		Threshold: 0.6,
		UpdatedAt: updated,
		Samples: []audioports.WakeWordSample{
			{Audio: []byte("aaa"), Format: audioports.AudioFormatWAV, SampleRateHz: 16000},
			{Audio: []byte("bbb"), Format: audioports.AudioFormatMP3, SampleRateHz: 22050},
		},
	}
	got := wakeWordTemplateFromProto(wakeWordTemplateToProto(in))
	if got.Label != in.Label || got.Threshold != in.Threshold {
		t.Errorf("scalar mismatch: %+v", got)
	}
	if !got.UpdatedAt.Equal(updated) {
		t.Errorf("UpdatedAt mismatch: got %v want %v", got.UpdatedAt, updated)
	}
	if len(got.Samples) != 2 {
		t.Fatalf("got %d samples, want 2", len(got.Samples))
	}
	if string(got.Samples[0].Audio) != "aaa" || got.Samples[0].Format != audioports.AudioFormatWAV || got.Samples[0].SampleRateHz != 16000 {
		t.Errorf("sample[0] mismatch: %+v", got.Samples[0])
	}
	if got.Samples[1].Format != audioports.AudioFormatMP3 || got.Samples[1].SampleRateHz != 22050 {
		t.Errorf("sample[1] mismatch: %+v", got.Samples[1])
	}
}

func TestWakeWordTemplateToProto_Nil(t *testing.T) {
	if got := wakeWordTemplateToProto(nil); got != nil {
		t.Errorf("nil template should yield nil proto, got %+v", got)
	}
}

func TestWakeWordTemplateFromProto_Nil(t *testing.T) {
	got := wakeWordTemplateFromProto(nil)
	if got.Label != "" || got.Samples != nil {
		t.Errorf("nil proto should yield zero value, got %+v", got)
	}
}

func TestWakeWordTemplateToProto_ZeroTime(t *testing.T) {
	got := wakeWordTemplateToProto(&audioports.WakeWordTemplate{Label: "x"})
	if got.UpdatedAt != nil {
		t.Errorf("zero UpdatedAt should not emit a timestamp, got %+v", got.UpdatedAt)
	}
}

func TestWakeWordConfigToProto(t *testing.T) {
	c := audioports.WakeWordConfig{Configured: true, Template: &audioports.WakeWordTemplate{Label: "hi"}}
	got := wakeWordConfigToProto(c)
	if !got.Configured || got.Template == nil || got.Template.Label != "hi" {
		t.Errorf("config not mapped: %+v", got)
	}

	// nil template stays nil
	got2 := wakeWordConfigToProto(audioports.WakeWordConfig{Configured: false})
	if got2.Configured || got2.Template != nil {
		t.Errorf("expected unconfigured with nil template, got %+v", got2)
	}
}

// -----------------------------------------------------------------------------
// SpeakerProfile / SpeakerStatus / SpeakerEnrollment one-way mappers
// -----------------------------------------------------------------------------

func TestSpeakerProfileToProto(t *testing.T) {
	created := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
	p := audioports.SpeakerProfile{
		ID:                     "p1",
		DisplayName:            "Alice",
		CreatedAt:              created,
		UpdatedAt:              updated,
		ModelName:              "ecapa",
		EmbeddingDim:           192,
		SampleRate:             16000,
		EnrollmentAudioSeconds: 12.5,
		Notes:                  "n",
	}
	got := speakerProfileToProto(p)
	if got.Id != "p1" || got.DisplayName != "Alice" || got.ModelName != "ecapa" ||
		got.EmbeddingDim != 192 || got.SampleRate != 16000 || got.EnrollmentAudioSeconds != 12.5 || got.Notes != "n" {
		t.Errorf("profile not mapped: %+v", got)
	}
	if got.CreatedAt == nil || !got.CreatedAt.AsTime().Equal(created) {
		t.Errorf("CreatedAt mismatch: %v", got.CreatedAt)
	}
	if got.UpdatedAt == nil || !got.UpdatedAt.AsTime().Equal(updated) {
		t.Errorf("UpdatedAt mismatch: %v", got.UpdatedAt)
	}
}

func TestSpeakerProfileToProto_ZeroTimes(t *testing.T) {
	got := speakerProfileToProto(audioports.SpeakerProfile{ID: "p1"})
	if got.CreatedAt != nil || got.UpdatedAt != nil {
		t.Errorf("zero times should not emit timestamps: %+v / %+v", got.CreatedAt, got.UpdatedAt)
	}
}

func TestSpeakerStatusToProto(t *testing.T) {
	checked := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	s := audioports.SpeakerStatus{
		Config:            audioports.SpeakerConfig{Enabled: true},
		Capability:        audioports.SpeakerCapabilityDegraded,
		CapabilityLabel:   "degraded",
		ResourceReady:     true,
		ProfileConfigured: true,
		ProfileExists:     true,
		ProfileCount:      2,
		Profiles:          []audioports.SpeakerProfile{{ID: "p1"}, {ID: "p2"}},
		Info: &audioports.SpeakerResourceInfo{
			Backend: "torch", Model: "ecapa", Device: "cuda", SampleRate: 16000, Version: "1.0", EmbeddingDim: 192,
		},
		CheckedAt: checked,
	}
	got := speakerStatusToProto(s)
	if !got.ResourceReady || got.ProfileCount != 2 || got.CapabilityLabel != "degraded" {
		t.Errorf("status scalars not mapped: %+v", got)
	}
	if got.Capability != audiocommonv1.SpeakerCapability_SPEAKER_CAPABILITY_DEGRADED {
		t.Errorf("capability enum not mapped: %v", got.Capability)
	}
	if len(got.Profiles) != 2 || got.Profiles[0].Id != "p1" {
		t.Errorf("profiles not mapped: %+v", got.Profiles)
	}
	if got.Info == nil || got.Info.Backend != "torch" || got.Info.EmbeddingDim != 192 {
		t.Errorf("info not mapped: %+v", got.Info)
	}
	if got.CheckedAt == nil || !got.CheckedAt.AsTime().Equal(checked) {
		t.Errorf("CheckedAt mismatch: %v", got.CheckedAt)
	}
}

func TestSpeakerStatusToProto_NilInfoZeroTime(t *testing.T) {
	got := speakerStatusToProto(audioports.SpeakerStatus{})
	if got.Info != nil {
		t.Errorf("nil Info should stay nil, got %+v", got.Info)
	}
	if got.CheckedAt != nil {
		t.Errorf("zero CheckedAt should not emit a timestamp, got %+v", got.CheckedAt)
	}
}

func TestSpeakerEnrollmentToProto(t *testing.T) {
	created := time.Date(2026, 4, 4, 0, 0, 0, 0, time.UTC)
	e := audioports.SpeakerEnrollment{
		ProfileID:              "p9",
		DisplayName:            "Carol",
		EmbeddingDim:           192,
		SampleRate:             16000,
		EnrollmentAudioSeconds: 8.0,
		ModelName:              "ecapa",
		CreatedAt:              created,
	}
	got := speakerEnrollmentToProto(e)
	if got.ProfileId != "p9" || got.DisplayName != "Carol" || got.EmbeddingDim != 192 ||
		got.SampleRate != 16000 || got.EnrollmentAudioSeconds != 8.0 || got.ModelName != "ecapa" {
		t.Errorf("enrollment not mapped: %+v", got)
	}
	if got.CreatedAt == nil || !got.CreatedAt.AsTime().Equal(created) {
		t.Errorf("CreatedAt mismatch: %v", got.CreatedAt)
	}
}

func TestSpeakerEnrollmentToProto_ZeroTime(t *testing.T) {
	got := speakerEnrollmentToProto(audioports.SpeakerEnrollment{ProfileID: "p1"})
	if got.CreatedAt != nil {
		t.Errorf("zero CreatedAt should not emit a timestamp, got %+v", got.CreatedAt)
	}
}

// -----------------------------------------------------------------------------
// SummarizeModel one-way mapper
// -----------------------------------------------------------------------------

func TestSummarizeModelToProto(t *testing.T) {
	m := audioports.SummarizeModel{
		ID:              "m1",
		DisplayName:     "Model One",
		Installed:       true,
		Recommended:     true,
		DefaultEligible: true,
		Reasoning:       true,
		StatusLabel:     "ready",
		PullCommand:     "ollama pull m1",
		SizeBytes:       123456789,
		ParameterSize:   "7B",
		SourceURL:       "https://example.com",
		Notes:           "notes",
	}
	got := summarizeModelToProto(m)
	if got.Id != "m1" || got.DisplayName != "Model One" || !got.Installed || !got.Recommended ||
		!got.DefaultEligible || !got.Reasoning || got.StatusLabel != "ready" || got.PullCommand != "ollama pull m1" ||
		got.SizeBytes != 123456789 || got.ParameterSize != "7B" || got.SourceUrl != "https://example.com" || got.Notes != "notes" {
		t.Errorf("model not mapped: %+v", got)
	}
}
