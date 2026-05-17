package testaudio

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSineSamples(t *testing.T) {
	b := SineSamples(440, 100)
	require.Equal(t, SampleRateHz*100/1000*2, len(b))
}

func TestSilenceSamples(t *testing.T) {
	b := SilenceSamples(100)
	require.Equal(t, SampleRateHz*100/1000*2, len(b))
	for _, v := range b {
		require.Equal(t, byte(0), v)
	}
}

func TestSpeechLike(t *testing.T) {
	b := SpeechLike()
	require.Greater(t, len(b), 0)
}

func TestSilence(t *testing.T) {
	b := Silence()
	require.Equal(t, SampleRateHz*1000/1000*2, len(b))
}
