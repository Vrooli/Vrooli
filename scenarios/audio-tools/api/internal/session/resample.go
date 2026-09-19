package session

import (
	"encoding/binary"
	"fmt"

	"audio-tools/internal/protoint"
)

// mu-law decode table for the 256 possible mu-law byte values.
var muLawDecode = func() [256]int16 {
	var t [256]int16
	for i := 0; i < 256; i++ {
		mulawByte := ^uint8(i)
		sign := mulawByte & 0x80
		exponent := (mulawByte >> 4) & 0x07
		mantissa := mulawByte & 0x0F
		sample := int16(mantissa)<<3 + 0x84
		sample <<= exponent
		sample -= 0x84
		if sign != 0 {
			t[i] = -sample
		} else {
			t[i] = sample
		}
	}
	return t
}()

// MuLawToPCM16 converts an 8 kHz mu-law byte stream (Twilio Media-Stream
// shape) to 16 kHz signed PCM16 little-endian. Uses linear interpolation for
// up-sampling — adequate for human voice; not for music fidelity.
func MuLawToPCM16(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}
	// Decode mu-law -> int16 at 8 kHz.
	pcm8k := make([]int16, len(in))
	for i, b := range in {
		pcm8k[i] = muLawDecode[b]
	}
	// Linear-interpolate up to 16 kHz: between each pair (a, b), insert
	// (a+b)/2; for the final sample, duplicate.
	out := make([]int16, len(pcm8k)*2)
	for i := 0; i < len(pcm8k)-1; i++ {
		out[i*2] = pcm8k[i]
		out[i*2+1] = protoint.FromInt32ToInt16((int32(pcm8k[i]) + int32(pcm8k[i+1])) / 2)
	}
	out[(len(pcm8k)-1)*2] = pcm8k[len(pcm8k)-1]
	out[(len(pcm8k)-1)*2+1] = pcm8k[len(pcm8k)-1]
	buf := make([]byte, len(out)*2)
	for i, s := range out {
		binary.LittleEndian.PutUint16(buf[i*2:], protoint.PCMUint16(s))
	}
	return buf
}

// muLawEncode is the inverse of muLawDecode. Used by PCM16To8kMuLaw.
func muLawEncode(sample int16) uint8 {
	const bias = 0x84
	const clip = 32635
	sign := uint8(0)
	if sample < 0 {
		sample = -sample
		sign = 0x80
	}
	if sample > clip {
		sample = clip
	}
	sample += bias
	exponent := uint8(7)
	for mask := int16(0x4000); (sample&mask) == 0 && exponent > 0; mask >>= 1 {
		exponent--
	}
	mantissa := uint8((sample >> (exponent + 3)) & 0x0F)
	return ^(sign | (exponent << 4) | mantissa)
}

// PCM16To8kMuLaw is the inverse path: 16 kHz signed PCM16 little-endian ->
// 8 kHz mu-law. Down-samples by averaging adjacent pairs.
func PCM16To8kMuLaw(in []byte) ([]byte, error) {
	if len(in)%2 != 0 {
		return nil, fmt.Errorf("audio-tools/session: PCM16 length %d not aligned to 2 bytes", len(in))
	}
	samples := len(in) / 2
	pcm := make([]int16, samples)
	for i := 0; i < samples; i++ {
		pcm[i] = protoint.PCMInt16(binary.LittleEndian.Uint16(in[i*2:]))
	}
	out := make([]byte, samples/2)
	for i := 0; i < samples/2; i++ {
		avg := (int32(pcm[i*2]) + int32(pcm[i*2+1])) / 2
		out[i] = muLawEncode(protoint.FromInt32ToInt16(avg))
	}
	return out, nil
}
