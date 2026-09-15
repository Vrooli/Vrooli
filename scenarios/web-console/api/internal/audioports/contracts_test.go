package audioports

import (
	"testing"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
)

func TestEnumAndConfigConversions(t *testing.T) {
	for _, v := range []AudioFormat{AudioFormatUnspecified, AudioFormatWAV, AudioFormatMP3, AudioFormatAAC, -1, 99} {
		_ = v.toProto()
	}
	for _, v := range []ResponseFormat{ResponseFormatUnspecified, ResponseFormatMP3, ResponseFormatFLAC, -1, 99} {
		_ = v.toProto()
	}
	for _, v := range []SpeakerMode{SpeakerModeUnspecified, SpeakerModeOff, SpeakerModeAdvisory, -1, 99} {
		_ = v.toProto()
	}
	for _, v := range []RejectBehavior{RejectBehaviorUnspecified, RejectBehaviorDrop, RejectBehaviorShowMuted, -1, 99} {
		_ = v.toProto()
	}
	for _, v := range []StreamingMode{StreamingModeUnspecified, StreamingModeAuto, StreamingModeOff, -1, 99} {
		_ = v.toProto()
	}
	for _, v := range []StrategyPreference{StrategyPreferenceUnspecified, StrategyPreferenceAuto, StrategyPreferencePassthrough, -1, 99} {
		_ = v.toProto()
	}
	for _, v := range []SummarizeLevel{SummarizeLevelUnspecified, SummarizeLevelLight, SummarizeLevelHeavy, -1, 99} {
		_ = v.toProto()
	}
	for _, s := range []string{"available", "degraded", "unavailable", "uninitialized", "", "other"} {
		_ = speakerCapabilityFromString(s)
	}
	_ = responseFormatFromProto(commonv1.ResponseFormat_RESPONSE_FORMAT_MP3)
	_ = providerTierFromProto(commonv1.ProviderTier_PROVIDER_TIER_LOCAL)
	_ = speakerModeFromProto(sttv1.SpeakerMode_SPEAKER_MODE_OFF)
	_ = rejectBehaviorFromProto(sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP)
	_ = streamingModeFromProto(sttv1.StreamingMode_STREAMING_MODE_AUTO)
	_ = strategyPreferenceFromProto(sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO)
	_ = summarizeLevelFromProto(summv1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT)
	_ = streamConfigFromProto(nil)
	_ = streamConfigToProto(StreamConfig{})
	_ = wakeWordTemplateFromProto(nil)
	_ = wakeWordTemplateToProto(&WakeWordTemplate{})
	_ = wakeWordConfigFromProto(nil)
	_ = speakerConfigFromProto(nil)
	_ = speakerConfigToProto(SpeakerConfig{})
	_ = speakerProfileFromProto(nil)
	_ = speakerStatusFromProto(nil)
	_ = speakerEnrollmentFromProto(nil)
	_ = summarizeConfigFromProto(nil)
	_ = summarizeModelFromProto(nil)
	_ = ttsConfigFromProto(nil)
	_ = ttsConfigToProto(TTSConfig{})
	_ = summarizeConfigToProto(SummarizeConfig{})
}
