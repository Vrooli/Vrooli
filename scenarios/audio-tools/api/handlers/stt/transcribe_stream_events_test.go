package stt

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// TestProtoForEvent_VadStateRoundtrip asserts that the chain→proto bridge
// faithfully carries every field of VadStateEvent. This is the cross-
// transport contract: WS and Connect bidi must surface the same data.
func TestProtoForEvent_VadStateRoundtrip(t *testing.T) {
	in := sttchain.StreamEvent{
		Kind: sttchain.StreamEventVadState,
		VadState: &sttchain.VadStateEvent{
			Voiced:           true,
			SilenceElapsedMs: 123,
			SilenceTimeoutMs: 1500,
			TickSeq:          9,
		},
	}
	got := protoForEvent(in)
	require.NotNil(t, got, "vad-state event must produce a non-nil proto envelope")

	vs, ok := got.Event.(*sttv1.TranscribeStreamEvent_VadState)
	require.True(t, ok, "envelope must hold a VadState oneof variant; got %T", got.Event)
	require.NotNil(t, vs.VadState)
	require.Equal(t, true, vs.VadState.Voiced)
	require.Equal(t, int64(123), vs.VadState.SilenceElapsedMs)
	require.Equal(t, int64(1500), vs.VadState.SilenceTimeoutMs)
	require.Equal(t, uint64(9), vs.VadState.TickSeq)
}

// TestProtoForEvent_VadStateNilDropsEvent asserts that a malformed
// vad-state event (nil VadState pointer) returns nil instead of panicking.
// Matches the defensive nil-guard pattern other branches use.
func TestProtoForEvent_VadStateNilDropsEvent(t *testing.T) {
	got := protoForEvent(sttchain.StreamEvent{Kind: sttchain.StreamEventVadState, VadState: nil})
	require.Nil(t, got)
}

// TestProtoForEvent_SpeakerRejectionCarriesScore asserts the chain→proto
// bridge forwards the similarity score and threshold so the Connect transport
// surfaces the same numbers as the WS transport — without them the rejection
// banner can only show 0.00/0.00.
func TestProtoForEvent_SpeakerRejectionCarriesScore(t *testing.T) {
	in := sttchain.StreamEvent{
		Kind: sttchain.StreamEventSpeakerRejection,
		SpeakerRejection: &sttchain.SpeakerRejectionEvent{
			Reason:    "speaker verification failed",
			Score:     0.42,
			Threshold: 0.7,
		},
	}
	got := protoForEvent(in)
	require.NotNil(t, got)
	sr, ok := got.Event.(*sttv1.TranscribeStreamEvent_SpeakerRejection)
	require.True(t, ok, "envelope must hold a SpeakerRejection oneof; got %T", got.Event)
	require.InDelta(t, 0.42, sr.SpeakerRejection.GetScore(), 1e-9)
	require.InDelta(t, 0.7, sr.SpeakerRejection.GetThreshold(), 1e-9)
}
