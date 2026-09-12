// Package speaker adapts the speaker-verification resource to the STT ingress
// and egress seams. It is intentionally separate from pipeline so neither
// package needs to depend on the other's downstream stream graph.
package speaker

import (
	"context"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/logx"
	"audio-tools/internal/stt/egress"
	"audio-tools/internal/stt/ingress"
	sttpipeline "audio-tools/internal/stt/pipeline"
)

type verification struct {
	cfg    sttpipeline.SpeakerConfig
	client *sttpipeline.SpeakerClient
	state  *sttpipeline.SessionSpeakerState
	logger logx.Logger
}

// NewIsolation builds one stateful verification stage per STT session.
func NewIsolation(cfg sttpipeline.SpeakerConfig, client *sttpipeline.SpeakerClient, logger logx.Logger) egress.SpeakerIsolation {
	if !cfg.Enabled || cfg.Mode == "off" {
		return nil
	}
	return &verification{cfg: cfg, client: client, state: sttpipeline.NewSessionSpeakerState(cfg), logger: logger}
}

func (s *verification) Evaluate(ctx context.Context, audio []byte) egress.SpeakerVerdict {
	decision := sttpipeline.EvaluateSpeaker(ctx, s.cfg, s.client, audio)
	allowed, smoothed, reason := s.state.Observe(decision)
	score := smoothed
	if !s.state.HasEvidence() {
		score = decision.Score
	}
	verdict := egress.SpeakerVerdict{Allowed: allowed, Score: score, Threshold: s.cfg.Threshold}
	if !allowed {
		if reason == "" {
			reason = sttpipeline.FormatSpeakerDecisionError(decision)
		}
		verdict.Reason = reason
	}
	if allowed && decision.Enabled && !decision.Applied {
		verdict.FallbackUsed = true
	}
	if s.logger != nil {
		s.logger.Printf("speaker-verify: allowed=%t applied=%t sufficient=%t voiced=%.2fs raw=%.3f smoothed=%.3f thr=%.2f mode=%s reason=%q", allowed, decision.Applied, decision.Sufficient, decision.VoicedSeconds, decision.Score, score, s.cfg.Threshold, s.cfg.Mode, verdict.Reason)
	}
	return verdict
}

func (s *verification) AllowMissingAudio() bool {
	return s.cfg.Mode == "advisory" || s.cfg.FallbackWithoutVerification
}

type extraction struct {
	cfg    sttpipeline.SpeakerConfig
	client *sttpipeline.SpeakerClient
}

// NewExtraction builds the target-speaker ingress stage for one session.
func NewExtraction(cfg sttpipeline.SpeakerConfig, client *sttpipeline.SpeakerClient) ingress.TargetExtractor {
	if !cfg.Enabled || cfg.Mode == "off" || !cfg.ExtractionEnabled || len(cfg.ProfileIDs) == 0 || client == nil {
		return nil
	}
	return extraction{cfg: sttpipeline.SpeakerConfig{ProfileIDs: cfg.ProfileIDs, Threshold: cfg.Threshold, Mode: cfg.Mode}, client: client}
}

func (s extraction) Extract(ctx context.Context, pcm []byte) ([]byte, error) {
	if s.client == nil || len(s.cfg.ProfileIDs) == 0 {
		return pcm, nil
	}
	wav := audioformat.WAVFromCanonicalPCM(pcm)
	var best []byte
	var bestScore float64
	found := false
	for _, profileID := range s.cfg.ProfileIDs {
		result, err := s.client.Extract(ctx, wav, profileID, true)
		if err != nil {
			continue
		}
		if result.Matched {
			return result.Audio, nil
		}
		if !found || result.Score > bestScore {
			bestScore, best, found = result.Score, result.Audio, true
		}
	}
	if !found || len(best) == 0 {
		return pcm, nil
	}
	return best, nil
}
