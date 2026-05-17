// contracts.go — central type/envelope normalization between web-console's
// audioports surface and the audio-tools wire shape.
//
// All Remote* adapters live in this package and consume audio-tools proto
// types directly; web-console handlers consume audioports types so the
// audio-tools wire shape never crosses the handler boundary. This file is
// the single conversion point.
package audioports

import (
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	summv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// -----------------------------------------------------------------------------
// Typed enums (web-console-local; mirror of the proto enums in
// packages/proto/schemas/web-console/v1/audio_common/audio_common.proto so
// handler code stays free of audio-tools proto imports).
// -----------------------------------------------------------------------------

type AudioFormat int32

const (
	AudioFormatUnspecified AudioFormat = 0
	AudioFormatWAV         AudioFormat = 1
	AudioFormatMP3         AudioFormat = 2
	AudioFormatFLAC        AudioFormat = 3
	AudioFormatOGG         AudioFormat = 4
	AudioFormatWebM        AudioFormat = 5
	AudioFormatOPUS        AudioFormat = 6
	AudioFormatAAC         AudioFormat = 7
)

func (f AudioFormat) toProto() commonv1.AudioFormat {
	if f < 0 || int(f) > int(commonv1.AudioFormat_AUDIO_FORMAT_AAC) {
		return commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED
	}
	return commonv1.AudioFormat(f)
}

type ResponseFormat int32

const (
	ResponseFormatUnspecified ResponseFormat = 0
	ResponseFormatMP3         ResponseFormat = 1
	ResponseFormatWAV         ResponseFormat = 2
	ResponseFormatOPUS        ResponseFormat = 3
	ResponseFormatFLAC        ResponseFormat = 4
)

func (f ResponseFormat) toProto() commonv1.ResponseFormat {
	if f < 0 || int(f) > int(commonv1.ResponseFormat_RESPONSE_FORMAT_FLAC) {
		return commonv1.ResponseFormat_RESPONSE_FORMAT_UNSPECIFIED
	}
	return commonv1.ResponseFormat(f)
}

func responseFormatFromProto(p commonv1.ResponseFormat) ResponseFormat {
	return ResponseFormat(p)
}

type ProviderTier int32

const (
	ProviderTierUnspecified ProviderTier = 0
	ProviderTierLocal       ProviderTier = 1
	ProviderTierBYOK        ProviderTier = 2
	ProviderTierVrooli      ProviderTier = 3
)

func providerTierFromProto(p commonv1.ProviderTier) ProviderTier {
	return ProviderTier(p)
}

type SpeakerMode int32

const (
	SpeakerModeUnspecified SpeakerMode = 0
	SpeakerModeOff         SpeakerMode = 1
	SpeakerModeFilter      SpeakerMode = 2
	SpeakerModeAdvisory    SpeakerMode = 3
)

func (m SpeakerMode) toProto() sttv1.SpeakerMode {
	if m < 0 || int(m) > int(sttv1.SpeakerMode_SPEAKER_MODE_ADVISORY) {
		return sttv1.SpeakerMode_SPEAKER_MODE_UNSPECIFIED
	}
	return sttv1.SpeakerMode(m)
}

func speakerModeFromProto(p sttv1.SpeakerMode) SpeakerMode { return SpeakerMode(p) }

type RejectBehavior int32

const (
	RejectBehaviorUnspecified RejectBehavior = 0
	RejectBehaviorDrop        RejectBehavior = 1
	RejectBehaviorShowMuted   RejectBehavior = 2
)

func (r RejectBehavior) toProto() sttv1.RejectBehavior {
	if r < 0 || int(r) > int(sttv1.RejectBehavior_REJECT_BEHAVIOR_SHOW_MUTED) {
		return sttv1.RejectBehavior_REJECT_BEHAVIOR_UNSPECIFIED
	}
	return sttv1.RejectBehavior(r)
}

func rejectBehaviorFromProto(p sttv1.RejectBehavior) RejectBehavior { return RejectBehavior(p) }

type StreamingMode int32

const (
	StreamingModeUnspecified StreamingMode = 0
	StreamingModeAuto        StreamingMode = 1
	StreamingModeOff         StreamingMode = 2
)

func (m StreamingMode) toProto() sttv1.StreamingMode {
	if m < 0 || int(m) > int(sttv1.StreamingMode_STREAMING_MODE_OFF) {
		return sttv1.StreamingMode_STREAMING_MODE_UNSPECIFIED
	}
	return sttv1.StreamingMode(m)
}

func streamingModeFromProto(p sttv1.StreamingMode) StreamingMode { return StreamingMode(p) }

type StrategyPreference int32

const (
	StrategyPreferenceUnspecified StrategyPreference = 0
	StrategyPreferenceAuto        StrategyPreference = 1
	StrategyPreferenceVAD         StrategyPreference = 2
	StrategyPreferenceOverlap     StrategyPreference = 3
	StrategyPreferencePassthrough StrategyPreference = 4
)

func (s StrategyPreference) toProto() sttv1.StrategyPreference {
	if s < 0 || int(s) > int(sttv1.StrategyPreference_STRATEGY_PREFERENCE_PASSTHROUGH) {
		return sttv1.StrategyPreference_STRATEGY_PREFERENCE_UNSPECIFIED
	}
	return sttv1.StrategyPreference(s)
}

func strategyPreferenceFromProto(p sttv1.StrategyPreference) StrategyPreference {
	return StrategyPreference(p)
}

type SummarizeLevel int32

const (
	SummarizeLevelUnspecified SummarizeLevel = 0
	SummarizeLevelLight       SummarizeLevel = 1
	SummarizeLevelModerate    SummarizeLevel = 2
	SummarizeLevelHeavy       SummarizeLevel = 3
)

func (l SummarizeLevel) toProto() summv1.SummarizeLevel {
	if l < 0 || int(l) > int(summv1.SummarizeLevel_SUMMARIZE_LEVEL_HEAVY) {
		return summv1.SummarizeLevel_SUMMARIZE_LEVEL_UNSPECIFIED
	}
	return summv1.SummarizeLevel(l)
}

func summarizeLevelFromProto(p summv1.SummarizeLevel) SummarizeLevel { return SummarizeLevel(p) }

// SpeakerCapability normalizes the stringly `capability` value returned by
// audio-tools' GetSpeakerStatus into a typed enum the handler/UI surface.
type SpeakerCapability int32

const (
	SpeakerCapabilityUnspecified   SpeakerCapability = 0
	SpeakerCapabilityAvailable     SpeakerCapability = 1
	SpeakerCapabilityDegraded      SpeakerCapability = 2
	SpeakerCapabilityUnavailable   SpeakerCapability = 3
	SpeakerCapabilityUninitialized SpeakerCapability = 4
)

func speakerCapabilityFromString(s string) SpeakerCapability {
	switch s {
	case "available":
		return SpeakerCapabilityAvailable
	case "degraded":
		return SpeakerCapabilityDegraded
	case "unavailable":
		return SpeakerCapabilityUnavailable
	case "uninitialized", "":
		return SpeakerCapabilityUninitialized
	default:
		return SpeakerCapabilityUnspecified
	}
}

// -----------------------------------------------------------------------------
// Domain structs (carry the same fields the proto messages do but live in
// the audioports package so handler code never imports an audio-tools proto
// type).
// -----------------------------------------------------------------------------

type StreamConfig struct {
	FlushIntervalMs    int32
	MinDeltaBytes      int32
	OverlapBytes       int32
	PersistentMode     bool
	WakeWordEnabled    bool
	WakeWordThreshold  float64
	SegmentSilenceMs   int32
	StreamingMode      StreamingMode
	StrategyPreference StrategyPreference
	VadSilenceMs       int32
	OverlapWindowMs    int32
	OverlapCommitRuns  int32
}

type WakeWordSample struct {
	Audio        []byte
	Format       AudioFormat
	SampleRateHz int32
}

type WakeWordTemplate struct {
	Label     string
	Threshold float64
	Samples   []WakeWordSample
	UpdatedAt time.Time
}

type WakeWordConfig struct {
	Configured bool
	Template   *WakeWordTemplate
}

type SpeakerConfig struct {
	Enabled                     bool
	ProfileIDs                  []string
	Threshold                   float64
	Mode                        SpeakerMode
	RejectBehavior              RejectBehavior
	FallbackWithoutVerification bool
	ExtractionEnabled           bool
}

type SpeakerProfile struct {
	ID                     string
	DisplayName            string
	CreatedAt              time.Time
	UpdatedAt              time.Time
	ModelName              string
	EmbeddingDim           int32
	SampleRate             int32
	EnrollmentAudioSeconds float64
	Notes                  string
}

type SpeakerResourceInfo struct {
	Backend      string
	Model        string
	Device       string
	SampleRate   int32
	Version      string
	EmbeddingDim int32
}

type SpeakerStatus struct {
	Config            SpeakerConfig
	Capability        SpeakerCapability
	CapabilityLabel   string
	ResourceReady     bool
	ProfileConfigured bool
	ProfileExists     bool
	ProfileCount      int32
	Profiles          []SpeakerProfile
	Info              *SpeakerResourceInfo
	CheckedAt         time.Time
}

type SpeakerEnrollment struct {
	ProfileID              string
	DisplayName            string
	EmbeddingDim           int32
	SampleRate             int32
	EnrollmentAudioSeconds float64
	ModelName              string
	CreatedAt              time.Time
}

type SpeakerEnrollResult struct {
	Enrollment SpeakerEnrollment
	Config     SpeakerConfig
}

type TTSConfig struct {
	AutoEnabled           bool
	DefaultVoice          string
	DefaultSpeed          float64
	DefaultResponseFormat ResponseFormat
}

type SummarizeConfig struct {
	Enabled        bool
	CharThreshold  int32
	Level          SummarizeLevel
	Model          string
	TimeoutSeconds int32
}

// -----------------------------------------------------------------------------
// Proto <-> domain mappers
// -----------------------------------------------------------------------------

func streamConfigFromProto(p *sttv1.StreamConfig) StreamConfig {
	if p == nil {
		return StreamConfig{}
	}
	return StreamConfig{
		FlushIntervalMs:    p.FlushIntervalMs,
		MinDeltaBytes:      p.MinDeltaBytes,
		OverlapBytes:       p.OverlapBytes,
		PersistentMode:     p.PersistentMode,
		WakeWordEnabled:    p.WakeWordEnabled,
		WakeWordThreshold:  p.WakeWordThreshold,
		SegmentSilenceMs:   p.SegmentSilenceMs,
		StreamingMode:      streamingModeFromProto(p.StreamingMode),
		StrategyPreference: strategyPreferenceFromProto(p.StrategyPreference),
		VadSilenceMs:       p.VadSilenceMs,
		OverlapWindowMs:    p.OverlapWindowMs,
		OverlapCommitRuns:  p.OverlapCommitRuns,
	}
}

func streamConfigToProto(s StreamConfig) *sttv1.StreamConfig {
	return &sttv1.StreamConfig{
		FlushIntervalMs:    s.FlushIntervalMs,
		MinDeltaBytes:      s.MinDeltaBytes,
		OverlapBytes:       s.OverlapBytes,
		PersistentMode:     s.PersistentMode,
		WakeWordEnabled:    s.WakeWordEnabled,
		WakeWordThreshold:  s.WakeWordThreshold,
		SegmentSilenceMs:   s.SegmentSilenceMs,
		StreamingMode:      s.StreamingMode.toProto(),
		StrategyPreference: s.StrategyPreference.toProto(),
		VadSilenceMs:       s.VadSilenceMs,
		OverlapWindowMs:    s.OverlapWindowMs,
		OverlapCommitRuns:  s.OverlapCommitRuns,
	}
}

func wakeWordTemplateFromProto(p *sttv1.WakeWordTemplate) *WakeWordTemplate {
	if p == nil {
		return nil
	}
	out := &WakeWordTemplate{
		Label:     p.Label,
		Threshold: p.Threshold,
		Samples:   make([]WakeWordSample, 0, len(p.Samples)),
	}
	if p.UpdatedAt != nil {
		out.UpdatedAt = p.UpdatedAt.AsTime()
	}
	for _, s := range p.Samples {
		out.Samples = append(out.Samples, WakeWordSample{
			Audio:        s.Audio,
			Format:       AudioFormat(s.Format),
			SampleRateHz: s.SampleRateHz,
		})
	}
	return out
}

func wakeWordTemplateToProto(t *WakeWordTemplate) *sttv1.WakeWordTemplate {
	if t == nil {
		return nil
	}
	out := &sttv1.WakeWordTemplate{
		Label:     t.Label,
		Threshold: t.Threshold,
		Samples:   make([]*sttv1.WakeWordSample, 0, len(t.Samples)),
	}
	for _, s := range t.Samples {
		out.Samples = append(out.Samples, &sttv1.WakeWordSample{
			Audio:        s.Audio,
			Format:       s.Format.toProto(),
			SampleRateHz: s.SampleRateHz,
		})
	}
	return out
}

func wakeWordConfigFromProto(p *sttv1.WakeWordConfig) WakeWordConfig {
	if p == nil {
		return WakeWordConfig{}
	}
	return WakeWordConfig{
		Configured: p.Configured,
		Template:   wakeWordTemplateFromProto(p.Template),
	}
}

func speakerConfigFromProto(p *sttv1.SpeakerConfig) SpeakerConfig {
	if p == nil {
		return SpeakerConfig{}
	}
	return SpeakerConfig{
		Enabled:                     p.Enabled,
		ProfileIDs:                  append([]string(nil), p.ProfileIds...),
		Threshold:                   p.Threshold,
		Mode:                        speakerModeFromProto(p.Mode),
		RejectBehavior:              rejectBehaviorFromProto(p.RejectBehavior),
		FallbackWithoutVerification: p.FallbackWithoutVerification,
		ExtractionEnabled:           p.ExtractionEnabled,
	}
}

func speakerConfigToProto(s SpeakerConfig) *sttv1.SpeakerConfig {
	return &sttv1.SpeakerConfig{
		Enabled:                     s.Enabled,
		ProfileIds:                  append([]string(nil), s.ProfileIDs...),
		Threshold:                   s.Threshold,
		Mode:                        s.Mode.toProto(),
		RejectBehavior:              s.RejectBehavior.toProto(),
		FallbackWithoutVerification: s.FallbackWithoutVerification,
		ExtractionEnabled:           s.ExtractionEnabled,
	}
}

func speakerProfileFromProto(p *sttv1.SpeakerProfile) SpeakerProfile {
	if p == nil {
		return SpeakerProfile{}
	}
	out := SpeakerProfile{
		ID:                     p.Id,
		DisplayName:            p.DisplayName,
		ModelName:              p.ModelName,
		EmbeddingDim:           p.EmbeddingDim,
		SampleRate:             p.SampleRate,
		EnrollmentAudioSeconds: p.EnrollmentAudioSeconds,
		Notes:                  p.Notes,
	}
	if p.CreatedAt != nil {
		out.CreatedAt = p.CreatedAt.AsTime()
	}
	if p.UpdatedAt != nil {
		out.UpdatedAt = p.UpdatedAt.AsTime()
	}
	return out
}

func speakerStatusFromProto(p *sttv1.SpeakerStatus) SpeakerStatus {
	if p == nil {
		return SpeakerStatus{}
	}
	out := SpeakerStatus{
		Config:            speakerConfigFromProto(p.Config),
		Capability:        speakerCapabilityFromString(p.Capability),
		CapabilityLabel:   p.CapabilityLabel,
		ResourceReady:     p.ResourceReady,
		ProfileConfigured: p.ProfileConfigured,
		ProfileExists:     p.ProfileExists,
		ProfileCount:      p.ProfileCount,
		Profiles:          make([]SpeakerProfile, 0, len(p.Profiles)),
	}
	for _, prof := range p.Profiles {
		out.Profiles = append(out.Profiles, speakerProfileFromProto(prof))
	}
	if p.Info != nil {
		out.Info = &SpeakerResourceInfo{
			Backend:      p.Info.Backend,
			Model:        p.Info.Model,
			Device:       p.Info.Device,
			SampleRate:   p.Info.SampleRate,
			Version:      p.Info.Version,
			EmbeddingDim: p.Info.EmbeddingDim,
		}
	}
	if p.CheckedAt != nil {
		out.CheckedAt = p.CheckedAt.AsTime()
	}
	return out
}

func speakerEnrollmentFromProto(p *sttv1.SpeakerEnrollment) SpeakerEnrollment {
	if p == nil {
		return SpeakerEnrollment{}
	}
	out := SpeakerEnrollment{
		ProfileID:              p.ProfileId,
		DisplayName:            p.DisplayName,
		EmbeddingDim:           p.EmbeddingDim,
		SampleRate:             p.SampleRate,
		EnrollmentAudioSeconds: p.EnrollmentAudioSeconds,
		ModelName:              p.ModelName,
	}
	if p.CreatedAt != nil {
		out.CreatedAt = p.CreatedAt.AsTime()
	}
	return out
}

func summarizeConfigFromProto(p *summv1.SummarizeConfig) SummarizeConfig {
	if p == nil {
		return SummarizeConfig{}
	}
	return SummarizeConfig{
		Enabled:        p.Enabled,
		CharThreshold:  p.CharThreshold,
		Level:          summarizeLevelFromProto(p.Level),
		Model:          p.Model,
		TimeoutSeconds: p.TimeoutSeconds,
	}
}

func ttsConfigFromProto(p *ttsv1.Config) TTSConfig {
	if p == nil {
		return TTSConfig{}
	}
	return TTSConfig{
		AutoEnabled:           p.AutoEnabled,
		DefaultVoice:          p.DefaultVoice,
		DefaultSpeed:          p.DefaultSpeed,
		DefaultResponseFormat: responseFormatFromProto(p.DefaultResponseFormat),
	}
}

func ttsConfigToProto(t TTSConfig) *ttsv1.Config {
	return &ttsv1.Config{
		AutoEnabled:           t.AutoEnabled,
		DefaultVoice:          t.DefaultVoice,
		DefaultSpeed:          t.DefaultSpeed,
		DefaultResponseFormat: t.DefaultResponseFormat.toProto(),
	}
}

func summarizeConfigToProto(s SummarizeConfig) *summv1.SummarizeConfig {
	return &summv1.SummarizeConfig{
		Enabled:        s.Enabled,
		CharThreshold:  s.CharThreshold,
		Level:          s.Level.toProto(),
		Model:          s.Model,
		TimeoutSeconds: s.TimeoutSeconds,
	}
}
