package protomap_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	summarizev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"

	"audio-tools/internal/protomap"
)

func TestProviderTier_Roundtrip(t *testing.T) {
	cases := []struct {
		s string
		p commonv1.ProviderTier
	}{
		{"byok", commonv1.ProviderTier_PROVIDER_TIER_BYOK},
		{"vrooli", commonv1.ProviderTier_PROVIDER_TIER_VROOLI},
		{"local", commonv1.ProviderTier_PROVIDER_TIER_LOCAL},
	}
	for _, c := range cases {
		if got := protomap.ProviderTierToProto(c.s); got != c.p {
			t.Errorf("ToProto(%q) = %v, want %v", c.s, got, c.p)
		}
		if got := protomap.ProviderTierFromProto(c.p); got != c.s {
			t.Errorf("FromProto(%v) = %q, want %q", c.p, got, c.s)
		}
	}
	if got := protomap.ProviderTierToProto("nope"); got != commonv1.ProviderTier_PROVIDER_TIER_UNSPECIFIED {
		t.Errorf("ToProto(unknown) = %v, want UNSPECIFIED", got)
	}
	if got := protomap.ProviderTierFromProto(commonv1.ProviderTier_PROVIDER_TIER_UNSPECIFIED); got != "" {
		t.Errorf("FromProto(UNSPECIFIED) = %q, want empty", got)
	}
}

func TestAudioFormat_Roundtrip(t *testing.T) {
	pairs := map[string]commonv1.AudioFormat{
		"wav":  commonv1.AudioFormat_AUDIO_FORMAT_WAV,
		"mp3":  commonv1.AudioFormat_AUDIO_FORMAT_MP3,
		"flac": commonv1.AudioFormat_AUDIO_FORMAT_FLAC,
		"ogg":  commonv1.AudioFormat_AUDIO_FORMAT_OGG,
		"webm": commonv1.AudioFormat_AUDIO_FORMAT_WEBM,
		"opus": commonv1.AudioFormat_AUDIO_FORMAT_OPUS,
	}
	for s, p := range pairs {
		if got := protomap.AudioFormatToProto(s); got != p {
			t.Errorf("ToProto(%q) = %v", s, got)
		}
		if got := protomap.AudioFormatFromProto(p); got != s {
			t.Errorf("FromProto(%v) = %q", p, got)
		}
	}
	if got := protomap.AudioFormatToProto("xyz"); got != commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED {
		t.Errorf("unknown → %v", got)
	}
	if got := protomap.AudioFormatFromProto(commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED); got != "" {
		t.Errorf("UNSPECIFIED → %q", got)
	}
}

func TestResponseFormat_Roundtrip(t *testing.T) {
	pairs := map[string]commonv1.ResponseFormat{
		"mp3":  commonv1.ResponseFormat_RESPONSE_FORMAT_MP3,
		"wav":  commonv1.ResponseFormat_RESPONSE_FORMAT_WAV,
		"opus": commonv1.ResponseFormat_RESPONSE_FORMAT_OPUS,
		"flac": commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC,
	}
	for s, p := range pairs {
		if got := protomap.ResponseFormatToProto(s); got != p {
			t.Errorf("ToProto(%q) = %v", s, got)
		}
		if got := protomap.ResponseFormatFromProto(p); got != s {
			t.Errorf("FromProto(%v) = %q", p, got)
		}
	}
	if got := protomap.ResponseFormatToProto("zzz"); got != commonv1.ResponseFormat_RESPONSE_FORMAT_UNSPECIFIED {
		t.Errorf("unknown → %v", got)
	}
}

func TestSpeakerMode_Roundtrip(t *testing.T) {
	pairs := map[string]sttv1.SpeakerMode{
		"off":      sttv1.SpeakerMode_SPEAKER_MODE_OFF,
		"filter":   sttv1.SpeakerMode_SPEAKER_MODE_FILTER,
		"advisory": sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY,
	}
	for s, p := range pairs {
		if got := protomap.SpeakerModeToProto(s); got != p {
			t.Errorf("ToProto(%q) = %v", s, got)
		}
		if got := protomap.SpeakerModeFromProto(p); got != s {
			t.Errorf("FromProto(%v) = %q", p, got)
		}
	}
}

func TestRejectBehavior_Roundtrip(t *testing.T) {
	if got := protomap.RejectBehaviorToProto("drop"); got != sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP {
		t.Error("drop")
	}
	if got := protomap.RejectBehaviorToProto("show-muted"); got != sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED {
		t.Error("show-muted")
	}
	if got := protomap.RejectBehaviorToProto("show_muted"); got != sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED {
		t.Error("show_muted alias")
	}
	if got := protomap.RejectBehaviorFromProto(sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP); got != "drop" {
		t.Error("from drop")
	}
	if got := protomap.RejectBehaviorFromProto(sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED); got != "show-muted" {
		t.Error("from show-muted")
	}
	if got := protomap.RejectBehaviorFromProto(sttv1.RejectBehavior_REJECT_BEHAVIOR_UNSPECIFIED); got != "" {
		t.Error("from unspecified")
	}
}

func TestStreamingMode_Roundtrip(t *testing.T) {
	if protomap.StreamingModeToProto("auto") != sttv1.StreamingMode_STREAMING_MODE_AUTO {
		t.Error("auto")
	}
	if protomap.StreamingModeToProto("off") != sttv1.StreamingMode_STREAMING_MODE_OFF {
		t.Error("off")
	}
	if protomap.StreamingModeToProto("?") != sttv1.StreamingMode_STREAMING_MODE_UNSPECIFIED {
		t.Error("unknown")
	}
	if protomap.StreamingModeFromProto(sttv1.StreamingMode_STREAMING_MODE_AUTO) != "auto" {
		t.Error("from auto")
	}
	if protomap.StreamingModeFromProto(sttv1.StreamingMode_STREAMING_MODE_OFF) != "off" {
		t.Error("from off")
	}
	if protomap.StreamingModeFromProto(sttv1.StreamingMode_STREAMING_MODE_UNSPECIFIED) != "" {
		t.Error("from unspecified")
	}
}

func TestStrategyPreference_Roundtrip(t *testing.T) {
	pairs := map[string]sttv1.StrategyPreference{
		"auto":        sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO,
		"vad":         sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD,
		"overlap":     sttv1.StrategyPreference_STRATEGY_PREFERENCE_OVERLAP,
		"passthrough": sttv1.StrategyPreference_STRATEGY_PREFERENCE_PASSTHROUGH,
	}
	for s, p := range pairs {
		if protomap.StrategyPreferenceToProto(s) != p {
			t.Errorf("To(%q)", s)
		}
		if protomap.StrategyPreferenceFromProto(p) != s {
			t.Errorf("From(%v)", p)
		}
	}
}

func TestSummarizeLevel_Roundtrip(t *testing.T) {
	pairs := map[string]summarizev1.SummarizeLevel{
		"light":    summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT,
		"moderate": summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE,
		"heavy":    summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY,
	}
	for s, p := range pairs {
		if protomap.SummarizeLevelToProto(s) != p {
			t.Errorf("To(%q)", s)
		}
		if protomap.SummarizeLevelFromProto(p) != s {
			t.Errorf("From(%v)", p)
		}
	}
}

func TestSessionTransport_Roundtrip(t *testing.T) {
	if protomap.SessionTransportToProto("browser-voice") != sessionv1.SessionTransport_SESSION_TRANSPORT_BROWSER_VOICE {
		t.Error("browser-voice")
	}
	if protomap.SessionTransportToProto("browser_voice") != sessionv1.SessionTransport_SESSION_TRANSPORT_BROWSER_VOICE {
		t.Error("browser_voice alias")
	}
	if protomap.SessionTransportToProto("fake") != sessionv1.SessionTransport_SESSION_TRANSPORT_FAKE {
		t.Error("fake")
	}
	if protomap.SessionTransportFromProto(sessionv1.SessionTransport_SESSION_TRANSPORT_BROWSER_VOICE) != "browser-voice" {
		t.Error("from browser-voice")
	}
	if protomap.SessionTransportFromProto(sessionv1.SessionTransport_SESSION_TRANSPORT_FAKE) != "fake" {
		t.Error("from fake")
	}
}

func TestVadState(t *testing.T) {
	if protomap.VadStateToProto("speech_start") != sessionv1.VadState_VAD_STATE_SPEECH_START {
		t.Error("speech_start")
	}
	if protomap.VadStateToProto("speech-start") != sessionv1.VadState_VAD_STATE_SPEECH_START {
		t.Error("speech-start")
	}
	if protomap.VadStateToProto("speech-end") != sessionv1.VadState_VAD_STATE_SPEECH_END {
		t.Error("speech-end")
	}
	if protomap.VadStateToProto("?") != sessionv1.VadState_VAD_STATE_UNSPECIFIED {
		t.Error("unknown")
	}
}

func TestBargeInReason(t *testing.T) {
	if protomap.BargeInReasonToProto("vad") != sessionv1.BargeInReason_BARGE_IN_REASON_VAD {
		t.Error("vad")
	}
	if protomap.BargeInReasonToProto("explicit") != sessionv1.BargeInReason_BARGE_IN_REASON_EXPLICIT {
		t.Error("explicit")
	}
	if protomap.BargeInReasonToProto("?") != sessionv1.BargeInReason_BARGE_IN_REASON_UNSPECIFIED {
		t.Error("unknown")
	}
}

func TestTimeToFromProto(t *testing.T) {
	if got := protomap.TimeToProto(time.Time{}); got != nil {
		t.Errorf("zero → %v, want nil", got)
	}
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	pb := protomap.TimeToProto(now)
	if pb == nil {
		t.Fatal("expected non-nil")
	}
	if !pb.AsTime().Equal(now) {
		t.Errorf("roundtrip: %v vs %v", pb.AsTime(), now)
	}
	if got := protomap.TimeFromProto(nil); !got.IsZero() {
		t.Errorf("nil → %v, want zero", got)
	}
	if got := protomap.TimeFromProto(timestamppb.New(now)); !got.Equal(now) {
		t.Errorf("non-nil: %v vs %v", got, now)
	}
}

func TestMaskHas(t *testing.T) {
	if protomap.MaskHas(nil, "x") {
		t.Error("nil mask")
	}
	m := &fieldmaskpb.FieldMask{Paths: []string{"a", "b.c"}}
	if !protomap.MaskHas(m, "a") {
		t.Error("a")
	}
	if !protomap.MaskHas(m, "b.c") {
		t.Error("b.c")
	}
	if protomap.MaskHas(m, "missing") {
		t.Error("missing should be false")
	}
}

func TestMaskPathsOutsideAllowed(t *testing.T) {
	if got := protomap.MaskPathsOutsideAllowed(nil, nil); got != nil {
		t.Errorf("nil → %v", got)
	}
	allowed := map[string]struct{}{"a": {}, "b": {}}
	m := &fieldmaskpb.FieldMask{Paths: []string{"a", "c", "b", "d"}}
	got := protomap.MaskPathsOutsideAllowed(m, allowed)
	if len(got) != 2 || got[0] != "c" || got[1] != "d" {
		t.Errorf("got %v, want [c d]", got)
	}
	mAllOK := &fieldmaskpb.FieldMask{Paths: []string{"a", "b"}}
	if got := protomap.MaskPathsOutsideAllowed(mAllOK, allowed); got != nil {
		t.Errorf("all allowed → %v", got)
	}
}
