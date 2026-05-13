package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

const maxSpeakerEnrollmentAudioSize = 20 << 20 // 20 MB

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

// evaluateSpeakerVerification runs the configured profiles against the
// supplied audio and returns the gate decision. Used by the voice stream
// transcription path and (via the Connect voice adapter) the
// VoiceService.Transcribe RPC.
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
