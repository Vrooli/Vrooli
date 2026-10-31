package audio_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audio"
	"audio-tools/internal/audio/mocks"
)

func TestNormalize_AllMethods(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	for _, m := range []string{"peak", "rms", "ebu"} {
		_, err := audio.Normalize(context.Background(), []byte("X"), "wav", m, -16.0, "wav")
		require.NoError(t, err)
	}
	// Default lufs path.
	_, err := audio.Normalize(context.Background(), []byte("X"), "wav", "ebu", 0, "")
	require.NoError(t, err)
}

func TestSplit_WithBoundaries(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	chunks, err := audio.Split(context.Background(), []byte("X"), "wav", 0, []float64{1.0, 2.0}, "wav")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(chunks), 1)
}

func TestSplit_WithChunkSeconds(t *testing.T) {
	probeJSON := `{"streams":[{"sample_rate":"16000","channels":1,"bit_rate":"128000","codec_name":"pcm_s16le"}],"format":{"duration":"6.0","format_name":"wav","bit_rate":"128000"}}`
	fake := &mocks.FakeRunner{
		Respond: func(name string, args []string) ([]byte, error) {
			if name == "ffprobe" {
				return []byte(probeJSON), nil
			}
			return []byte("OUT"), nil
		},
	}
	swapRunner(t, fake)
	chunks, err := audio.Split(context.Background(), []byte("X"), "wav", 2.0, nil, "wav")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(chunks), 2)
}

func TestMerge_NoSources(t *testing.T) {
	_, err := audio.Merge(context.Background(), nil, nil, "wav", 0)
	require.Error(t, err)
}

func TestMerge_Concat(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	_, err := audio.Merge(context.Background(), [][]byte{[]byte("a"), []byte("b")}, nil, "wav", 0)
	require.NoError(t, err)
}

func TestMerge_Crossfade(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	_, err := audio.Merge(context.Background(), [][]byte{[]byte("a"), []byte("b"), []byte("c")}, nil, "wav", 0.5)
	require.NoError(t, err)
}

func TestTranscodeOpts_AllSet(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	_, err := audio.TranscodeOpts(context.Background(), []byte("X"), "mp3", 22050, 2, 128000)
	require.NoError(t, err)
	require.Greater(t, len(fake.Calls), 0)
}

func TestTranscodeOpts_Defaults(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("OUT"), nil)
	swapRunner(t, fake)
	_, err := audio.TranscodeOpts(context.Background(), []byte("X"), "", 0, 0, 0)
	require.NoError(t, err)
}

func TestProbe_OK(t *testing.T) {
	probeJSON := `{"streams":[{"sample_rate":"16000","channels":1,"bit_rate":"128000","codec_name":"pcm_s16le"}],"format":{"duration":"3.0","format_name":"wav","bit_rate":"128000","tags":{"title":"x"}}}`
	fake := &mocks.FakeRunner{
		Respond: func(name string, args []string) ([]byte, error) {
			return []byte(probeJSON), nil
		},
	}
	swapRunner(t, fake)
	m, err := audio.Probe(context.Background(), []byte("X"))
	require.NoError(t, err)
	require.Equal(t, "wav", m.Format)
	require.Equal(t, int32(16000), m.SampleRate)
}
