package egress_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/stt/egress"
	"audio-tools/internal/stt/egress/mocks"
)

func TestSpeakerStage_AllowEmitsRejectBlocks(t *testing.T) {
	ctx := context.Background()

	allow := egress.SpeakerStage{Isolation: &mocks.FakeSpeakerIsolation{Verdict: egress.SpeakerVerdict{Allowed: true}}}
	out := allow.Apply(ctx, egress.SegmentDecision{Text: "hi", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Emit, out.Outcome)

	reject := egress.SpeakerStage{Isolation: &mocks.FakeSpeakerIsolation{Verdict: egress.SpeakerVerdict{Allowed: false, Reason: "not enrolled", Score: 0.12, Threshold: 0.7}}}
	out = reject.Apply(ctx, egress.SegmentDecision{Text: "music", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Reject, out.Outcome)
	require.Equal(t, "not enrolled", out.Reason)
	// The verdict's similarity numbers must land on the decision so the
	// rejection event (and the UI banner) can show the real score/threshold
	// instead of 0.00/0.00.
	require.InDelta(t, 0.12, out.Score, 1e-9)
	require.InDelta(t, 0.7, out.Threshold, 1e-9)
}

// TestSpeakerStage_StampsScoreOnAllow proves the stage also carries the
// similarity numbers when a segment is allowed — advisory-mode annotations
// rely on the score being present even when nothing is dropped.
func TestSpeakerStage_StampsScoreOnAllow(t *testing.T) {
	stage := egress.SpeakerStage{Isolation: &mocks.FakeSpeakerIsolation{Verdict: egress.SpeakerVerdict{Allowed: true, Score: 0.88, Threshold: 0.7}}}
	out := stage.Apply(context.Background(), egress.SegmentDecision{Text: "hi", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Emit, out.Outcome)
	require.InDelta(t, 0.88, out.Score, 1e-9)
	require.InDelta(t, 0.7, out.Threshold, 1e-9)
}

func TestSpeakerStage_FallbackFlagPropagates(t *testing.T) {
	stage := egress.SpeakerStage{Isolation: &mocks.FakeSpeakerIsolation{Verdict: egress.SpeakerVerdict{Allowed: true, FallbackUsed: true}}}
	out := stage.Apply(context.Background(), egress.SegmentDecision{Text: "hi", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Emit, out.Outcome)
	require.True(t, out.FallbackUsed)
}

// TestSpeakerStage_NoAudioOrNoIsolationIsNoOp proves the audio-domain stage
// only applies to PCM segments with a wired isolation; otherwise it emits.
func TestSpeakerStage_NoAudioFailsClosedUnlessPolicyAllowsFallback(t *testing.T) {
	ctx := context.Background()
	iso := &mocks.FakeSpeakerIsolation{Verdict: egress.SpeakerVerdict{Allowed: false, Reason: "x"}}

	// Missing span audio must not silently bypass a required filter.
	out := egress.SpeakerStage{Isolation: iso}.Apply(ctx, egress.SegmentDecision{Text: "hi", Outcome: egress.Emit})
	require.Equal(t, egress.Reject, out.Outcome)
	require.Contains(t, out.Reason, "audio is unavailable")
	require.Empty(t, iso.Seen)

	// No isolation wired -> emit.
	out = egress.SpeakerStage{}.Apply(ctx, egress.SegmentDecision{Text: "hi", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Emit, out.Outcome)
}
