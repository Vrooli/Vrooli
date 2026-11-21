package audio_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audio"
	"audio-tools/internal/audio/mocks"
)

func TestDurationSecondsProbesEncodedOutput(t *testing.T) {
	const probeJSON = `{"streams":[{"codec_name":"pcm_s16le","sample_rate":"44100","channels":1}],"format":{"format_name":"wav","duration":"2.75"}}`
	fake := &mocks.FakeRunner{Respond: func(name string, _ []string) ([]byte, error) {
		require.Equal(t, "ffprobe", name, "duration must come from a real probe of the output")
		return []byte(probeJSON), nil
	}}
	swapRunner(t, fake)

	require.InDelta(t, 2.75, audio.DurationSeconds(context.Background(), []byte("OUT")), 1e-9)
	require.Len(t, fake.Calls, 1)
}

func TestDurationSecondsIsZeroWhenUnknown(t *testing.T) {
	cases := map[string]struct {
		probeJSON string
		err       error
	}{
		"probe error":          {err: errors.New("boom")},
		"missing duration":     {probeJSON: `{"format":{"format_name":"wav"}}`},
		"non-numeric duration": {probeJSON: `{"format":{"duration":"N/A"}}`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fake := &mocks.FakeRunner{Respond: func(string, []string) ([]byte, error) {
				return []byte(tc.probeJSON), tc.err
			}}
			swapRunner(t, fake)

			require.Zero(t, audio.DurationSeconds(context.Background(), []byte("OUT")))
		})
	}
}

func TestDurationSecondsIsZeroWithoutFfprobe(t *testing.T) {
	restore := audio.SetFfmpegAvailableForTest(true, false)
	t.Cleanup(restore)

	require.Zero(t, audio.DurationSeconds(context.Background(), []byte("OUT")))
}
