// Package protomap converts between audio-tools domain string values
// (used throughout internal/ai/*chain, internal/stt, internal/tts, etc.)
// and the typed proto enums + well-known types now used on the API
// surface.
//
// Two-way maps are intentionally explicit per enum so unknown inputs
// translate to the UNSPECIFIED zero value rather than panicking — the
// API layer rejects UNSPECIFIED via protovalidate or explicit checks.
package protomap

import (
	"time"

	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/session"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	summarizev1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
)

// -----------------------------------------------------------------------------
// ProviderTier
// -----------------------------------------------------------------------------

func ProviderTierToProto(s string) commonv1.ProviderTier {
	switch s {
	case "byok":
		return commonv1.ProviderTier_PROVIDER_TIER_BYOK
	case "vrooli":
		return commonv1.ProviderTier_PROVIDER_TIER_VROOLI
	case "local":
		return commonv1.ProviderTier_PROVIDER_TIER_LOCAL
	default:
		return commonv1.ProviderTier_PROVIDER_TIER_UNSPECIFIED
	}
}

func ProviderTierFromProto(t commonv1.ProviderTier) string {
	switch t {
	case commonv1.ProviderTier_PROVIDER_TIER_BYOK:
		return "byok"
	case commonv1.ProviderTier_PROVIDER_TIER_VROOLI:
		return "vrooli"
	case commonv1.ProviderTier_PROVIDER_TIER_LOCAL:
		return "local"
	default:
		return ""
	}
}

// -----------------------------------------------------------------------------
// AudioFormat
// -----------------------------------------------------------------------------

func AudioFormatToProto(s string) commonv1.AudioFormat {
	switch s {
	case "wav":
		return commonv1.AudioFormat_AUDIO_FORMAT_WAV
	case "mp3":
		return commonv1.AudioFormat_AUDIO_FORMAT_MP3
	case "flac":
		return commonv1.AudioFormat_AUDIO_FORMAT_FLAC
	case "ogg":
		return commonv1.AudioFormat_AUDIO_FORMAT_OGG
	case "webm":
		return commonv1.AudioFormat_AUDIO_FORMAT_WEBM
	case "opus":
		return commonv1.AudioFormat_AUDIO_FORMAT_OPUS
	default:
		return commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED
	}
}

func AudioFormatFromProto(f commonv1.AudioFormat) string {
	switch f {
	case commonv1.AudioFormat_AUDIO_FORMAT_WAV:
		return "wav"
	case commonv1.AudioFormat_AUDIO_FORMAT_MP3:
		return "mp3"
	case commonv1.AudioFormat_AUDIO_FORMAT_FLAC:
		return "flac"
	case commonv1.AudioFormat_AUDIO_FORMAT_OGG:
		return "ogg"
	case commonv1.AudioFormat_AUDIO_FORMAT_WEBM:
		return "webm"
	case commonv1.AudioFormat_AUDIO_FORMAT_OPUS:
		return "opus"
	default:
		return ""
	}
}

// -----------------------------------------------------------------------------
// ResponseFormat
// -----------------------------------------------------------------------------

func ResponseFormatToProto(s string) commonv1.ResponseFormat {
	switch s {
	case "mp3":
		return commonv1.ResponseFormat_RESPONSE_FORMAT_MP3
	case "wav":
		return commonv1.ResponseFormat_RESPONSE_FORMAT_WAV
	case "opus":
		return commonv1.ResponseFormat_RESPONSE_FORMAT_OPUS
	case "flac":
		return commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC
	default:
		return commonv1.ResponseFormat_RESPONSE_FORMAT_UNSPECIFIED
	}
}

func ResponseFormatFromProto(f commonv1.ResponseFormat) string {
	switch f {
	case commonv1.ResponseFormat_RESPONSE_FORMAT_MP3:
		return "mp3"
	case commonv1.ResponseFormat_RESPONSE_FORMAT_WAV:
		return "wav"
	case commonv1.ResponseFormat_RESPONSE_FORMAT_OPUS:
		return "opus"
	case commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC:
		return "flac"
	default:
		return ""
	}
}

// -----------------------------------------------------------------------------
// STT enums
// -----------------------------------------------------------------------------

func SpeakerModeToProto(s string) sttv1.SpeakerMode {
	switch s {
	case "off":
		return sttv1.SpeakerMode_SPEAKER_MODE_OFF
	case "filter":
		return sttv1.SpeakerMode_SPEAKER_MODE_FILTER
	case "advisory":
		return sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY
	default:
		return sttv1.SpeakerMode_SPEAKER_MODE_UNSPECIFIED
	}
}

func SpeakerModeFromProto(m sttv1.SpeakerMode) string {
	switch m {
	case sttv1.SpeakerMode_SPEAKER_MODE_OFF:
		return "off"
	case sttv1.SpeakerMode_SPEAKER_MODE_FILTER:
		return "filter"
	case sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY:
		return "advisory"
	default:
		return ""
	}
}

func RejectBehaviorToProto(s string) sttv1.RejectBehavior {
	switch s {
	case "drop":
		return sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP
	case "show-muted", "show_muted":
		return sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED
	default:
		return sttv1.RejectBehavior_REJECT_BEHAVIOR_UNSPECIFIED
	}
}

func RejectBehaviorFromProto(r sttv1.RejectBehavior) string {
	switch r {
	case sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP:
		return "drop"
	case sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED:
		return "show-muted"
	default:
		return ""
	}
}

func StreamingModeToProto(s string) sttv1.StreamingMode {
	switch s {
	case "auto":
		return sttv1.StreamingMode_STREAMING_MODE_AUTO
	case "off":
		return sttv1.StreamingMode_STREAMING_MODE_OFF
	default:
		return sttv1.StreamingMode_STREAMING_MODE_UNSPECIFIED
	}
}

func StreamingModeFromProto(m sttv1.StreamingMode) string {
	switch m {
	case sttv1.StreamingMode_STREAMING_MODE_AUTO:
		return "auto"
	case sttv1.StreamingMode_STREAMING_MODE_OFF:
		return "off"
	default:
		return ""
	}
}

func StrategyPreferenceToProto(s string) sttv1.StrategyPreference {
	switch s {
	case "auto":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO
	case "vad":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD
	case "overlap":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_OVERLAP
	case "passthrough":
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_PASSTHROUGH
	default:
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_UNSPECIFIED
	}
}

func StrategyPreferenceFromProto(p sttv1.StrategyPreference) string {
	switch p {
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO:
		return "auto"
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD:
		return "vad"
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_OVERLAP:
		return "overlap"
	case sttv1.StrategyPreference_STRATEGY_PREFERENCE_PASSTHROUGH:
		return "passthrough"
	default:
		return ""
	}
}

// -----------------------------------------------------------------------------
// TTS / Summarize enums
// -----------------------------------------------------------------------------

func SummarizeLevelToProto(s string) summarizev1.SummarizeLevel {
	switch s {
	case "light":
		return summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT
	case "moderate":
		return summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE
	case "heavy":
		return summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY
	default:
		return summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_UNSPECIFIED
	}
}

func SummarizeLevelFromProto(l summarizev1.SummarizeLevel) string {
	switch l {
	case summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT:
		return "light"
	case summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE:
		return "moderate"
	case summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY:
		return "heavy"
	default:
		return ""
	}
}

// -----------------------------------------------------------------------------
// Session enums
// -----------------------------------------------------------------------------

func SessionTransportToProto(s string) sessionv1.SessionTransport {
	switch s {
	case "browser-voice", "browser_voice":
		return sessionv1.SessionTransport_SESSION_TRANSPORT_BROWSER_VOICE
	case "fake":
		return sessionv1.SessionTransport_SESSION_TRANSPORT_FAKE
	default:
		return sessionv1.SessionTransport_SESSION_TRANSPORT_UNSPECIFIED
	}
}

func SessionTransportFromProto(t sessionv1.SessionTransport) string {
	switch t {
	case sessionv1.SessionTransport_SESSION_TRANSPORT_BROWSER_VOICE:
		return "browser-voice"
	case sessionv1.SessionTransport_SESSION_TRANSPORT_FAKE:
		return "fake"
	default:
		return ""
	}
}

func VadStateToProto(s string) sessionv1.VadState {
	switch s {
	case "speech_start", "speech-start":
		return sessionv1.VadState_VAD_STATE_SPEECH_START
	case "speech_end", "speech-end":
		return sessionv1.VadState_VAD_STATE_SPEECH_END
	default:
		return sessionv1.VadState_VAD_STATE_UNSPECIFIED
	}
}

func BargeInReasonToProto(s string) sessionv1.BargeInReason {
	switch s {
	case "vad":
		return sessionv1.BargeInReason_BARGE_IN_REASON_VAD
	case "explicit":
		return sessionv1.BargeInReason_BARGE_IN_REASON_EXPLICIT
	default:
		return sessionv1.BargeInReason_BARGE_IN_REASON_UNSPECIFIED
	}
}

// -----------------------------------------------------------------------------
// Timestamps
// -----------------------------------------------------------------------------

// TimeToProto returns nil for zero times so the JSON encoding omits the
// field rather than producing the unix-epoch sentinel.
func TimeToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t.UTC())
}

// TimeFromProto returns the zero time when ts is nil so callers don't
// have to nil-check before formatting.
func TimeFromProto(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// -----------------------------------------------------------------------------
// FieldMask helpers
// -----------------------------------------------------------------------------

// MaskHas reports whether the FieldMask contains the given path. An empty
// or nil mask returns false (callers should reject empty masks at the
// validation boundary).
func MaskHas(m *fieldmaskpb.FieldMask, path string) bool {
	if m == nil {
		return false
	}
	for _, p := range m.GetPaths() {
		if p == path {
			return true
		}
	}
	return false
}

// MaskPathsOutsideAllowed returns any paths in the mask that are not in
// the allowed set. The caller uses this to reject Update*Request payloads
// referencing unknown fields with CodeInvalidArgument.
func MaskPathsOutsideAllowed(m *fieldmaskpb.FieldMask, allowed map[string]struct{}) []string {
	if m == nil {
		return nil
	}
	var bad []string
	for _, p := range m.GetPaths() {
		if _, ok := allowed[p]; !ok {
			bad = append(bad, p)
		}
	}
	return bad
}
