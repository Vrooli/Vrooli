// Package audio_admin is web-console's audio configuration surface. The
// UI talks to this handler same-origin; the handler delegates to
// internal/audioports.* (Remote* adapters) for inter-scenario calls.
//
// Wire proto: packages/proto/schemas/web-console/v1/audio_admin/audio_admin.proto.
package audio_admin

import (
	"web-console/internal/audioports"

	audioadminv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_admin"
	audiocommonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/audio_common"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// -----------------------------------------------------------------------------
// Stream config
// -----------------------------------------------------------------------------

func streamConfigToProto(s audioports.StreamConfig) *audioadminv1.StreamConfig {
	return &audioadminv1.StreamConfig{
		FlushIntervalMs:    s.FlushIntervalMs,
		MinDeltaBytes:      s.MinDeltaBytes,
		OverlapBytes:       s.OverlapBytes,
		PersistentMode:     s.PersistentMode,
		WakeWordEnabled:    s.WakeWordEnabled,
		WakeWordThreshold:  s.WakeWordThreshold,
		SegmentSilenceMs:   s.SegmentSilenceMs,
		StreamingMode:      audiocommonv1.StreamingMode(s.StreamingMode),
		StrategyPreference: audiocommonv1.StrategyPreference(s.StrategyPreference),
		VadSilenceMs:       s.VadSilenceMs,
		OverlapWindowMs:    s.OverlapWindowMs,
		OverlapCommitRuns:  s.OverlapCommitRuns,
	}
}

func streamConfigFromProto(p *audioadminv1.StreamConfig) audioports.StreamConfig {
	if p == nil {
		return audioports.StreamConfig{}
	}
	return audioports.StreamConfig{
		FlushIntervalMs:    p.FlushIntervalMs,
		MinDeltaBytes:      p.MinDeltaBytes,
		OverlapBytes:       p.OverlapBytes,
		PersistentMode:     p.PersistentMode,
		WakeWordEnabled:    p.WakeWordEnabled,
		WakeWordThreshold:  p.WakeWordThreshold,
		SegmentSilenceMs:   p.SegmentSilenceMs,
		StreamingMode:      audioports.StreamingMode(p.StreamingMode),
		StrategyPreference: audioports.StrategyPreference(p.StrategyPreference),
		VadSilenceMs:       p.VadSilenceMs,
		OverlapWindowMs:    p.OverlapWindowMs,
		OverlapCommitRuns:  p.OverlapCommitRuns,
	}
}

// -----------------------------------------------------------------------------
// Wake word
// -----------------------------------------------------------------------------

func wakeWordTemplateFromProto(p *audioadminv1.WakeWordTemplate) audioports.WakeWordTemplate {
	if p == nil {
		return audioports.WakeWordTemplate{}
	}
	out := audioports.WakeWordTemplate{
		Label:     p.Label,
		Threshold: p.Threshold,
		Samples:   make([]audioports.WakeWordSample, 0, len(p.Samples)),
	}
	if p.UpdatedAt != nil {
		out.UpdatedAt = p.UpdatedAt.AsTime()
	}
	for _, s := range p.Samples {
		out.Samples = append(out.Samples, audioports.WakeWordSample{
			Audio:        s.Audio,
			Format:       audioports.AudioFormat(s.Format),
			SampleRateHz: s.SampleRateHz,
		})
	}
	return out
}

func wakeWordTemplateToProto(t *audioports.WakeWordTemplate) *audioadminv1.WakeWordTemplate {
	if t == nil {
		return nil
	}
	out := &audioadminv1.WakeWordTemplate{
		Label:     t.Label,
		Threshold: t.Threshold,
		Samples:   make([]*audioadminv1.WakeWordSample, 0, len(t.Samples)),
	}
	if !t.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(t.UpdatedAt)
	}
	for _, s := range t.Samples {
		out.Samples = append(out.Samples, &audioadminv1.WakeWordSample{
			Audio:        s.Audio,
			Format:       audiocommonv1.AudioFormat(s.Format),
			SampleRateHz: s.SampleRateHz,
		})
	}
	return out
}

func wakeWordConfigToProto(c audioports.WakeWordConfig) *audioadminv1.WakeWordConfig {
	return &audioadminv1.WakeWordConfig{
		Configured: c.Configured,
		Template:   wakeWordTemplateToProto(c.Template),
	}
}

// -----------------------------------------------------------------------------
// Speaker
// -----------------------------------------------------------------------------

func speakerConfigFromProto(p *audioadminv1.SpeakerConfig) audioports.SpeakerConfig {
	if p == nil {
		return audioports.SpeakerConfig{}
	}
	return audioports.SpeakerConfig{
		Enabled:                     p.Enabled,
		ProfileIDs:                  append([]string(nil), p.ProfileIds...),
		Threshold:                   p.Threshold,
		Mode:                        audioports.SpeakerMode(p.Mode),
		RejectBehavior:              audioports.RejectBehavior(p.RejectBehavior),
		FallbackWithoutVerification: p.FallbackWithoutVerification,
		ExtractionEnabled:           p.ExtractionEnabled,
	}
}

func speakerConfigToProto(s audioports.SpeakerConfig) *audioadminv1.SpeakerConfig {
	return &audioadminv1.SpeakerConfig{
		Enabled:                     s.Enabled,
		ProfileIds:                  append([]string(nil), s.ProfileIDs...),
		Threshold:                   s.Threshold,
		Mode:                        audiocommonv1.SpeakerMode(s.Mode),
		RejectBehavior:              audiocommonv1.RejectBehavior(s.RejectBehavior),
		FallbackWithoutVerification: s.FallbackWithoutVerification,
		ExtractionEnabled:           s.ExtractionEnabled,
	}
}

func speakerProfileToProto(p audioports.SpeakerProfile) *audioadminv1.SpeakerProfile {
	out := &audioadminv1.SpeakerProfile{
		Id:                     p.ID,
		DisplayName:            p.DisplayName,
		ModelName:              p.ModelName,
		EmbeddingDim:           p.EmbeddingDim,
		SampleRate:             p.SampleRate,
		EnrollmentAudioSeconds: p.EnrollmentAudioSeconds,
		Notes:                  p.Notes,
	}
	if !p.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(p.CreatedAt)
	}
	if !p.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(p.UpdatedAt)
	}
	return out
}

func speakerStatusToProto(s audioports.SpeakerStatus) *audioadminv1.SpeakerStatus {
	out := &audioadminv1.SpeakerStatus{
		Config:            speakerConfigToProto(s.Config),
		Capability:        audiocommonv1.SpeakerCapability(s.Capability),
		CapabilityLabel:   s.CapabilityLabel,
		ResourceReady:     s.ResourceReady,
		ProfileConfigured: s.ProfileConfigured,
		ProfileExists:     s.ProfileExists,
		ProfileCount:      s.ProfileCount,
		Profiles:          make([]*audioadminv1.SpeakerProfile, 0, len(s.Profiles)),
	}
	for _, p := range s.Profiles {
		out.Profiles = append(out.Profiles, speakerProfileToProto(p))
	}
	if s.Info != nil {
		out.Info = &audioadminv1.SpeakerResourceInfo{
			Backend:      s.Info.Backend,
			Model:        s.Info.Model,
			Device:       s.Info.Device,
			SampleRate:   s.Info.SampleRate,
			Version:      s.Info.Version,
			EmbeddingDim: s.Info.EmbeddingDim,
		}
	}
	if !s.CheckedAt.IsZero() {
		out.CheckedAt = timestamppb.New(s.CheckedAt)
	}
	return out
}

func speakerEnrollmentToProto(e audioports.SpeakerEnrollment) *audioadminv1.SpeakerEnrollment {
	out := &audioadminv1.SpeakerEnrollment{
		ProfileId:              e.ProfileID,
		DisplayName:            e.DisplayName,
		EmbeddingDim:           e.EmbeddingDim,
		SampleRate:             e.SampleRate,
		EnrollmentAudioSeconds: e.EnrollmentAudioSeconds,
		ModelName:              e.ModelName,
	}
	if !e.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(e.CreatedAt)
	}
	return out
}

// -----------------------------------------------------------------------------
// TTS / Summarize config
// -----------------------------------------------------------------------------

func ttsConfigFromProto(p *audioadminv1.TTSConfig) audioports.TTSConfig {
	if p == nil {
		return audioports.TTSConfig{}
	}
	return audioports.TTSConfig{
		AutoEnabled:           p.AutoEnabled,
		DefaultVoice:          p.DefaultVoice,
		DefaultSpeed:          p.DefaultSpeed,
		DefaultResponseFormat: audioports.ResponseFormat(p.DefaultResponseFormat),
	}
}

func ttsConfigToProto(t audioports.TTSConfig) *audioadminv1.TTSConfig {
	return &audioadminv1.TTSConfig{
		AutoEnabled:           t.AutoEnabled,
		DefaultVoice:          t.DefaultVoice,
		DefaultSpeed:          t.DefaultSpeed,
		DefaultResponseFormat: audiocommonv1.ResponseFormat(t.DefaultResponseFormat),
	}
}

func summarizeConfigFromProto(p *audioadminv1.SummarizeConfig) audioports.SummarizeConfig {
	if p == nil {
		return audioports.SummarizeConfig{}
	}
	return audioports.SummarizeConfig{
		Enabled:        p.Enabled,
		CharThreshold:  p.CharThreshold,
		Level:          audioports.SummarizeLevel(p.Level),
		Model:          p.Model,
		TimeoutSeconds: p.TimeoutSeconds,
	}
}

func summarizeConfigToProto(s audioports.SummarizeConfig) *audioadminv1.SummarizeConfig {
	return &audioadminv1.SummarizeConfig{
		Enabled:        s.Enabled,
		CharThreshold:  s.CharThreshold,
		Level:          audiocommonv1.SummarizeLevel(s.Level),
		Model:          s.Model,
		TimeoutSeconds: s.TimeoutSeconds,
	}
}
