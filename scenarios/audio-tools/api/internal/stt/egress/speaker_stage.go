package egress

import "context"

// SpeakerVerdict is the audio-domain outcome for one segment: did the active
// speaker-isolation method allow it, and why. It is the seam's return shape so
// the egress package does not depend on the pipeline's richer SpeakerDecision.
type SpeakerVerdict struct {
	// Allowed reports whether the segment may be emitted. The isolation
	// implementation already applies mode semantics (advisory always allows;
	// filter rejects a non-match; fallback-without-verification allows when
	// verification could not run).
	Allowed bool
	// Reason explains a rejection (surfaced on the speaker-rejection event).
	Reason string
	// FallbackUsed is true when the segment was allowed WITHOUT verification
	// actually running (resource down / no enrolled profile) under
	// FallbackWithoutVerification semantics — surfaced so the consumer knows
	// the gate did not truly verify.
	FallbackUsed bool
	// Score is the best cosine-similarity the active profiles produced for this
	// segment (0 when verification could not run). Threshold is the configured
	// match cutoff. Both are surfaced on the rejection event so the UI banner
	// can show the real "score X < threshold Y" instead of 0.00/0.00.
	Score     float64
	Threshold float64
}

// SpeakerIsolation is the pluggable audio-domain identity check for the EGRESS
// gate: it can Emit/Drop/Reject a finished segment's text but cannot substitute
// audio. The manifest's active speakerIsolation method selects which
// implementation is wired ("verification" today). Swapping the method is a
// one-field manifest edit — callers never branch on the method name. Isolating
// the audio itself (target-speaker extraction) is the complementary INGRESS
// seam ingress.TargetExtractor, gated by config — not an egress method.
//
// seam: SpeakerIsolation is the audio-domain egress seam (SEAMS.md row
// "egress.SpeakerIsolation"). Production wires the verification adapter
// (internal/stt/pipeline) built from the live SpeakerConfig + the
// speaker-verification resource client; tests substitute a fake.
type SpeakerIsolation interface {
	Evaluate(ctx context.Context, audio []byte) SpeakerVerdict
}

// SpeakerStage is the audio-domain egress stage. It runs the active
// SpeakerIsolation method against the segment's canonical-PCM audio and, when
// the segment is not allowed (a non-enrolled voice under filter mode), assigns
// a Reject outcome so the Segmenter emits a speaker-rejection event instead of
// the text. Segments without audio bytes (non-PCM strategies like Passthrough)
// pass through unverified — speaker isolation only applies to the PCM path.
type SpeakerStage struct {
	Isolation SpeakerIsolation
}

func (SpeakerStage) Name() string { return "speaker" }

func (s SpeakerStage) Apply(ctx context.Context, in SegmentDecision) SegmentDecision {
	if s.Isolation == nil || len(in.Audio) == 0 {
		return in // no isolation wired, or no audio to verify -> emit unchanged
	}
	v := s.Isolation.Evaluate(ctx, in.Audio)
	in.Score = v.Score
	in.Threshold = v.Threshold
	if v.Allowed {
		in.FallbackUsed = v.FallbackUsed
		return in
	}
	in.Outcome = Reject
	in.Reason = v.Reason
	in.FallbackUsed = v.FallbackUsed
	return in
}
