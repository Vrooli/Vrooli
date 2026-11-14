package egress

import "context"

// HallucinationStage is the text-domain stage. It drops segments whose text
// matches one of Whisper's known silence-hallucination phrases ("thank you
// for watching", "please subscribe", ...). The predicate is injected —
// production wires pipeline.IsWhisperHallucination — so the stage stays
// decoupled from the phrase list's owner and a future manifest can scope
// the list per engine.
type HallucinationStage struct {
	// IsHallucination reports whether the candidate text is a known
	// hallucination phrase. A nil predicate makes the stage a no-op.
	IsHallucination func(string) bool
}

// Name identifies the stage in Gate.Stages().
func (s HallucinationStage) Name() string { return "hallucination" }

// Apply drops the segment when the injected predicate flags its text.
func (s HallucinationStage) Apply(_ context.Context, in SegmentDecision) SegmentDecision {
	if s.IsHallucination != nil && s.IsHallucination(in.Text) {
		in.Outcome = Drop
		in.Reason = "hallucination"
	}
	return in
}

var _ Stage = HallucinationStage{}
