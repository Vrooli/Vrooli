package sttchain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVirtualReplayProviderIsExplicitlyEnvironmentGated(t *testing.T) {
	t.Setenv("VROOLI_AUDIO_SOAK_REPLAY", "")
	require.False(t, NewVirtualReplayProvider().IsAvailable(context.Background()))
	t.Setenv("VROOLI_AUDIO_SOAK_REPLAY", "1")
	require.True(t, NewVirtualReplayProvider().IsAvailable(context.Background()))
}

func TestVirtualReplayProviderAcknowledgesCoverageAndEmitsOneSegment(t *testing.T) {
	t.Setenv("VROOLI_AUDIO_SOAK_REPLAY", "1")
	p := NewVirtualReplayProvider()
	chunks := make(chan AudioChunk, 2)
	chunks <- AudioChunk{Sequence: 0, StartSample: 0, EndSample: 1600, Audio: make([]byte, 3200)}
	chunks <- AudioChunk{Sequence: 1, StartSample: 1600, EndSample: 3200, Audio: make([]byte, 3200)}
	close(chunks)
	events, err := p.TranscribeStreaming(context.Background(), StreamStart{EngineID: virtualReplayEngineID, SessionID: "s"}, chunks)
	require.NoError(t, err)
	var acks, segments, dones int
	for event := range events {
		switch event.Kind {
		case StreamEventAcknowledgement:
			acks++
		case StreamEventSegment:
			segments++
			require.Equal(t, int64(1600), event.Segment.EndSample)
		case StreamEventDone:
			dones++
			require.Equal(t, virtualReplayProvider, event.Done.ProviderID)
		}
	}
	require.Equal(t, 2, acks)
	require.Equal(t, 1, segments)
	require.Equal(t, 1, dones)
}
