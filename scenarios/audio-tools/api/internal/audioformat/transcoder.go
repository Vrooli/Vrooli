package audioformat

import (
	"context"

	intaudio "audio-tools/internal/audio"
)

// Transcoder adapts the domain-neutral audio transformation operation to
// consumers that only need a format conversion. Keeping this adapter beside
// the format engine prevents composition roots from owning ffmpeg behavior.
type Transcoder struct{}

// Transcode converts audio into outputFormat with no trim, fade, or volume
// adjustments. It satisfies diagnostics' narrow transcoder seam.
func (Transcoder) Transcode(ctx context.Context, audio []byte, outputFormat string) ([]byte, error) {
	return intaudio.TranscodeOpts(ctx, audio, outputFormat, 0, 0, 0)
}
