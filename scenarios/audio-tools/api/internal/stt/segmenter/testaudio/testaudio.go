// Package testaudio synthesizes deterministic PCM audio fixtures for
// streaming-pipeline tests.
//
// Real speech is not required for the parity test: the buffered
// fallback and VAD-segment strategies are driven by byte boundaries
// and silence detection, not by transcript content. Tests assert
// projections (event kind, ordering, byte counts) that hold for
// synthetic input. When a strategy needs a content-bearing fixture
// later (e.g. OverlapAgreeStrategy with a known prefix), this package
// is the right place to add a recorded WAV asset.
package testaudio

import (
	"encoding/binary"
	"math"
)

// SampleRateHz is the standard 16 kHz mono PCM rate used by Whisper
// and by every adapter in the chain.
const SampleRateHz = 16000

// SineSamples returns 16-bit little-endian PCM samples for a sine wave
// of the given frequency and duration. Amplitude is fixed at half the
// 16-bit range so it never clips when mixed with silence.
func SineSamples(freqHz float64, durationMs int) []byte {
	n := SampleRateHz * durationMs / 1000
	buf := make([]byte, n*2)
	amp := math.MaxInt16 / 2
	for i := 0; i < n; i++ {
		v := int16(float64(amp) * math.Sin(2*math.Pi*freqHz*float64(i)/float64(SampleRateHz)))
		binary.LittleEndian.PutUint16(buf[i*2:], uint16(v))
	}
	return buf
}

// SilenceSamples returns zero-filled 16-bit PCM samples for the given
// duration. The strategies treat sustained zeros as a VAD silence
// boundary.
func SilenceSamples(durationMs int) []byte {
	n := SampleRateHz * durationMs / 1000
	return make([]byte, n*2)
}

// SpeechLike returns a 3-second sine-wave-with-pause fixture used as
// the default "sample.wav" replacement. The shape is: 1s tone, 0.7s
// silence, 1.3s tone — enough for VAD to find one mid-stream
// boundary at the default vad_silence_ms=700.
func SpeechLike() []byte {
	out := SineSamples(440, 1000)
	out = append(out, SilenceSamples(700)...)
	out = append(out, SineSamples(660, 1300)...)
	return out
}

// Silence returns a 1-second silence fixture used to assert "no
// segments emitted" behavior for VAD-segment.
func Silence() []byte {
	return SilenceSamples(1000)
}
