package ingress

import (
	"context"

	"audio-tools/internal/ai/sttchain"
)

// DefaultExtractionWindowBytes is ~3 s of canonical PCM (16 kHz · mono · s16le
// = 32_000 bytes/s). Target-speaker separation models operate on a window of
// audio, not a continuous frame stream, so the enhancer buffers the incoming
// PCM into utterance-sized windows before each Extract call. Three seconds
// balances separation quality against added latency; true frame-level streaming
// extraction is an explicit non-goal (see docs/reference/configuration.md).
const DefaultExtractionWindowBytes = 16000 * 2 * 3

// TargetExtractor isolates the enrolled speaker's voice from a window of
// canonical PCM (s16le / 16 kHz / mono) and returns cleaned canonical PCM of
// the SAME format. It is the audio-domain identity primitive backing the
// ingress ExtractionEnhancer.
//
// Contract: an implementation that cannot run (resource down, no enrolled
// profile, no match under the active mode) returns the INPUT bytes unchanged
// with a nil error, so the stream degrades to passthrough rather than dropping
// audio. A non-nil error means the window could not be processed at all; the
// enhancer then passes the original window through unmodified.
type TargetExtractor interface {
	Extract(ctx context.Context, pcm []byte) ([]byte, error)
}

// ExtractionEnhancer is the pre-recognition ingress stage that isolates the
// enrolled speaker BEFORE the VAD and recognizer run, so co-occurring voices
// (a second person, background speech) never reach transcription. Unlike the
// egress verification gate — which can only DROP a finished segment's text —
// extraction modifies the audio itself, removing the interfering speaker rather
// than just suppressing a mis-attributed transcript.
//
// It operates in the canonical-PCM audio domain, so like the denoise enhancer
// it is wired on the PCM-decode path (Whisper VAD/Overlap) in v1. Covering
// native-streaming engines (Kyutai), which consume the raw stream rather than
// canonical PCM, would require normalize→extract→re-encode and is future work.
//
// seam: ExtractionEnhancer is an ingress audio-enhancement seam (SEAMS.md row
// "ingress.Enhancer" / TargetExtractor). Production wires the speaker-extraction
// adapter (handlers/stt) built from the live SpeakerConfig + the
// speaker-verification resource client; tests substitute a fake TargetExtractor.
type ExtractionEnhancer struct {
	Extractor TargetExtractor
	// WindowBytes overrides DefaultExtractionWindowBytes when > 0.
	WindowBytes int
}

// Name implements Enhancer.
func (ExtractionEnhancer) Name() string { return "target-extraction" }

// Process buffers canonical-PCM chunks into windows, runs each through the
// extractor, and emits the cleaned PCM. On clean input EOF it flushes the
// remaining tail through one final extraction. A per-window extractor error is
// non-fatal — the original window passes through — so a transient resource
// failure degrades quality, it does not lose the utterance.
func (e ExtractionEnhancer) Process(ctx context.Context, in <-chan sttchain.AudioChunk) (<-chan sttchain.AudioChunk, func(), error) {
	windowBytes := e.WindowBytes
	if windowBytes <= 0 {
		windowBytes = DefaultExtractionWindowBytes
	}

	out := make(chan sttchain.AudioChunk, 16)
	nctx, ncancel := context.WithCancel(ctx)

	emit := func(pcm []byte) bool {
		cleaned, err := e.Extractor.Extract(nctx, pcm)
		if err != nil || len(cleaned) == 0 {
			cleaned = pcm // degrade to passthrough on failure
		}
		select {
		case out <- sttchain.AudioChunk{Audio: cleaned}:
			return true
		case <-nctx.Done():
			return false
		}
	}

	go func() {
		defer close(out)
		var buf []byte
		for {
			select {
			case c, ok := <-in:
				if !ok {
					if len(buf) > 0 {
						emit(buf)
					}
					return
				}
				buf = append(buf, c.Audio...)
				for len(buf) >= windowBytes {
					window := make([]byte, windowBytes)
					copy(window, buf[:windowBytes])
					if !emit(window) {
						return
					}
					buf = buf[windowBytes:]
				}
			case <-nctx.Done():
				return
			}
		}
	}()

	return out, func() { ncancel() }, nil
}

var _ Enhancer = ExtractionEnhancer{}
