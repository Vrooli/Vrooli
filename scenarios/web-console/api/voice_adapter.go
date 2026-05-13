package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	voiceH "web-console/handlers/voice"
)

// newVoiceAdapter constructs the voice.Service implementation backed by the
// existing Server fields (voice config store, Whisper transcribe helpers,
// wake-word store, speaker-verification config/client).
func newVoiceAdapter(s *Server) voiceH.Service {
	return &voiceAdapter{s: s}
}

type voiceAdapter struct {
	s *Server
}

// -----------------------------------------------------------------------------
// Transcribe
// -----------------------------------------------------------------------------

func (a *voiceAdapter) Transcribe(ctx context.Context, in voiceH.TranscribeInput) (string, error) {
	if !a.s.capabilities.IsAvailable(ctx, "whisper-stt") {
		return "", fmt.Errorf("%w: whisper transcription is currently unavailable", voiceH.ErrUnavailable)
	}
	if len(in.Audio) == 0 {
		return "", fmt.Errorf("%w: audio is required", voiceH.ErrInvalidArgument)
	}
	if len(in.Audio) > maxAudioSize {
		return "", fmt.Errorf("%w: audio exceeds %d bytes", voiceH.ErrInvalidArgument, maxAudioSize)
	}

	if in.SkipSpeakerVerification {
		if a.s.metrics != nil {
			a.s.metrics.VoiceSkipVerificationTotal.Add(1)
		}
		log.Printf("voice-http: speaker verification bypassed bytes=%d", len(in.Audio))
	} else {
		verifyCtx, verifyCancel := context.WithTimeout(ctx, 10*time.Second)
		decision := a.s.evaluateSpeakerVerification(verifyCtx, in.Audio)
		verifyCancel()
		if decision.Enabled {
			if decision.Applied {
				log.Printf(
					"voice-http: speaker decision matched=%v allowed=%v score=%.3f threshold=%.3f profile=%s mode=%s",
					decision.Matched, decision.Allowed, decision.Score, decision.Threshold,
					decision.ProfileID, decision.Mode,
				)
			} else if decision.ErrorMessage != "" {
				log.Printf("voice-http: %s", formatSpeakerDecisionError(decision))
			}
			if !decision.Allowed {
				return "", nil
			}
		}
	}

	text, err := a.s.transcribeBytes(ctx, in.Audio, in.Language, true, "")
	if err != nil {
		log.Printf("voice-http: whisper failed: %v", err)
		return "", fmt.Errorf("%w: whisper request failed", voiceH.ErrInternal)
	}
	if isWhisperHallucination(text) {
		log.Printf("voice-http: filtered hallucination: %q", text)
		text = ""
	}
	return text, nil
}

// -----------------------------------------------------------------------------
// Stream config
// -----------------------------------------------------------------------------

func (a *voiceAdapter) GetStreamConfig(_ context.Context) (voiceH.StreamConfig, error) {
	return streamConfigToHandler(a.s.getVoiceConfig()), nil
}

func (a *voiceAdapter) UpdateStreamConfig(_ context.Context, patch voiceH.StreamConfigPatch) (voiceH.StreamConfig, error) {
	current := a.s.getVoiceConfig()
	if patch.FlushIntervalMs != nil {
		current.FlushIntervalMs = *patch.FlushIntervalMs
	}
	if patch.MinDeltaBytes != nil {
		current.MinDeltaBytes = *patch.MinDeltaBytes
	}
	if patch.OverlapBytes != nil {
		current.OverlapBytes = *patch.OverlapBytes
	}
	if patch.PersistentMode != nil {
		current.PersistentMode = *patch.PersistentMode
	}
	if patch.WakeWordEnabled != nil {
		current.WakeWordEnabled = *patch.WakeWordEnabled
	}
	if patch.WakeWordThreshold != nil {
		current.WakeWordThreshold = *patch.WakeWordThreshold
	}
	if patch.SegmentSilenceMs != nil {
		current.SegmentSilenceMs = *patch.SegmentSilenceMs
	}
	if err := current.Validate(); err != nil {
		return voiceH.StreamConfig{}, fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	a.s.setVoiceConfig(current)
	if err := saveVoiceConfig(a.s.voiceConfigPath, current); err != nil {
		log.Printf("voice-config: persist failed (in-memory updated): %v", err)
	}
	log.Printf("voice-config: updated: flush=%dms delta=%d overlap=%d",
		current.FlushIntervalMs, current.MinDeltaBytes, current.OverlapBytes)
	return streamConfigToHandler(current), nil
}

func streamConfigToHandler(c VoiceStreamConfig) voiceH.StreamConfig {
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

// -----------------------------------------------------------------------------
// Wake word
// -----------------------------------------------------------------------------

func (a *voiceAdapter) GetWakeWordConfig(_ context.Context) (voiceH.WakeWordConfig, error) {
	tmpl := a.s.getWakeWordTemplate()
	return wakeWordToHandler(tmpl), nil
}

func (a *voiceAdapter) UpdateWakeWordTemplate(_ context.Context, templateJSON string) (voiceH.WakeWordConfig, error) {
	var tmpl WakeWordTemplate
	if err := json.Unmarshal([]byte(templateJSON), &tmpl); err != nil {
		return voiceH.WakeWordConfig{}, fmt.Errorf("%w: parse template_json: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	if err := validateWakeWordTemplate(&tmpl); err != nil {
		return voiceH.WakeWordConfig{}, fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	a.s.setWakeWordTemplate(&tmpl)
	if err := saveWakeWordTemplate(a.s.wakeWordTemplatePath, &tmpl); err != nil {
		log.Printf("wakeword: persist failed (in-memory updated): %v", err)
	}
	log.Printf("wakeword: template saved: label=%q samples=%d threshold=%.2f",
		tmpl.Label, len(tmpl.Samples), tmpl.Threshold)
	return wakeWordToHandler(&tmpl), nil
}

func (a *voiceAdapter) DeleteWakeWordTemplate(_ context.Context) (voiceH.WakeWordConfig, error) {
	a.s.setWakeWordTemplate(nil)
	if err := deleteWakeWordTemplate(a.s.wakeWordTemplatePath); err != nil {
		log.Printf("wakeword: delete failed: %v", err)
	}
	log.Printf("wakeword: template cleared")
	return voiceH.WakeWordConfig{Configured: false}, nil
}

func wakeWordToHandler(tmpl *WakeWordTemplate) voiceH.WakeWordConfig {
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

// -----------------------------------------------------------------------------
// Speaker config
// -----------------------------------------------------------------------------

func (a *voiceAdapter) GetSpeakerConfig(_ context.Context) (voiceH.SpeakerConfig, error) {
	return speakerConfigToHandler(a.s.getSpeakerVerificationConfig()), nil
}

func (a *voiceAdapter) UpdateSpeakerConfig(_ context.Context, patch voiceH.SpeakerConfigPatch) (voiceH.SpeakerConfig, error) {
	current := a.s.getSpeakerVerificationConfig()
	if patch.Enabled != nil {
		current.Enabled = *patch.Enabled
	}
	if patch.ProfileIDs != nil {
		current.ProfileIDs = append([]string(nil), (*patch.ProfileIDs)...)
	}
	if patch.Threshold != nil {
		current.Threshold = *patch.Threshold
	}
	if patch.Mode != nil {
		current.Mode = *patch.Mode
	}
	if patch.RejectBehavior != nil {
		current.RejectBehavior = *patch.RejectBehavior
	}
	if patch.FallbackWithoutVerification != nil {
		current.FallbackWithoutVerification = *patch.FallbackWithoutVerification
	}
	if patch.ExtractionEnabled != nil {
		current.ExtractionEnabled = *patch.ExtractionEnabled
	}
	if current.Mode == "" {
		current.Mode = "filter"
	}
	if current.RejectBehavior == "" {
		current.RejectBehavior = "drop"
	}
	if err := current.Validate(); err != nil {
		return voiceH.SpeakerConfig{}, fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	a.s.setSpeakerVerificationConfig(current)
	if err := saveSpeakerVerificationConfig(a.s.speakerVerificationConfigPath, current); err != nil {
		log.Printf("speaker-verification-config: persist failed (in-memory updated): %v", err)
	}
	return speakerConfigToHandler(current), nil
}

func (a *voiceAdapter) GetSpeakerStatus(ctx context.Context) (voiceH.SpeakerStatus, error) {
	cfg := a.s.getSpeakerVerificationConfig()
	out := voiceH.SpeakerStatus{
		Config:            speakerConfigToHandler(cfg),
		Capability:        string(StatusUnknown),
		ProfileConfigured: len(cfg.ProfileIDs) > 0,
		CheckedAt:         time.Now().UTC().Format(time.RFC3339),
	}

	probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	for _, cap := range a.s.capabilities.ResolveLiveness(probeCtx) {
		if cap.ID != "speaker-verification" {
			continue
		}
		out.Capability = string(cap.Status)
		out.CapabilityLabel = cap.Message
		break
	}

	if a.s.speakerVerification == nil || out.Capability != string(StatusAvailable) {
		return out, nil
	}

	ready, err := a.s.speakerVerification.Ready(probeCtx)
	if err == nil && ready.Status == "ready" {
		out.ResourceReady = true
	}

	profiles, err := a.s.speakerVerification.ListProfiles(probeCtx)
	if err == nil {
		out.ProfileCount = profiles.Count
		out.Profiles = profilesToHandler(profiles.Profiles)
		configuredSet := make(map[string]struct{}, len(cfg.ProfileIDs))
		for _, id := range cfg.ProfileIDs {
			configuredSet[id] = struct{}{}
		}
		for _, profile := range profiles.Profiles {
			if _, ok := configuredSet[profile.ID]; ok {
				out.ProfileExists = true
				break
			}
		}
	}

	info, err := a.s.speakerVerification.Info(probeCtx)
	if err == nil {
		out.Info = &voiceH.SpeakerResourceInfo{
			Backend:      info.Backend,
			Model:        info.Model,
			Device:       info.Device,
			SampleRate:   info.SampleRate,
			Version:      info.Version,
			EmbeddingDim: info.EmbeddingDim,
		}
	}
	return out, nil
}

func (a *voiceAdapter) ListSpeakerProfiles(ctx context.Context) ([]voiceH.SpeakerProfile, int, error) {
	if a.s.speakerVerification == nil {
		return nil, 0, fmt.Errorf("%w: speaker verification resource is not configured", voiceH.ErrUnavailable)
	}
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	list, err := a.s.speakerVerification.ListProfiles(probeCtx)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: list speaker profiles: %s", voiceH.ErrInternal, err.Error())
	}
	return profilesToHandler(list.Profiles), list.Count, nil
}

func (a *voiceAdapter) EnrollSpeakerProfile(ctx context.Context, in voiceH.EnrollInput) (voiceH.SpeakerEnrollment, voiceH.SpeakerConfig, error) {
	if a.s.speakerVerification == nil {
		return voiceH.SpeakerEnrollment{}, voiceH.SpeakerConfig{}, fmt.Errorf("%w: speaker verification resource is not configured", voiceH.ErrUnavailable)
	}
	if len(in.Audio) > maxSpeakerEnrollmentAudioSize {
		return voiceH.SpeakerEnrollment{}, voiceH.SpeakerConfig{}, fmt.Errorf("%w: audio exceeds %d bytes", voiceH.ErrInvalidArgument, maxSpeakerEnrollmentAudioSize)
	}

	profileID := in.ProfileID
	if profileID == "" {
		profileID = defaultSpeakerVerificationProfileID()
	}
	displayName := in.DisplayName
	if displayName == "" {
		displayName = "My Voice"
	}
	addToActive := true
	if in.AddToActive != nil {
		addToActive = *in.AddToActive
	}
	enable := true
	if in.Enable != nil {
		enable = *in.Enable
	}

	enrollCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	enrollment, err := a.s.speakerVerification.Enroll(enrollCtx, in.Audio, profileID, displayName, in.Notes)
	if err != nil {
		log.Printf("speaker-verification-enroll: %v", err)
		return voiceH.SpeakerEnrollment{}, voiceH.SpeakerConfig{}, fmt.Errorf("%w: failed to enroll speaker profile", voiceH.ErrInternal)
	}

	cfg := a.s.getSpeakerVerificationConfig()
	if addToActive {
		if !containsString(cfg.ProfileIDs, profileID) {
			cfg.ProfileIDs = append(cfg.ProfileIDs, profileID)
		}
	}
	if enable {
		cfg.Enabled = true
		if cfg.Mode == "" {
			cfg.Mode = "filter"
		}
		if cfg.RejectBehavior == "" {
			cfg.RejectBehavior = "drop"
		}
		if cfg.Threshold == 0 {
			cfg.Threshold = DefaultSpeakerVerificationConfig().Threshold
		}
	}
	if err := cfg.Validate(); err != nil {
		return voiceH.SpeakerEnrollment{}, voiceH.SpeakerConfig{}, fmt.Errorf("%w: enrollment succeeded, but speaker verification config is invalid", voiceH.ErrInternal)
	}
	a.s.setSpeakerVerificationConfig(cfg)
	if err := saveSpeakerVerificationConfig(a.s.speakerVerificationConfigPath, cfg); err != nil {
		log.Printf("speaker-verification-config: persist failed after enrollment: %v", err)
	}

	return voiceH.SpeakerEnrollment{
		ProfileID:              enrollment.ProfileID,
		DisplayName:            enrollment.DisplayName,
		EmbeddingDim:           enrollment.EmbeddingDim,
		SampleRate:             enrollment.SampleRate,
		EnrollmentAudioSeconds: enrollment.EnrollmentAudioSeconds,
		ModelName:              enrollment.ModelName,
		CreatedAt:              enrollment.CreatedAt,
	}, speakerConfigToHandler(cfg), nil
}

func (a *voiceAdapter) ClearSpeakerProfileBinding(_ context.Context) (voiceH.SpeakerConfig, error) {
	cfg := a.s.getSpeakerVerificationConfig()
	cfg.Enabled = false
	cfg.ProfileIDs = nil
	if err := cfg.Validate(); err != nil {
		return voiceH.SpeakerConfig{}, fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	a.s.setSpeakerVerificationConfig(cfg)
	if err := saveSpeakerVerificationConfig(a.s.speakerVerificationConfigPath, cfg); err != nil {
		log.Printf("speaker-verification-config: persist failed after clear binding: %v", err)
	}
	return speakerConfigToHandler(cfg), nil
}

func (a *voiceAdapter) RemoveSpeakerProfile(_ context.Context, profileID string) (voiceH.SpeakerConfig, error) {
	cfg := a.s.getSpeakerVerificationConfig()
	cfg.ProfileIDs = removeString(cfg.ProfileIDs, profileID)
	if len(cfg.ProfileIDs) == 0 {
		cfg.Enabled = false
	}
	if err := cfg.Validate(); err != nil {
		return voiceH.SpeakerConfig{}, fmt.Errorf("%w: %s", voiceH.ErrInvalidArgument, err.Error())
	}
	a.s.setSpeakerVerificationConfig(cfg)
	if err := saveSpeakerVerificationConfig(a.s.speakerVerificationConfigPath, cfg); err != nil {
		log.Printf("speaker-verification-config: persist failed after remove profile: %v", err)
	}
	return speakerConfigToHandler(cfg), nil
}

func (a *voiceAdapter) DeleteSpeakerProfile(ctx context.Context, profileID string) (voiceH.SpeakerConfig, error) {
	if a.s.speakerVerification == nil {
		return voiceH.SpeakerConfig{}, fmt.Errorf("%w: speaker verification resource is not configured", voiceH.ErrUnavailable)
	}
	delCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := a.s.speakerVerification.DeleteProfile(delCtx, profileID); err != nil {
		log.Printf("speaker-verification-delete: %v", err)
		return voiceH.SpeakerConfig{}, fmt.Errorf("%w: failed to delete speaker profile from resource", voiceH.ErrInternal)
	}
	cfg := a.s.getSpeakerVerificationConfig()
	cfg.ProfileIDs = removeString(cfg.ProfileIDs, profileID)
	if len(cfg.ProfileIDs) == 0 {
		cfg.Enabled = false
	}
	if err := cfg.Validate(); err != nil {
		return voiceH.SpeakerConfig{}, fmt.Errorf("%w: %s", voiceH.ErrInternal, err.Error())
	}
	a.s.setSpeakerVerificationConfig(cfg)
	if err := saveSpeakerVerificationConfig(a.s.speakerVerificationConfigPath, cfg); err != nil {
		log.Printf("speaker-verification-config: persist failed after delete: %v", err)
	}
	return speakerConfigToHandler(cfg), nil
}

// -----------------------------------------------------------------------------
// Mappers
// -----------------------------------------------------------------------------

func speakerConfigToHandler(c SpeakerVerificationConfig) voiceH.SpeakerConfig {
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

func profilesToHandler(in []SpeakerVerificationProfile) []voiceH.SpeakerProfile {
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
