package ingress_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/stt/ingress"
	"audio-tools/internal/stt/ingress/mocks"
)

// drain collects every chunk from a channel until it closes.
func drain(t *testing.T, ch <-chan sttchain.AudioChunk) [][]byte {
	t.Helper()
	var out [][]byte
	for c := range ch {
		out = append(out, c.Audio)
	}
	return out
}

func feed(chunks ...[]byte) <-chan sttchain.AudioChunk {
	in := make(chan sttchain.AudioChunk, len(chunks))
	for _, c := range chunks {
		in <- sttchain.AudioChunk{Audio: c}
	}
	close(in)
	return in
}

// An empty Pipeline is the identity: same channel, no-op cleanup.
func TestPipeline_EmptyIsIdentity(t *testing.T) {
	in := feed([]byte("a"))
	out, cleanup, err := ingress.NewPipeline().Process(context.Background(), in)
	require.NoError(t, err)
	require.NotNil(t, cleanup)
	cleanup() // must be safe
	require.Equal(t, [][]byte{[]byte("a")}, drain(t, out))
}

// Enhancers run in order: the first's output feeds the next. Tags prove order.
func TestPipeline_RunsEnhancersInOrder(t *testing.T) {
	first := &mocks.FakeEnhancer{NameVal: "first", Transform: func(b []byte) []byte { return append(append([]byte{}, b...), '1') }}
	second := &mocks.FakeEnhancer{NameVal: "second", Transform: func(b []byte) []byte { return append(append([]byte{}, b...), '2') }}
	p := ingress.NewPipeline(first, second)
	require.Equal(t, []string{"first", "second"}, p.Names())

	out, cleanup, err := p.Process(context.Background(), feed([]byte("x")))
	require.NoError(t, err)
	defer cleanup()
	// "x" → first appends '1' → second appends '2' → "x12"
	require.Equal(t, [][]byte{[]byte("x12")}, drain(t, out))
}

// A start error from a later enhancer rolls back the cleanups of the stages
// already started, and returns the error with no channel.
func TestPipeline_StartErrorRollsBack(t *testing.T) {
	first := &mocks.FakeEnhancer{NameVal: "first"}
	boom := errors.New("denoise backend unavailable")
	second := &mocks.FakeEnhancer{NameVal: "second", StartErr: boom}
	p := ingress.NewPipeline(first, second)

	out, cleanup, err := p.Process(context.Background(), feed([]byte("x")))
	require.ErrorIs(t, err, boom)
	require.Nil(t, out)
	require.NotNil(t, cleanup)
	// first was started then rolled back; second never started.
	require.Equal(t, 1, first.Started)
	require.Equal(t, 1, first.Cleaned, "the started stage must be cleaned up on rollback")
	require.Equal(t, 0, second.Started)
}
