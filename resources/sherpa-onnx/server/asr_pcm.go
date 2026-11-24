package main

import "encoding/binary"

const maxRetainedPCMScratchSamples = streamingSampleRate

// decodePCM16LE reuses caller-owned scratch storage for Sherpa's float input.
// The native AcceptWaveform call consumes the slice synchronously, so a stream
// can safely reuse its buffer after that call returns.
func decodePCM16LE(dst []float32, pcm []byte) []float32 {
	samples := len(pcm) / 2
	if cap(dst) < samples {
		dst = make([]float32, samples)
	} else {
		dst = dst[:samples]
	}
	for i := range dst {
		dst[i] = float32(int16(binary.LittleEndian.Uint16(pcm[i*2:]))) / 32768
	}
	return dst
}

// retainPCMScratch bounds per-session retained memory. Large but valid WebSocket
// packets can use a temporary decode buffer without pinning it for the rest of
// the session; ordinary packets retain at most one second of float samples.
func retainPCMScratch(samples []float32) []float32 {
	if cap(samples) > maxRetainedPCMScratchSamples {
		return nil
	}
	return samples
}
