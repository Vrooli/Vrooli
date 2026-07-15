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
