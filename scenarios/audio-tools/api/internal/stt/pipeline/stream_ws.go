// DOC: docs/internal/SEAMS.md#voice-stream-websocket-seam
//
// This file previously hosted voice.Service.HandleStreamWS — a ~440-line
// fused WS handler that owned WebM init handling, client-driven VAD,
// partial transcription with overlap dedup, segment-final speaker
// verification, and final tail transcription as one pipeline. The
// streaming-STT decoupling plan
// (audio-tools-streaming-stt-decouple-strategy-from-provider) deleted
// that handler when both transports moved to the new Segmenter +
// StrategySelector pipeline. The shared constants and small pure
// helpers that survived the rewrite live here so the rest of
// internal/stt/pipeline (config, speaker, transcribe, types) and the new
// strategy package can use them.
package pipeline

import (
	"bytes"
)

// VadLookbackMs is the audio lookback margin applied when a client
// signals speech onset. The legacy WS handler used this to rewind
// segment offsets against the ~300-500ms client-side VAD latency.
// Retained as a public constant for strategy implementations that
// adopt the same lookback semantics.
const VadLookbackMs = 600

// AudioBitrateBps is the expected audio bitrate (matches the embed's
// AUDIO_BITRATE constant of 48 kbps Opus). Retained for strategies
// that need to convert millisecond windows to byte offsets.
const AudioBitrateBps = 48_000

// FindWebMInitEnd locates the end of the WebM initialization segment
// by scanning for the first Cluster element ID (0x1F43B675). Returns
// the byte offset of the Cluster start, or 0 if not found. Used by
// transports that need to split a WebM-framed stream so a downstream
// batch transcriber can decode mid-stream slices.
func FindWebMInitEnd(buf []byte) int {
	clusterID := []byte{0x1F, 0x43, 0xB6, 0x75}
	idx := bytes.Index(buf, clusterID)
	if idx < 0 {
		return 0
	}
	return idx
}
