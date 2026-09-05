// Package quality applies the post-recognition STT egress policy to any
// user-facing transcript, regardless of transport.
package quality

import (
	"context"

	"audio-tools/internal/ai/sttchain"
	sttpkg "audio-tools/internal/stt"
	"audio-tools/internal/stt/egress"
	voice "audio-tools/internal/stt/pipeline"
	"audio-tools/internal/sttengine"
)

// Decision is the structured policy result callers surface to users.
type Decision struct {
	Text         string
	Filtered     bool
	FilterReason string
	Stages       []string
}

// Policy wraps the existing egress Gate so streaming and unary paths share the
// same stage construction and decision semantics.
type Policy struct {
	gate *egress.Gate
}

// New constructs a policy from the active StreamConfig and optional engine
// registry. nil registry preserves the old direct cfg-driven stage set.
func New(cfg sttpkg.StreamConfig, registry *sttengine.Registry, speaker egress.SpeakerIsolation) Policy {
	params := sttengine.EgressParams{
		HallucinationFilterEnabled: cfg.HallucinationFilterEnabled,
		NoSpeechThreshold:          cfg.NoSpeechThreshold,
		LogProbThreshold:           cfg.LogProbThreshold,
		IsHallucination:            voice.IsWhisperHallucination,
		SpeakerIsolation:           speaker,
	}
	if registry != nil {
		engineID := cfg.EngineID
		if engineID == "" {
			engineID = registry.ResolveEngineID(nil)
		}
		return Policy{gate: egress.NewGate(registry.EgressStages(engineID, params)...)}
	}

	var stages []egress.Stage
	if params.HallucinationFilterEnabled {
		stages = append(stages, egress.HallucinationStage{IsHallucination: params.IsHallucination})
	}
	stages = append(stages, egress.ConfidenceStage{
		NoSpeechThreshold: params.NoSpeechThreshold,
		LogProbThreshold:  params.LogProbThreshold,
	})
	if params.SpeakerIsolation != nil {
		stages = append(stages, egress.SpeakerStage{Isolation: params.SpeakerIsolation})
	}
	return Policy{gate: egress.NewGate(stages...)}
}

// Stages reports the ordered stage names used by this policy.
func (p Policy) Stages() []string {
	if p.gate == nil {
		return nil
	}
	return p.gate.Stages()
}

// Gate exposes the shared egress gate for streaming, which still needs the
// full SegmentDecision details for speaker-rejection events.
func (p Policy) Gate() *egress.Gate { return p.gate }

// Apply runs text and recognition signals through the shared egress policy.
func (p Policy) Apply(ctx context.Context, text, language string, confidence *sttchain.Confidence, audio []byte) Decision {
	d := p.gate.Apply(ctx, egress.SegmentDecision{
		Text:       text,
		Language:   language,
		Confidence: confidence,
		Audio:      audio,
	})
	out := Decision{Text: text, Stages: p.Stages()}
	if d.Outcome == egress.Drop || d.Outcome == egress.Reject {
		out.Text = ""
		out.Filtered = true
		out.FilterReason = d.Reason
	}
	return out
}

// ApplyResult applies the policy to a unary chain result.
func (p Policy) ApplyResult(ctx context.Context, res *sttchain.Result, audio []byte) Decision {
	if res == nil {
		return p.Apply(ctx, "", "", nil, audio)
	}
	return p.Apply(ctx, res.Text, res.DetectedLanguage, res.Confidence, audio)
}
