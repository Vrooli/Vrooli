package audioformat

import (
	"context"
	"fmt"
)

// canonicalDecodeArgs builds the ffmpeg argv that decodes ANY container
// (ffmpeg auto-detects the input format from the byte stream) into the
// canonical STT representation written to stdout. Decode-only: no filters,
// to keep the untrusted-input attack surface minimal.
func canonicalDecodeArgs() []string {
	return []string{
		"-y", "-loglevel", "error",
		"-i", "pipe:0",
		"-f", "s16le",
		"-ar", fmt.Sprint(CanonicalSampleRate),
		"-ac", fmt.Sprint(CanonicalChannels),
		"pipe:1",
	}
}

// Normalize converts a whole-file (batch) audio buffer to the canonical
// STT representation (16-bit LE PCM, mono, 16 kHz).
//
//   - codec==CodecPCMS16LE → fast-path: the bytes are already canonical,
//     returned unchanged with no ffmpeg invocation.
//   - any other codec → one-shot ffmpeg decode via the Runner seam.
//   - ffmpeg absent for a non-PCM codec → ErrFfmpegRequired (the caller
//     turns this into a capability decision; the substrate never returns
//     undecoded bytes pretending to be PCM).
func (e *Engine) Normalize(ctx context.Context, codec Codec, in []byte) ([]byte, error) {
	if codec.IsCanonicalPCM() {
		return in, nil
	}
	if codec == CodecUnknown {
		return nil, ErrUnknownFormat
	}
	if !e.hasFfmpeg() {
		return nil, ErrFfmpegRequired
	}
	out, err := e.runner.Run(ctx, "ffmpeg", in, canonicalDecodeArgs()...)
	if err != nil {
		return nil, fmt.Errorf("audio-tools/audioformat: normalize %s: %w", codec, err)
	}
	return out, nil
}
