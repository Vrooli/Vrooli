package strategy

import "bytes"

// VadLookbackMs is the audio lookback margin applied when a client signals
// speech onset, used to rewind segment offsets against the ~300-500ms
// client-side VAD latency.
const VadLookbackMs = 600

// AudioBitrateBps is the expected audio bitrate (48 kbps Opus, matching the
// embed's AUDIO_BITRATE).
const AudioBitrateBps = 48_000

// FindWebMInitEnd locates the end of the WebM initialization segment by
// scanning for the first Cluster element ID (0x1F43B675). Returns the byte
// offset of the Cluster start, or 0 if not found.
func FindWebMInitEnd(buf []byte) int {
	clusterID := []byte{0x1F, 0x43, 0xB6, 0x75}
	idx := bytes.Index(buf, clusterID)
	if idx < 0 {
		return 0
	}
	return idx
}
