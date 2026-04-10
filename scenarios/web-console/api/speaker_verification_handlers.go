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
	AddToActive bool   `json:"addToActive"`
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
	}
	if !decision.Enabled {
		return decision
	}
	if len(cfg.ProfileIDs) == 0 {
		decision.Allowed = cfg.FallbackWithoutVerification
		decision.ErrorMessage = "no speaker profiles configured"
		return decision
	}
	if s.speakerVerification == nil {
		decision.Allowed = cfg.FallbackWithoutVerification
		decision.ErrorMessage = "speaker verification resource is not configured"
		return decision
	}

	// Try each profile — accept on first match (any-match strategy).
	var bestScore float64
	var bestProfileID string
	var lastErr error
	for _, profileID := range cfg.ProfileIDs {
		result, err := s.speakerVerification.Verify(ctx, audio, profileID, cfg.Threshold)
		if err != nil {
			lastErr = err
			continue
		}
		if result.Score > bestScore {
			bestScore = result.Score
			bestProfileID = result.ProfileID
			if bestProfileID == "" {
				bestProfileID = profileID
			}
		}
		if result.Matched {
			decision.Applied = true
			decision.Matched = true
			decision.Score = result.Score
			decision.Threshold = result.Threshold
			decision.ProfileID = bestProfileID
			decision.Allowed = true
			return decision
		}
	}

	// No profile matched — report the best score.
	if bestProfileID != "" {
		decision.Applied = true
		decision.Score = bestScore
		decision.ProfileID = bestProfileID
	} else if lastErr != nil {
		decision.ErrorMessage = lastErr.Error()
	}
	if cfg.Mode == "advisory" {
		decision.Allowed = true
		return decision
	}
	if !decision.Applied {
		decision.Allowed = cfg.FallbackWithoutVerification
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
		AddToActive: r.FormValue("addToActive") != "false",
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
	if req.AddToActive {
		if !containsString(cfg.ProfileIDs, req.ProfileID) {
			cfg.ProfileIDs = append(cfg.ProfileIDs, req.ProfileID)
		}
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
	cfg.ProfileIDs = nil
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

func (s *Server) handleRemoveSpeakerProfile(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProfileID string `json:"profileId"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.ProfileID == "" {
		writeCatalogError(w, "invalid_body", "profileId is required")
		return
	}

	cfg := s.getSpeakerVerificationConfig()
	cfg.ProfileIDs = removeString(cfg.ProfileIDs, body.ProfileID)
	if len(cfg.ProfileIDs) == 0 {
		cfg.Enabled = false
	}
	if err := cfg.Validate(); err != nil {
		writeCatalogError(w, "speaker_profile_remove_failed", err.Error())
		return
	}
	s.setSpeakerVerificationConfig(cfg)
	if err := saveSpeakerVerificationConfig(s.speakerVerificationConfigPath, cfg); err != nil {
		log.Printf("speaker-verification-config: persist failed after remove profile: %v", err)
	}
	writeJSON(w, http.StatusOK, cfg)
}

// handleDeleteSpeakerProfile deletes a profile from the speaker-verification
// resource AND removes it from the active config list.
func (s *Server) handleDeleteSpeakerProfile(w http.ResponseWriter, r *http.Request) {
	if s.speakerVerification == nil {
		writeCatalogError(w, "speaker_verification_unavailable", "Speaker verification resource is not configured")
		return
	}

	var body struct {
		ProfileID string `json:"profileId"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.ProfileID == "" {
		writeCatalogError(w, "invalid_body", "profileId is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	if err := s.speakerVerification.DeleteProfile(ctx, body.ProfileID); err != nil {
		log.Printf("speaker-verification-delete: %v", err)
		writeCatalogError(w, "speaker_delete_failed", "Failed to delete speaker profile from resource")
		return
	}

	// Also remove from the active config list.
	cfg := s.getSpeakerVerificationConfig()
	cfg.ProfileIDs = removeString(cfg.ProfileIDs, body.ProfileID)
	if len(cfg.ProfileIDs) == 0 {
		cfg.Enabled = false
	}
	if err := cfg.Validate(); err != nil {
		writeCatalogError(w, "speaker_delete_failed", err.Error())
		return
	}
	s.setSpeakerVerificationConfig(cfg)
	if err := saveSpeakerVerificationConfig(s.speakerVerificationConfigPath, cfg); err != nil {
		log.Printf("speaker-verification-config: persist failed after delete: %v", err)
	}
	writeJSON(w, http.StatusOK, cfg)
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func removeString(ss []string, s string) []string {
	result := make([]string, 0, len(ss))
	for _, v := range ss {
		if v != s {
			result = append(result, v)
		}
	}
	return result
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
	if len(cfg.ProfileIDs) == 0 {
		decision := s.evaluateSpeakerVerification(ctx, audio)
		return audio, decision
	}
	if s.speakerVerification == nil {
		decision := s.evaluateSpeakerVerification(ctx, audio)
		return audio, decision
	}

	// Try extraction against each profile — accept on first match.
	extractCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var bestScore float64
	var bestProfileID string
	var bestAudio []byte
	for _, profileID := range cfg.ProfileIDs {
		result, err := s.speakerVerification.Extract(extractCtx, audio, profileID, true)
		if err != nil {
			log.Printf("speaker-extraction: profile %s failed: %v", profileID, err)
			continue
		}
		if result.Score > bestScore {
			bestScore = result.Score
			bestProfileID = profileID
			bestAudio = result.Audio
		}
		if result.Matched {
			return result.Audio, speakerVerificationGateDecision{
				Enabled:   true,
				Applied:   true,
				Matched:   true,
				Score:     result.Score,
				Threshold: cfg.Threshold,
				ProfileID: profileID,
				Mode:      cfg.Mode,
				Extracted: true,
				Allowed:   true,
			}
		}
	}

	// No profile matched via extraction. If we got at least one result,
	// report the best score; otherwise fall back to verify-only.
	if bestProfileID == "" {
		log.Printf("speaker-extraction: all profiles failed, falling back to verify-only")
		decision := s.evaluateSpeakerVerification(ctx, audio)
		return audio, decision
	}

	decision := speakerVerificationGateDecision{
		Enabled:   true,
		Applied:   true,
		Matched:   false,
		Score:     bestScore,
		Threshold: cfg.Threshold,
		ProfileID: bestProfileID,
		Mode:      cfg.Mode,
		Extracted: true,
	}
	if cfg.Mode == "advisory" {
		decision.Allowed = true
		return bestAudio, decision
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
