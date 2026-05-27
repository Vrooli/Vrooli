// Package egress is the post-recognition gate for streaming STT: the
// single seam every segment passes through after a strategy produces it
// and before it reaches the wire. It mirrors the audioformat substrate's
// ingress point (one canonical-PCM decode at the Segmenter) with a
// symmetric egress point — one place where quality, confidence, and
// speaker-identity checks live, instead of being scattered across (or
// missing from) each strategy.
//
// The Segmenter constructs one Gate per session from the active engine's
// capability manifest + the resolved StreamConfig, then runs every
// SegmentEvent through it. Strategies never call the Gate directly — they
// emit candidate segments and the Segmenter's interceptor applies the Gate.
package egress

import (
	"context"

	"audio-tools/internal/ai/sttchain"
)

// Outcome is the terminal disposition a Gate assigns to a candidate segment.
type Outcome string

const (
	// Emit forwards the segment to the wire unchanged.
	Emit Outcome = "emit"
	// Drop suppresses the segment entirely: no wire event, and it is
	// excluded from the reconstructed final transcript. Used by the
	// quality stages (hallucination phrase filter, low-confidence).
	Drop Outcome = "drop"
	// Reject suppresses the segment's text but signals an explicit
	// rejection to the consumer (the Segmenter emits a
	// StreamEventSpeakerRejection). Used by the audio-domain speaker stage.
	Reject Outcome = "reject"
)

// SegmentDecision is the unit a Gate processes: one candidate segment plus
// the signals the ordered stages consult. A stage reads the candidate and
// may assign a terminal Outcome (Drop/Reject) with a Reason; once a stage
// goes terminal, later stages are skipped.
//
// The three signal domains map to the three field groups: Text (text
// domain — phrase filter), Confidence (signal domain — no_speech/logprob),
// Audio (audio domain — speaker identity).
type SegmentDecision struct {
	Text       string
	Language   string
	Confidence *sttchain.Confidence
	Audio      []byte

	Outcome Outcome
	Reason  string
	// FallbackUsed is set by the speaker stage when a segment is allowed
	// through despite verification being unavailable (resource down / no
	// profile) under FallbackWithoutVerification semantics. Surfaced on the
	// rejection event so the consumer knows verification did not actually run.
	FallbackUsed bool
}

// Stage is one ordered post-recognition check. A stage is pure with respect
// to session state: every per-segment input arrives on the SegmentDecision.
//
// seam: Stage is the egress post-recognition stage seam (SEAMS.md row
// "egress.Stage"). Production wires the hallucination / confidence /
// speaker stages built from the engine manifest + StreamConfig; tests
// substitute fakes from internal/stt/egress/mocks.
type Stage interface {
	Name() string
	Apply(ctx context.Context, in SegmentDecision) SegmentDecision
}

// Gate runs an ordered list of stages over a candidate segment. A Gate with
// no stages is the identity gate (every segment emits unchanged).
type Gate struct {
	stages []Stage
}

// NewGate constructs a Gate from an ordered stage list.
func NewGate(stages ...Stage) *Gate {
	return &Gate{stages: stages}
}

// Stages reports the ordered stage names — for telemetry and tests.
func (g *Gate) Stages() []string {
	if g == nil {
		return nil
	}
	names := make([]string, 0, len(g.stages))
	for _, s := range g.stages {
		names = append(names, s.Name())
	}
	return names
}

// Apply runs the candidate through every stage in order, stopping at the
// first stage that assigns a terminal (non-Emit) outcome. A nil Gate and a
// stage-less Gate both emit every segment.
func (g *Gate) Apply(ctx context.Context, in SegmentDecision) SegmentDecision {
	d := in
	d.Outcome = Emit
	if g == nil {
		return d
	}
	for _, s := range g.stages {
		d = s.Apply(ctx, d)
		if d.Outcome != Emit {
			break
		}
	}
	return d
}
