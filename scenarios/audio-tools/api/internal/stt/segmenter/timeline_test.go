// [REQ:ATD-P0-007] Recognition spans must never bind unrelated audio.
package segmenter

import (
	"testing"

	"audio-tools/internal/ai/sttchain"

	"github.com/stretchr/testify/require"
)

func TestAudioTimelineBindsExactRangeAndNamesEviction(t *testing.T) {
	timeline := NewAudioTimeline(4)
	require.NoError(t, timeline.Append(sttchain.AudioChunk{StartSample: 0, EndSample: 2, Audio: []byte{1, 0, 2, 0}}))
	require.NoError(t, timeline.Append(sttchain.AudioChunk{StartSample: 2, EndSample: 4, Audio: []byte{3, 0, 4, 0}}))
	got := timeline.Lookup(1, 3)
	require.Equal(t, AudioRangeAttached, got.Status)
	require.Equal(t, []byte{2, 0, 3, 0}, got.PCM)
	require.NoError(t, timeline.Append(sttchain.AudioChunk{StartSample: 4, EndSample: 6, Audio: []byte{5, 0, 6, 0}}))
	require.Equal(t, AudioRangeEvicted, timeline.Lookup(0, 1).Status)
	require.Equal(t, AudioRangeMissing, timeline.Lookup(5, 7).Status)
}

// TestAudioTimelineRejectsRangeThatDoesNotMatchPCM is the regression for a
// chunk whose declared sample range disagrees with its byte count. Lookup
// slices by sample offset, so accepting such a chunk made a later egress
// verification read a Missing range (or bytes belonging to a different
// position) for audio that was actually captured. The mismatch must be
// rejected at the append seam, not hidden until lookup.
func TestAudioTimelineRejectsRangeThatDoesNotMatchPCM(t *testing.T) {
	timeline := NewAudioTimeline(0)

	err := timeline.Append(sttchain.AudioChunk{StartSample: 0, EndSample: 100, Audio: []byte{1, 0}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not match")

	// An odd byte count cannot be addressed as whole int16 samples.
	err = timeline.Append(sttchain.AudioChunk{StartSample: 0, EndSample: 1, Audio: []byte{1, 0, 1}})
	require.Error(t, err)
	require.Contains(t, err.Error(), "int16")

	// A zero-length chunk at the current cursor remains valid.
	require.NoError(t, timeline.Append(sttchain.AudioChunk{StartSample: 0, EndSample: 0}))
}
