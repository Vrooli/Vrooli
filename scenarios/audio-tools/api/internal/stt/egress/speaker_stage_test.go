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

	reject := egress.SpeakerStage{Isolation: &mocks.FakeSpeakerIsolation{Verdict: egress.SpeakerVerdict{Allowed: false, Reason: "not enrolled"}}}
	out = reject.Apply(ctx, egress.SegmentDecision{Text: "music", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Reject, out.Outcome)
	require.Equal(t, "not enrolled", out.Reason)
}

func TestSpeakerStage_FallbackFlagPropagates(t *testing.T) {
	stage := egress.SpeakerStage{Isolation: &mocks.FakeSpeakerIsolation{Verdict: egress.SpeakerVerdict{Allowed: true, FallbackUsed: true}}}
	out := stage.Apply(context.Background(), egress.SegmentDecision{Text: "hi", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Emit, out.Outcome)
	require.True(t, out.FallbackUsed)
}

// TestSpeakerStage_NoAudioOrNoIsolationIsNoOp proves the audio-domain stage
// only applies to PCM segments with a wired isolation; otherwise it emits.
func TestSpeakerStage_NoAudioOrNoIsolationIsNoOp(t *testing.T) {
	ctx := context.Background()
	iso := &mocks.FakeSpeakerIsolation{Verdict: egress.SpeakerVerdict{Allowed: false, Reason: "x"}}

	// No audio bytes (e.g. Passthrough/Kyutai segments) -> emit, isolation not consulted.
	out := egress.SpeakerStage{Isolation: iso}.Apply(ctx, egress.SegmentDecision{Text: "hi", Outcome: egress.Emit})
	require.Equal(t, egress.Emit, out.Outcome)
	require.Empty(t, iso.Seen)

	// No isolation wired -> emit.
	out = egress.SpeakerStage{}.Apply(ctx, egress.SegmentDecision{Text: "hi", Audio: []byte("pcm"), Outcome: egress.Emit})
	require.Equal(t, egress.Emit, out.Outcome)
}
