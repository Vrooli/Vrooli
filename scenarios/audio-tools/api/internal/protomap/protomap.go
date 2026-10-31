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
	return enumToProto(s, providerTierToProto, commonv1.ProviderTier_PROVIDER_TIER_UNSPECIFIED)
}

func ProviderTierFromProto(t commonv1.ProviderTier) string {
	return enumFromProto(t, providerTierFromProto)
}

var providerTierToProto = map[string]commonv1.ProviderTier{
	"byok": commonv1.ProviderTier_PROVIDER_TIER_BYOK, "vrooli": commonv1.ProviderTier_PROVIDER_TIER_VROOLI, "local": commonv1.ProviderTier_PROVIDER_TIER_LOCAL,
}

var providerTierFromProto = map[commonv1.ProviderTier]string{
	commonv1.ProviderTier_PROVIDER_TIER_BYOK: "byok", commonv1.ProviderTier_PROVIDER_TIER_VROOLI: "vrooli", commonv1.ProviderTier_PROVIDER_TIER_LOCAL: "local",
}

// -----------------------------------------------------------------------------
// AudioFormat
// -----------------------------------------------------------------------------

func AudioFormatToProto(s string) commonv1.AudioFormat {
	return enumToProto(s, audioFormatToProto, commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED)
}

func AudioFormatFromProto(f commonv1.AudioFormat) string {
	return enumFromProto(f, audioFormatFromProto)
}

var audioFormatToProto = map[string]commonv1.AudioFormat{
	"wav": commonv1.AudioFormat_AUDIO_FORMAT_WAV, "mp3": commonv1.AudioFormat_AUDIO_FORMAT_MP3, "flac": commonv1.AudioFormat_AUDIO_FORMAT_FLAC,
	"ogg": commonv1.AudioFormat_AUDIO_FORMAT_OGG, "webm": commonv1.AudioFormat_AUDIO_FORMAT_WEBM, "opus": commonv1.AudioFormat_AUDIO_FORMAT_OPUS,
	"aac": commonv1.AudioFormat_AUDIO_FORMAT_AAC, "pcm_s16le": commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE, "pcm": commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE,
}

var audioFormatFromProto = map[commonv1.AudioFormat]string{
	commonv1.AudioFormat_AUDIO_FORMAT_WAV: "wav", commonv1.AudioFormat_AUDIO_FORMAT_MP3: "mp3", commonv1.AudioFormat_AUDIO_FORMAT_FLAC: "flac",
	commonv1.AudioFormat_AUDIO_FORMAT_OGG: "ogg", commonv1.AudioFormat_AUDIO_FORMAT_WEBM: "webm", commonv1.AudioFormat_AUDIO_FORMAT_OPUS: "opus",
	commonv1.AudioFormat_AUDIO_FORMAT_AAC: "aac", commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE: "pcm_s16le",
}

// -----------------------------------------------------------------------------
// ResponseFormat
// -----------------------------------------------------------------------------

func ResponseFormatToProto(s string) commonv1.ResponseFormat {
	return enumToProto(s, responseFormatToProto, commonv1.ResponseFormat_RESPONSE_FORMAT_UNSPECIFIED)
}

func ResponseFormatFromProto(f commonv1.ResponseFormat) string {
	return enumFromProto(f, responseFormatFromProto)
}

var responseFormatToProto = map[string]commonv1.ResponseFormat{
	"mp3": commonv1.ResponseFormat_RESPONSE_FORMAT_MP3, "wav": commonv1.ResponseFormat_RESPONSE_FORMAT_WAV,
	"opus": commonv1.ResponseFormat_RESPONSE_FORMAT_OPUS, "flac": commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC,
}

var responseFormatFromProto = map[commonv1.ResponseFormat]string{
	commonv1.ResponseFormat_RESPONSE_FORMAT_MP3: "mp3", commonv1.ResponseFormat_RESPONSE_FORMAT_WAV: "wav",
	commonv1.ResponseFormat_RESPONSE_FORMAT_OPUS: "opus", commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC: "flac",
}

// -----------------------------------------------------------------------------
// STT enums
// -----------------------------------------------------------------------------

func SpeakerModeToProto(s string) sttv1.SpeakerMode {
	return enumToProto(s, speakerModeToProto, sttv1.SpeakerMode_SPEAKER_MODE_UNSPECIFIED)
}

func SpeakerModeFromProto(m sttv1.SpeakerMode) string {
	return enumFromProto(m, speakerModeFromProto)
}

var speakerModeToProto = map[string]sttv1.SpeakerMode{
	"off": sttv1.SpeakerMode_SPEAKER_MODE_OFF, "filter": sttv1.SpeakerMode_SPEAKER_MODE_FILTER, "advisory": sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY,
}

var speakerModeFromProto = map[sttv1.SpeakerMode]string{
	sttv1.SpeakerMode_SPEAKER_MODE_OFF: "off", sttv1.SpeakerMode_SPEAKER_MODE_FILTER: "filter", sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY: "advisory",
}

func RejectBehaviorToProto(s string) sttv1.RejectBehavior {
	return enumToProto(s, rejectBehaviorToProto, sttv1.RejectBehavior_REJECT_BEHAVIOR_UNSPECIFIED)
}

func RejectBehaviorFromProto(r sttv1.RejectBehavior) string {
	return enumFromProto(r, rejectBehaviorFromProto)
}

var rejectBehaviorToProto = map[string]sttv1.RejectBehavior{
	"drop": sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP, "show-muted": sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED, "show_muted": sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED,
}

var rejectBehaviorFromProto = map[sttv1.RejectBehavior]string{
	sttv1.RejectBehavior_REJECT_BEHAVIOR_DROP: "drop", sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED: "show-muted",
}

func StreamingModeToProto(s string) sttv1.StreamingMode {
	return enumToProto(s, streamingModeToProto, sttv1.StreamingMode_STREAMING_MODE_UNSPECIFIED)
}

func StreamingModeFromProto(m sttv1.StreamingMode) string {
	return enumFromProto(m, streamingModeFromProto)
}

var (
	streamingModeToProto   = map[string]sttv1.StreamingMode{"auto": sttv1.StreamingMode_STREAMING_MODE_AUTO, "off": sttv1.StreamingMode_STREAMING_MODE_OFF}
	streamingModeFromProto = map[sttv1.StreamingMode]string{sttv1.StreamingMode_STREAMING_MODE_AUTO: "auto", sttv1.StreamingMode_STREAMING_MODE_OFF: "off"}
)

func StrategyPreferenceToProto(s string) sttv1.StrategyPreference {
	return enumToProto(s, strategyPreferenceToProto, sttv1.StrategyPreference_STRATEGY_PREFERENCE_UNSPECIFIED)
}

func StrategyPreferenceFromProto(p sttv1.StrategyPreference) string {
	return enumFromProto(p, strategyPreferenceFromProto)
}

var strategyPreferenceToProto = map[string]sttv1.StrategyPreference{
	"auto": sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO, "vad": sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD, "overlap": sttv1.StrategyPreference_STRATEGY_PREFERENCE_OVERLAP, "passthrough": sttv1.StrategyPreference_STRATEGY_PREFERENCE_PASSTHROUGH,
}

var strategyPreferenceFromProto = map[sttv1.StrategyPreference]string{
	sttv1.StrategyPreference_STRATEGY_PREFERENCE_AUTO: "auto", sttv1.StrategyPreference_STRATEGY_PREFERENCE_VAD: "vad", sttv1.StrategyPreference_STRATEGY_PREFERENCE_OVERLAP: "overlap", sttv1.StrategyPreference_STRATEGY_PREFERENCE_PASSTHROUGH: "passthrough",
}

// -----------------------------------------------------------------------------
// TTS / Summarize enums
// -----------------------------------------------------------------------------

func SummarizeLevelToProto(s string) summarizev1.SummarizeLevel {
	return enumToProto(s, summarizeLevelToProto, summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_UNSPECIFIED)
}

func SummarizeLevelFromProto(l summarizev1.SummarizeLevel) string {
	return enumFromProto(l, summarizeLevelFromProto)
}

var summarizeLevelToProto = map[string]summarizev1.SummarizeLevel{
	"light": summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT, "moderate": summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE, "heavy": summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY,
}

var summarizeLevelFromProto = map[summarizev1.SummarizeLevel]string{
	summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_LIGHT: "light", summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_MODERATE: "moderate", summarizev1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY: "heavy",
}

func enumToProto[T comparable](value string, values map[string]T, unspecified T) T {
	if mapped, ok := values[value]; ok {
		return mapped
	}
	return unspecified
}

func enumFromProto[T comparable](value T, values map[T]string) string { return values[value] }

// -----------------------------------------------------------------------------
// Session enums
// -----------------------------------------------------------------------------

func SessionTransportToProto(s string) sessionv1.SessionTransport {
	return enumToProto(s, sessionTransportToProto, sessionv1.SessionTransport_SESSION_TRANSPORT_UNSPECIFIED)
}

func SessionTransportFromProto(t sessionv1.SessionTransport) string {
	return enumFromProto(t, sessionTransportFromProto)
}

var sessionTransportToProto = map[string]sessionv1.SessionTransport{
	"browser-voice": sessionv1.SessionTransport_SESSION_TRANSPORT_BROWSER_VOICE, "browser_voice": sessionv1.SessionTransport_SESSION_TRANSPORT_BROWSER_VOICE, "fake": sessionv1.SessionTransport_SESSION_TRANSPORT_FAKE,
}

var sessionTransportFromProto = map[sessionv1.SessionTransport]string{
	sessionv1.SessionTransport_SESSION_TRANSPORT_BROWSER_VOICE: "browser-voice", sessionv1.SessionTransport_SESSION_TRANSPORT_FAKE: "fake",
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
