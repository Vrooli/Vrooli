package streamharness

import (
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixtureIsSixtySecondsOfCanonicalPCM(t *testing.T) {
	b, err := io.ReadAll(Fixture())
	require.NoError(t, err)
	require.Len(t, b, SampleRate*FixtureSeconds*2)
}

func TestFrameCarriesCanonicalBoundsAndDigest(t *testing.T) {
	pcm := make([]byte, BatchSamples*2)
	frame := Frame(7, 3200, pcm)
	require.Equal(t, 4+8+8+8+32+len(pcm), len(frame))
	require.Equal(t, []byte(FrameMagic), frame[:4])
	require.Equal(t, int64(7), int64(binary.BigEndian.Uint64(frame[4:12])))
	require.Equal(t, int64(3200), int64(binary.BigEndian.Uint64(frame[12:20])))
	require.Equal(t, int64(3200+BatchSamples), int64(binary.BigEndian.Uint64(frame[20:28])))
}

func TestThresholdsRejectMissingLiveSignals(t *testing.T) {
	err := (Result{Segments: 1, TerminalReason: "completed"}).Validate(DefaultThresholds)
	require.ErrorContains(t, err, "partials")
	good := (Result{Partials: 10, Segments: 1, TerminalReason: "completed"}).Validate(DefaultThresholds)
	require.NoError(t, good)
	tooSparse := (Result{Partials: 10, Segments: 1, MaxPartialGapBatches: 31, TerminalReason: "completed"}).Validate(DefaultThresholds)
	require.ErrorContains(t, tooSparse, "partial gap")
}
