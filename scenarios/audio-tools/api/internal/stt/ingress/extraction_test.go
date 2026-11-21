package ingress_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/stt/ingress"
	"audio-tools/internal/stt/ingress/mocks"
)

func TestDefaultExtractionWindowIsInteractiveSized(t *testing.T) {
	// Canonical PCM is 32,000 bytes per second. The default must not recreate
	// the old ~3 s first-text stall when extraction is enabled for dictation.
	require.Equal(t, 25_600, ingress.DefaultExtractionWindowBytes)
	require.Equal(t, 16_000, ingress.DefaultInitialExtractionWindowBytes)
}

func TestExtractionEnhancer_UsesShortInitialWindowThenSteadyState(t *testing.T) {
	ext := &mocks.FakeTargetExtractor{}
	enh := ingress.ExtractionEnhancer{Extractor: ext}

	pcm := make([]byte, ingress.DefaultInitialExtractionWindowBytes+ingress.DefaultExtractionWindowBytes)
	out, cleanup, err := enh.Process(context.Background(), feed(pcm))
	require.NoError(t, err)
	defer cleanup()
	drain(t, out)

	require.Equal(t, [][]byte{
		make([]byte, ingress.DefaultInitialExtractionWindowBytes),
		make([]byte, ingress.DefaultExtractionWindowBytes),
	}, ext.Windows)
}

// TestExtractionEnhancer_WindowsAndPassesThroughCleaned proves the enhancer
// chops the stream into fixed windows, runs each through the extractor, flushes
// a short final tail on close, and forwards the cleaned PCM.
func TestExtractionEnhancer_WindowsAndPassesThroughCleaned(t *testing.T) {
	ext := &mocks.FakeTargetExtractor{} // identity transform
	enh := ingress.ExtractionEnhancer{Extractor: ext, WindowBytes: 4}

	in := feed([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	out, cleanup, err := enh.Process(context.Background(), in)
	require.NoError(t, err)
	defer cleanup()

	got := drain(t, out)
	require.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, bytes.Join(got, nil), "identity extractor reconstructs the stream")

	// 2 full 4-byte windows + a 2-byte tail flushed on close.
	require.Equal(t, [][]byte{{1, 2, 3, 4}, {5, 6, 7, 8}, {9, 10}}, ext.Windows)
}

// TestExtractionEnhancer_CleanedAudioReplacesInput proves the cleaned PCM the
// extractor returns is what flows downstream — extraction substitutes audio
// before recognition (the whole point of an ingress stage vs the egress gate).
func TestExtractionEnhancer_CleanedAudioReplacesInput(t *testing.T) {
	ext := &mocks.FakeTargetExtractor{Transform: func(pcm []byte) []byte {
		// Return a distinct, same-length "isolated" signal.
		cleaned := make([]byte, len(pcm))
		for i := range cleaned {
			cleaned[i] = 0xAA
		}
		return cleaned
	}}
	enh := ingress.ExtractionEnhancer{Extractor: ext, WindowBytes: 4}

	out, cleanup, err := enh.Process(context.Background(), feed([]byte{1, 2, 3, 4}))
	require.NoError(t, err)
	defer cleanup()

	got := bytes.Join(drain(t, out), nil)
	require.Equal(t, []byte{0xAA, 0xAA, 0xAA, 0xAA}, got, "downstream sees the cleaned audio, not the original mixture")
}

// TestExtractionEnhancer_ErrorDegradesToPassthrough proves a per-window
// extractor failure does not drop the utterance — the original window passes
// through unmodified so recognition still happens (degraded, not lost).
func TestExtractionEnhancer_ErrorDegradesToPassthrough(t *testing.T) {
	ext := &mocks.FakeTargetExtractor{Err: errors.New("resource down")}
	enh := ingress.ExtractionEnhancer{Extractor: ext, WindowBytes: 4}

	out, cleanup, err := enh.Process(context.Background(), feed([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
	require.NoError(t, err)
	defer cleanup()

	require.Equal(t, []byte{1, 2, 3, 4, 5, 6, 7, 8}, bytes.Join(drain(t, out), nil), "failed extraction passes the original audio through")
}

// TestExtractionEnhancer_CtxCancelClosesOutput proves cancellation tears down
// the pump and closes the output channel (no goroutine leak).
func TestExtractionEnhancer_CtxCancelClosesOutput(t *testing.T) {
	ext := &mocks.FakeTargetExtractor{}
	enh := ingress.ExtractionEnhancer{Extractor: ext, WindowBytes: 4}

	ctx, cancel := context.WithCancel(context.Background())
	in := make(chan sttchain.AudioChunk) // never closes
	out, cleanup, err := enh.Process(ctx, in)
	require.NoError(t, err)
	defer cleanup()

	cancel()
	// Output channel must close once the context is cancelled.
	for range out { //nolint:revive // draining until close
	}
}
