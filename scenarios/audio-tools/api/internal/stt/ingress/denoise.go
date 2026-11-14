package ingress

import (
	"context"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/audioformat"
)

// DefaultDenoiseFilter is the ffmpeg -af chain used by the denoise enhancer.
// afftdn is ffmpeg's built-in adaptive FFT denoiser — no model file, no GPU,
// available in any standard ffmpeg build. nf=-25 sets a moderate noise floor
// (dB): aggressive enough to attenuate steady background noise (fans, TV hum)
// without clipping speech. Operators can override via the StreamConfig lever.
const DefaultDenoiseFilter = "afftdn=nf=-25"

// FilterRunner is the substrate seam the DenoiseEnhancer drives: a per-session
// canonical-PCM→canonical-PCM streaming filter. Production wires
// *audioformat.Engine (NewStreamFilter); tests substitute a fake.
type FilterRunner interface {
	NewStreamFilter(ctx context.Context, filter string) (audioformat.StreamDecoder, error)
}

// DenoiseEnhancer suppresses background noise on the canonical-PCM stream
// before the VAD and Whisper see it, improving both silence detection and
// transcription accuracy in noisy environments. It is backed by a per-session
// ffmpeg afftdn process via the FilterRunner seam (mirrors the audioformat
// stream decoder's lifecycle).
type DenoiseEnhancer struct {
	Runner FilterRunner
	// Filter overrides DefaultDenoiseFilter when non-empty.
	Filter string
}

// Name implements Enhancer.
func (DenoiseEnhancer) Name() string { return "denoise" }

// Process starts the ffmpeg filter process and pumps in→stdin, stdout→out,
// mirroring segmenter.normalizeChunks. A start error (e.g. ErrFfmpegRequired)
// is returned to the caller; the Segmenter only adds this enhancer when ffmpeg
// is available, so a start error here is a genuine session failure.
func (d DenoiseEnhancer) Process(ctx context.Context, in <-chan sttchain.AudioChunk) (<-chan sttchain.AudioChunk, func(), error) {
	filter := d.Filter
	if filter == "" {
		filter = DefaultDenoiseFilter
	}
	dec, err := d.Runner.NewStreamFilter(ctx, filter)
	if err != nil {
		return nil, nil, err
	}

	out := make(chan sttchain.AudioChunk, 16)
	nctx, ncancel := context.WithCancel(ctx)

	// Feeder: inbound PCM chunks → filter stdin. CloseInput on clean EOF so
	// ffmpeg flushes its tail; exit on cancel.
	go func() {
		for {
			select {
			case c, ok := <-in:
				if !ok {
					_ = dec.CloseInput()
					return
				}
				if err := dec.Write(c.Audio); err != nil {
					return
				}
			case <-nctx.Done():
				return
			}
		}
	}()

	// Adapter: filter stdout PCM frames → out. Closes out when the filter's
	// Frames channel closes (clean EOF / process death) or on cancel.
	go func() {
		defer close(out)
		for {
			select {
			case f, ok := <-dec.Frames():
				if !ok {
					return
				}
				select {
				case out <- sttchain.AudioChunk{Audio: f}:
				case <-nctx.Done():
					return
				}
			case <-nctx.Done():
				return
			}
		}
	}()

	cleanup := func() {
		ncancel()
		_ = dec.Close()
	}
	return out, cleanup, nil
}

var (
	_ Enhancer     = DenoiseEnhancer{}
	_ FilterRunner = (*audioformat.Engine)(nil)
)
