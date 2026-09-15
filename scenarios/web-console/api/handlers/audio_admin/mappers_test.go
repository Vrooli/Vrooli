package audio_admin

import (
	"reflect"
	"testing"
	"time"

	"web-console/internal/audioports"

	audioadminv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_admin"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMapperRoundTripsAndOptionalFields(t *testing.T) {
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	stream := streamConfigToProto(audioports.StreamConfig{
		FlushIntervalMs: 10, MinDeltaBytes: 11, OverlapBytes: 12, PersistentMode: true,
		WakeWordEnabled: true, WakeWordThreshold: .7, SegmentSilenceMs: 13,
		StreamingMode: audioports.StreamingModeAuto, StrategyPreference: audioports.StrategyPreferencePassthrough,
		VadSilenceMs: 14, OverlapWindowMs: 15, OverlapCommitRuns: 16,
	})
	if got := streamConfigFromProto(stream); got.FlushIntervalMs != 10 || !got.PersistentMode || got.StrategyPreference != audioports.StrategyPreferencePassthrough {
		t.Fatalf("stream mapping lost fields: %+v", got)
	}
	if got := streamConfigFromProto(nil); got != (audioports.StreamConfig{}) {
		t.Fatalf("nil stream: %+v", got)
	}

	wake := wakeWordTemplateFromProto(&audioadminv1.WakeWordTemplate{
		Label: "hello", Threshold: .8, UpdatedAt: timestamppb.New(now),
		Samples: []*audioadminv1.WakeWordSample{{Audio: []byte{1}, Format: 2, SampleRateHz: 16000}},
	})
	if wake.Label != "hello" || len(wake.Samples) != 1 || wake.UpdatedAt.IsZero() {
		t.Fatalf("wake mapping: %+v", wake)
	}
	if !reflect.DeepEqual(wakeWordTemplateFromProto(nil), audioports.WakeWordTemplate{}) || wakeWordTemplateToProto(nil) != nil {
		t.Fatal("nil wake template should remain empty")
	}
	if got := wakeWordTemplateToProto(&audioports.WakeWordTemplate{Label: "hello", UpdatedAt: now, Samples: wake.Samples}); got.UpdatedAt == nil || len(got.Samples) != 1 {
		t.Fatalf("wake reverse mapping: %+v", got)
	}
	if got := wakeWordConfigToProto(audioports.WakeWordConfig{Configured: true}); !got.Configured || got.Template != nil {
		t.Fatalf("wake config: %+v", got)
	}

	speaker := audioports.SpeakerConfig{Enabled: true, ProfileIDs: []string{"p1"}, Threshold: .5, Mode: audioports.SpeakerModeFilter, RejectBehavior: audioports.RejectBehaviorDrop, FallbackWithoutVerification: true, ExtractionEnabled: true}
	if got := speakerConfigFromProto(speakerConfigToProto(speaker)); got.Enabled != speaker.Enabled || got.ProfileIDs[0] != "p1" || got.Mode != speaker.Mode {
		t.Fatalf("speaker mapping: %+v", got)
	}
	if !reflect.DeepEqual(speakerConfigFromProto(nil), audioports.SpeakerConfig{}) {
		t.Fatal("nil speaker should remain empty")
	}
	profile := audioports.SpeakerProfile{ID: "p1", DisplayName: "One", CreatedAt: now, UpdatedAt: now}
	status := speakerStatusToProto(audioports.SpeakerStatus{Config: speaker, Profiles: []audioports.SpeakerProfile{profile}, Info: &audioports.SpeakerResourceInfo{Backend: "x"}, CheckedAt: now})
	if len(status.Profiles) != 1 || status.Info == nil || status.CheckedAt == nil {
		t.Fatalf("speaker status: %+v", status)
	}
	if speakerEnrollmentToProto(audioports.SpeakerEnrollment{ProfileID: "p1", CreatedAt: now}).CreatedAt == nil {
		t.Fatal("enrollment timestamp missing")
	}

	if got := ttsConfigFromProto(ttsConfigToProto(audioports.TTSConfig{AutoEnabled: true, DefaultVoice: "v", DefaultSpeed: 1.2, DefaultResponseFormat: audioports.ResponseFormatWAV})); !got.AutoEnabled || got.DefaultVoice != "v" || got.DefaultResponseFormat != audioports.ResponseFormatWAV {
		t.Fatalf("tts mapping: %+v", got)
	}
	if !reflect.DeepEqual(ttsConfigFromProto(nil), audioports.TTSConfig{}) || !reflect.DeepEqual(summarizeConfigFromProto(nil), audioports.SummarizeConfig{}) {
		t.Fatal("nil config mapping should remain empty")
	}
	summary := audioports.SummarizeConfig{Enabled: true, CharThreshold: 100, Level: audioports.SummarizeLevelHeavy, Model: "m", TimeoutSeconds: 9}
	if got := summarizeConfigFromProto(summarizeConfigToProto(summary)); got != summary {
		t.Fatalf("summary mapping: %+v", got)
	}
	model := summarizeModelToProto(audioports.SummarizeModel{ID: "m", DisplayName: "Model", Installed: true, Recommended: true, DefaultEligible: true, SizeBytes: 42})
	if model.Id != "m" || !model.Installed || model.SizeBytes != 42 {
		t.Fatalf("model mapping: %+v", model)
	}
}
