package pipeline

import (
	"context"
	"fmt"
	"log"
	"time"
)

// SpeakerDecision holds the outcome of running speaker verification against
// an audio sample. The transcription pipeline gates final output on the
// Allowed flag; UI clients learn from Applied/Matched whether verification
// actually ran and what the score was.
type SpeakerDecision struct {
	Enabled      bool
	Applied      bool
	Allowed      bool
	Matched      bool
	ProfileID    string
	Score        float64
	Threshold    float64
	Mode         string
	ErrorMessage string
	Extracted    bool
}

// EvaluateSpeaker runs the configured profiles against the supplied audio
// and returns the gate decision. client may be nil — the decision uses
// FallbackWithoutVerification semantics in that case.
func EvaluateSpeaker(ctx context.Context, cfg SpeakerConfig, client *SpeakerClient, audio []byte) SpeakerDecision {
	decision := SpeakerDecision{
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
	if client == nil {
		decision.Allowed = cfg.FallbackWithoutVerification
		decision.ErrorMessage = "speaker verification resource is not configured"
		return decision
	}

	var bestScore float64
	var bestProfileID string
	var lastErr error
	for _, profileID := range cfg.ProfileIDs {
		result, err := client.Verify(ctx, audio, profileID, cfg.Threshold)
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

// ExtractTargetSpeaker attempts to isolate the enrolled speaker's voice from
// the audio mixture using Target Speaker Extraction (TSE). Returns the
// cleaned audio and a gate decision. Falls back to verification-only when
// extraction is disabled or unavailable.
//
// DOC: docs/internal/SEAMS.md#extract-target-speaker-seam
func ExtractTargetSpeaker(ctx context.Context, cfg SpeakerConfig, client *SpeakerClient, audio []byte) ([]byte, SpeakerDecision) {
	if !cfg.ExtractionEnabled || !cfg.Enabled || cfg.Mode == "off" {
		return audio, EvaluateSpeaker(ctx, cfg, client, audio)
	}
	if len(cfg.ProfileIDs) == 0 || client == nil {
		return audio, EvaluateSpeaker(ctx, cfg, client, audio)
	}

	extractCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var bestScore float64
	var bestProfileID string
	var bestAudio []byte
	for _, profileID := range cfg.ProfileIDs {
		result, err := client.Extract(extractCtx, audio, profileID, true)
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
			return result.Audio, SpeakerDecision{
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

	if bestProfileID == "" {
		log.Printf("speaker-extraction: all profiles failed, falling back to verify-only")
		return audio, EvaluateSpeaker(ctx, cfg, client, audio)
	}

	decision := SpeakerDecision{
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

func FormatSpeakerDecisionError(decision SpeakerDecision) string {
	if decision.ErrorMessage == "" {
		return "speaker verification failed"
	}
	return fmt.Sprintf("speaker verification failed: %s", decision.ErrorMessage)
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
