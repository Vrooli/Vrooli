package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const maxSpeakerEnrollmentAudioSize = 20 << 20 // 20 MB

type SpeakerVerificationProfilesResponse struct {
	Profiles []SpeakerVerificationProfile `json:"profiles"`
	Count    int                          `json:"count"`
}

type SpeakerVerificationEnrollmentRequest struct {
	ProfileID   string `json:"profileId"`
	DisplayName string `json:"displayName"`
	Notes       string `json:"notes"`
	SetActive   bool   `json:"setActive"`
	Enable      bool   `json:"enable"`
}

type SpeakerVerificationEnrollmentResult struct {
	Enrollment SpeakerVerificationEnrollmentResponse `json:"enrollment"`
	Config     SpeakerVerificationConfig             `json:"config"`
}

type speakerVerificationGateDecision struct {
	Enabled      bool
	Applied      bool
	Allowed      bool
	Matched      bool
	ProfileID    string
	Score        float64
	Threshold    float64
	Mode         string
	ErrorMessage string
	Extracted    bool // True when TSE was used to isolate the speaker
}

func defaultSpeakerVerificationProfileID() string {
	return "default"
}

func (s *Server) evaluateSpeakerVerification(ctx context.Context, audio []byte) speakerVerificationGateDecision {
	cfg := s.getSpeakerVerificationConfig()
	decision := speakerVerificationGateDecision{
		Enabled:   cfg.Enabled && cfg.Mode != "off",
		Allowed:   true,
		Threshold: cfg.Threshold,
		Mode:      cfg.Mode,
		ProfileID: cfg.ProfileID,
	}
	if !decision.Enabled {
		return decision
	}
	if cfg.ProfileID == "" {
		decision.Allowed = cfg.FallbackWithoutVerification
		decision.ErrorMessage = "speaker profile is not configured"
		return decision
	}
	if s.speakerVerification == nil {
		decision.Allowed = cfg.FallbackWithoutVerification
		decision.ErrorMessage = "speaker verification resource is not configured"
		return decision
	}

	result, err := s.speakerVerification.Verify(ctx, audio, cfg.ProfileID, cfg.Threshold)
	if err != nil {
		decision.Allowed = cfg.FallbackWithoutVerification
		decision.ErrorMessage = err.Error()
		return decision
	}

	decision.Applied = true
	decision.Matched = result.Matched
	decision.Score = result.Score
	decision.Threshold = result.Threshold
	if result.ProfileID != "" {
		decision.ProfileID = result.ProfileID
	}
	if result.Matched || cfg.Mode == "advisory" {
		decision.Allowed = true
		return decision
	}
	decision.Allowed = false
	return decision
}

func (s *Server) handleGetSpeakerVerificationProfiles(w http.ResponseWriter, r *http.Request) {
	if s.speakerVerification == nil {
		writeCatalogError(w, "speaker_verification_unavailable", "Speaker verification resource is not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	profiles, err := s.speakerVerification.ListProfiles(ctx)
	if err != nil {
		writeCatalogError(w, "speaker_verification_failed", "Failed to list speaker profiles")
		return
	}
	writeJSON(w, http.StatusOK, SpeakerVerificationProfilesResponse(profiles))
}

func (s *Server) handleEnrollSpeakerProfile(w http.ResponseWriter, r *http.Request) {
	if s.speakerVerification == nil {
		writeCatalogError(w, "speaker_verification_unavailable", "Speaker verification resource is not configured")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxSpeakerEnrollmentAudioSize)
	if err := r.ParseMultipartForm(maxSpeakerEnrollmentAudioSize); err != nil {
		writeCatalogError(w, "invalid_body", "Failed to parse multipart form: "+err.Error())
		return
	}

	file, _, err := r.FormFile("audio_file")
	if err != nil {
		writeCatalogError(w, "invalid_body", "Missing audio_file field")
		return
	}
	defer file.Close()

	audioBytes, err := io.ReadAll(file)
	if err != nil || len(audioBytes) == 0 {
		writeCatalogError(w, "invalid_body", "Enrollment audio could not be read")
		return
	}

	req := SpeakerVerificationEnrollmentRequest{
		ProfileID:   strings.TrimSpace(r.FormValue("profileId")),
		DisplayName: strings.TrimSpace(r.FormValue("displayName")),
		Notes:       strings.TrimSpace(r.FormValue("notes")),
		SetActive:   r.FormValue("setActive") != "false",
		Enable:      r.FormValue("enable") != "false",
	}
	if req.ProfileID == "" {
		req.ProfileID = defaultSpeakerVerificationProfileID()
	}
	if req.DisplayName == "" {
		req.DisplayName = "My Voice"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	enrollment, err := s.speakerVerification.Enroll(ctx, audioBytes, req.ProfileID, req.DisplayName, req.Notes)
	if err != nil {
		log.Printf("speaker-verification-enroll: %v", err)
		writeCatalogError(w, "speaker_enrollment_failed", "Failed to enroll speaker profile")
		return
	}

	cfg := s.getSpeakerVerificationConfig()
	if req.SetActive {
		cfg.ProfileID = req.ProfileID
	}
	if req.Enable {
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
		writeCatalogError(w, "speaker_enrollment_failed", "Enrollment succeeded, but speaker verification config is invalid")
		return
	}
	s.setSpeakerVerificationConfig(cfg)
	if err := saveSpeakerVerificationConfig(s.speakerVerificationConfigPath, cfg); err != nil {
		log.Printf("speaker-verification-config: persist failed after enrollment: %v", err)
	}

	writeJSON(w, http.StatusOK, SpeakerVerificationEnrollmentResult{
		Enrollment: enrollment,
		Config:     cfg,
	})
}

func (s *Server) handleClearSpeakerProfileBinding(w http.ResponseWriter, _ *http.Request) {
	cfg := s.getSpeakerVerificationConfig()
	cfg.Enabled = false
	cfg.ProfileID = ""
	if err := cfg.Validate(); err != nil {
		writeCatalogError(w, "speaker_profile_clear_failed", err.Error())
		return
	}
	s.setSpeakerVerificationConfig(cfg)
	if err := saveSpeakerVerificationConfig(s.speakerVerificationConfigPath, cfg); err != nil {
		log.Printf("speaker-verification-config: persist failed after clear binding: %v", err)
	}
	writeJSON(w, http.StatusOK, cfg)
}

// extractTargetSpeaker attempts to isolate the enrolled speaker's voice from
// the audio mixture using Target Speaker Extraction (TSE). It returns the
// cleaned audio and a gate decision.
//
// Fallback chain:
//   - TSE enabled + resource available → extract + verify extracted audio
//   - TSE enabled + resource error    → fall back to verify-only on original audio
//   - TSE disabled                    → verify-only on original audio (current behavior)
//
// DOC: docs/internal/SEAMS.md#extract-target-speaker-seam
func (s *Server) extractTargetSpeaker(ctx context.Context, audio []byte) ([]byte, speakerVerificationGateDecision) {
	cfg := s.getSpeakerVerificationConfig()

	// If TSE is not enabled or not configured, fall back to verification-only.
	if !cfg.ExtractionEnabled || !cfg.Enabled || cfg.Mode == "off" {
		decision := s.evaluateSpeakerVerification(ctx, audio)
		return audio, decision
	}
	if cfg.ProfileID == "" {
		decision := s.evaluateSpeakerVerification(ctx, audio)
		return audio, decision
	}
	if s.speakerVerification == nil {
		decision := s.evaluateSpeakerVerification(ctx, audio)
		return audio, decision
	}

	// Call the extraction endpoint with a generous timeout.
	extractCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	result, err := s.speakerVerification.Extract(extractCtx, audio, cfg.ProfileID, true)
	if err != nil {
		// TSE failed — fall back to verification-only on original audio.
		log.Printf("speaker-extraction: failed, falling back to verify-only: %v", err)
		decision := s.evaluateSpeakerVerification(ctx, audio)
		return audio, decision
	}

	decision := speakerVerificationGateDecision{
		Enabled:   true,
		Applied:   true,
		Matched:   result.Matched,
		Score:     result.Score,
		Threshold: cfg.Threshold,
		ProfileID: cfg.ProfileID,
		Mode:      cfg.Mode,
		Extracted: true,
	}

	if result.Matched || cfg.Mode == "advisory" {
		decision.Allowed = true
		return result.Audio, decision
	}

	decision.Allowed = false
	return nil, decision
}

func formatSpeakerDecisionError(decision speakerVerificationGateDecision) string {
	if decision.ErrorMessage == "" {
		return "speaker verification failed"
	}
	return fmt.Sprintf("speaker verification failed: %s", decision.ErrorMessage)
}
