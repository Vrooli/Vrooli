package pipeline

import (
	"context"
	"fmt"

	"audio-tools/internal/audioformat"
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

	// Sufficient is false when the resource judged the segment to carry too
	// little voiced audio to verify. VoicedSeconds is the voiced duration the
	// resource measured (after VAD trim). An insufficient segment is never a
	// rejection — it is undetermined evidence the session verifier ignores.
	Sufficient    bool
	VoicedSeconds float64
}

// EvaluateSpeaker runs the configured profiles against the supplied audio
// and returns the gate decision. client may be nil — the decision uses
// FallbackWithoutVerification semantics in that case.
func EvaluateSpeaker(ctx context.Context, cfg SpeakerConfig, client *SpeakerClient, audio []byte) SpeakerDecision {
	decision := SpeakerDecision{
		Enabled:    cfg.Enabled && cfg.Mode != "off",
		Allowed:    true,
		Threshold:  cfg.Threshold,
		Mode:       cfg.Mode,
		Sufficient: true,
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

	// The egress speaker stage holds raw canonical PCM; the resource decodes
	// via container sniffing and cannot read headerless PCM. Wrap it in a WAV
	// header so verification can actually run.
	verifyAudio := audioformat.WAVFromCanonicalPCM(audio)

	var bestScore float64
	var bestProfileID string
	var lastErr error
	for _, profileID := range cfg.ProfileIDs {
		result, err := client.Verify(ctx, verifyAudio, profileID, cfg.Threshold)
		if err != nil {
			lastErr = err
			continue
		}
		// Sufficiency is a property of the segment audio (identical voiced span
		// across profiles), so the first result settles it: too little voiced
		// audio to judge → undetermined, never a rejection. The session verifier
		// treats this as warm-up passthrough.
		if !result.Sufficient {
			decision.Sufficient = false
			decision.VoicedSeconds = result.VoicedSeconds
			decision.Applied = false
			decision.Allowed = true
			return decision
		}
		decision.VoicedSeconds = result.VoicedSeconds
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

func FormatSpeakerDecisionError(decision SpeakerDecision) string {
	if decision.ErrorMessage == "" {
		return "speaker verification failed"
	}
	return fmt.Sprintf("speaker verification failed: %s", decision.ErrorMessage)
}
