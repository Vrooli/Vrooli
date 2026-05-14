package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	voiceH "web-console/handlers/voice"
)

// voiceServiceShim adapts package-main *Server (with its internal voice
// config types and resource clients) to the transport-neutral voiceH.Deps
// interface. All internal↔transport conversion happens here so
// handlers/voice stays free of package-main type references.
type voiceServiceShim struct {
	s *Server
}

func newVoiceServiceShim(s *Server) *voiceServiceShim { return &voiceServiceShim{s: s} }

// ----- Capability / metrics -----

func (v *voiceServiceShim) WhisperAvailable(ctx context.Context) bool {
	return v.s.capabilities.IsAvailable(ctx, "whisper-stt")
}

func (v *voiceServiceShim) IncrSkipVerification() {
	if v.s.metrics != nil {
		v.s.metrics.VoiceSkipVerificationTotal.Add(1)
	}
}

func (v *voiceServiceShim) SpeakerCapability(ctx context.Context) (string, string) {
	for _, cap := range v.s.capabilities.ResolveLiveness(ctx) {
		if cap.ID == "speaker-verification" {
			return string(cap.Status), cap.Message
		}
	}
	return "", ""
}

// ----- Transcribe path -----

func (v *voiceServiceShim) EvaluateSpeaker(ctx context.Context, audio []byte) voiceH.SpeakerDecision {
	d := v.s.evaluateSpeakerVerification(ctx, audio)
	return voiceH.SpeakerDecision{
		Enabled:      d.Enabled,
		Applied:      d.Applied,
		Allowed:      d.Allowed,
		Matched:      d.Matched,
		ProfileID:    d.ProfileID,
		Score:        d.Score,
		Threshold:    d.Threshold,
		Mode:         d.Mode,
		ErrorMessage: d.ErrorMessage,
	}
}

func (v *voiceServiceShim) FormatSpeakerDecisionError(d voiceH.SpeakerDecision) string {
	return formatSpeakerDecisionError(speakerVerificationGateDecision{
		Enabled:      d.Enabled,
		Applied:      d.Applied,
		Allowed:      d.Allowed,
		Matched:      d.Matched,
		ProfileID:    d.ProfileID,
		Score:        d.Score,
		Threshold:    d.Threshold,
		Mode:         d.Mode,
		ErrorMessage: d.ErrorMessage,
	})
}

func (v *voiceServiceShim) Transcribe(ctx context.Context, audio []byte, language string) (string, error) {
	return v.s.transcribeBytes(ctx, audio, language, true, "")
}

func (v *voiceServiceShim) IsWhisperHallucination(text string) bool {
	return isWhisperHallucination(text)
}

// ----- Stream config -----

func (v *voiceServiceShim) GetStreamConfig() voiceH.StreamConfig {
	return streamConfigToTransport(v.s.getVoiceConfig())
}

func (v *voiceServiceShim) SaveStreamConfig(c voiceH.StreamConfig) error {
	internal := VoiceStreamConfig{
		FlushIntervalMs:   c.FlushIntervalMs,
		MinDeltaBytes:     c.MinDeltaBytes,
		OverlapBytes:      c.OverlapBytes,
		PersistentMode:    c.PersistentMode,
		WakeWordEnabled:   c.WakeWordEnabled,
		WakeWordThreshold: c.WakeWordThreshold,
		SegmentSilenceMs:  c.SegmentSilenceMs,
	}
	if err := internal.Validate(); err != nil {
		return fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	v.s.setVoiceConfig(internal)
	if err := saveVoiceConfig(v.s.voiceConfigPath, internal); err != nil {
		log.Printf("voice-config: persist failed (in-memory updated): %v", err)
	}
	return nil
}

// ----- Wake word -----

func (v *voiceServiceShim) GetWakeWord() voiceH.WakeWordConfig {
	return wakeWordToTransport(v.s.getWakeWordTemplate())
}

func (v *voiceServiceShim) SetWakeWord(templateJSON string) (voiceH.WakeWordConfig, error) {
	var tmpl WakeWordTemplate
	if err := json.Unmarshal([]byte(templateJSON), &tmpl); err != nil {
		return voiceH.WakeWordConfig{}, fmt.Errorf("%w: parse template_json: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	if err := validateWakeWordTemplate(&tmpl); err != nil {
		return voiceH.WakeWordConfig{}, fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	v.s.setWakeWordTemplate(&tmpl)
	if err := saveWakeWordTemplate(v.s.wakeWordTemplatePath, &tmpl); err != nil {
		log.Printf("wakeword: persist failed (in-memory updated): %v", err)
	}
	log.Printf("wakeword: template saved: label=%q samples=%d threshold=%.2f",
		tmpl.Label, len(tmpl.Samples), tmpl.Threshold)
	return wakeWordToTransport(&tmpl), nil
}

func (v *voiceServiceShim) ClearWakeWord() error {
	v.s.setWakeWordTemplate(nil)
	return deleteWakeWordTemplate(v.s.wakeWordTemplatePath)
}

// ----- Speaker config -----

func (v *voiceServiceShim) GetSpeakerConfig() voiceH.SpeakerConfig {
	return speakerConfigToTransport(v.s.getSpeakerVerificationConfig())
}

func (v *voiceServiceShim) SaveSpeakerConfig(c voiceH.SpeakerConfig) error {
	internal := SpeakerVerificationConfig{
		Enabled:                     c.Enabled,
		ProfileIDs:                  append([]string(nil), c.ProfileIDs...),
		Threshold:                   c.Threshold,
		Mode:                        c.Mode,
		RejectBehavior:              c.RejectBehavior,
		FallbackWithoutVerification: c.FallbackWithoutVerification,
		ExtractionEnabled:           c.ExtractionEnabled,
	}
	if err := internal.Validate(); err != nil {
		return fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	v.s.setSpeakerVerificationConfig(internal)
	if err := saveSpeakerVerificationConfig(v.s.speakerVerificationConfigPath, internal); err != nil {
		log.Printf("speaker-verification-config: persist failed (in-memory updated): %v", err)
	}
	return nil
}

func (v *voiceServiceShim) DefaultSpeakerThreshold() float64 {
	return DefaultSpeakerVerificationConfig().Threshold
}

func (v *voiceServiceShim) DefaultSpeakerProfileID() string {
	return defaultSpeakerVerificationProfileID()
}

// ----- Speaker resource client -----

func (v *voiceServiceShim) SpeakerClientConfigured() bool { return v.s.speakerVerification != nil }

func (v *voiceServiceShim) SpeakerReady(ctx context.Context) bool {
	if v.s.speakerVerification == nil {
		return false
	}
	ready, err := v.s.speakerVerification.Ready(ctx)
	return err == nil && ready.Status == "ready"
}

func (v *voiceServiceShim) ListSpeakerProfiles(ctx context.Context) ([]voiceH.SpeakerProfile, int, error) {
	list, err := v.s.speakerVerification.ListProfiles(ctx)
	if err != nil {
		return nil, 0, err
	}
	return profilesToTransport(list.Profiles), list.Count, nil
}

func (v *voiceServiceShim) SpeakerInfo(ctx context.Context) (voiceH.SpeakerResourceInfo, bool) {
	if v.s.speakerVerification == nil {
		return voiceH.SpeakerResourceInfo{}, false
	}
	info, err := v.s.speakerVerification.Info(ctx)
	if err != nil {
		return voiceH.SpeakerResourceInfo{}, false
	}
	return voiceH.SpeakerResourceInfo{
		Backend:      info.Backend,
		Model:        info.Model,
		Device:       info.Device,
		SampleRate:   info.SampleRate,
		Version:      info.Version,
		EmbeddingDim: info.EmbeddingDim,
	}, true
}

func (v *voiceServiceShim) EnrollSpeaker(ctx context.Context, audio []byte, profileID, displayName, notes string) (voiceH.SpeakerEnrollment, error) {
	enrollment, err := v.s.speakerVerification.Enroll(ctx, audio, profileID, displayName, notes)
	if err != nil {
		return voiceH.SpeakerEnrollment{}, err
	}
	return voiceH.SpeakerEnrollment{
		ProfileID:              enrollment.ProfileID,
		DisplayName:            enrollment.DisplayName,
		EmbeddingDim:           enrollment.EmbeddingDim,
		SampleRate:             enrollment.SampleRate,
		EnrollmentAudioSeconds: enrollment.EnrollmentAudioSeconds,
		ModelName:              enrollment.ModelName,
		CreatedAt:              enrollment.CreatedAt,
	}, nil
}

func (v *voiceServiceShim) DeleteSpeakerBackend(ctx context.Context, profileID string) error {
	return v.s.speakerVerification.DeleteProfile(ctx, profileID)
}

// ----- mappers -----

func streamConfigToTransport(c VoiceStreamConfig) voiceH.StreamConfig {
	return voiceH.StreamConfig{
		FlushIntervalMs:   c.FlushIntervalMs,
		MinDeltaBytes:     c.MinDeltaBytes,
		OverlapBytes:      c.OverlapBytes,
		PersistentMode:    c.PersistentMode,
		WakeWordEnabled:   c.WakeWordEnabled,
		WakeWordThreshold: c.WakeWordThreshold,
		SegmentSilenceMs:  c.SegmentSilenceMs,
	}
}

func wakeWordToTransport(tmpl *WakeWordTemplate) voiceH.WakeWordConfig {
	if tmpl == nil {
		return voiceH.WakeWordConfig{Configured: false}
	}
	data, err := json.Marshal(tmpl)
	if err != nil {
		log.Printf("wakeword: marshal failed: %v", err)
		return voiceH.WakeWordConfig{Configured: true}
	}
	return voiceH.WakeWordConfig{Configured: true, TemplateJSON: string(data)}
}

func speakerConfigToTransport(c SpeakerVerificationConfig) voiceH.SpeakerConfig {
	return voiceH.SpeakerConfig{
		Enabled:                     c.Enabled,
		ProfileIDs:                  append([]string(nil), c.ProfileIDs...),
		Threshold:                   c.Threshold,
		Mode:                        c.Mode,
		RejectBehavior:              c.RejectBehavior,
		FallbackWithoutVerification: c.FallbackWithoutVerification,
		ExtractionEnabled:           c.ExtractionEnabled,
	}
}

func profilesToTransport(in []SpeakerVerificationProfile) []voiceH.SpeakerProfile {
	out := make([]voiceH.SpeakerProfile, 0, len(in))
	for _, p := range in {
		out = append(out, voiceH.SpeakerProfile{
			ID:                     p.ID,
			DisplayName:            p.DisplayName,
			CreatedAt:              p.CreatedAt,
			UpdatedAt:              p.UpdatedAt,
			ModelName:              p.ModelName,
			EmbeddingDim:           p.EmbeddingDim,
			SampleRate:             p.SampleRate,
			EnrollmentAudioSeconds: p.EnrollmentAudioSeconds,
			Notes:                  p.Notes,
		})
	}
	return out
}
