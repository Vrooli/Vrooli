package audioformat

import (
	"context"
	"encoding/binary"
	"fmt"

	"audio-tools/internal/protoint"
)

// Encode converts canonical engine PCM (16-bit LE, mono, 16 kHz) to the
// requested TTS output container, returning the encoded bytes and the
// matching content type.
//
//   - OutputWAV → native encode: a 44-byte RIFF/WAVE header is prepended,
//     no ffmpeg required (so WAV egress works even without ffmpeg).
//   - mp3/opus/flac → one-shot ffmpeg encode via the Runner seam; absent
//     ffmpeg returns ErrFfmpegRequired.
//   - any other format → ErrUnsupportedOutput.
func (e *Engine) Encode(ctx context.Context, pcm []byte, target OutputFormat) ([]byte, string, error) {
	switch target {
	case OutputWAV:
		return encodeWAV(pcm), OutputWAV.ContentType(), nil
	case OutputMP3, OutputOpus, OutputFLAC:
		if !e.hasFfmpeg() {
			return nil, "", ErrFfmpegRequired
		}
		out, err := e.runner.Run(ctx, "ffmpeg", pcm, encodeArgs(target)...)
		if err != nil {
			return nil, "", fmt.Errorf("audio-tools/audioformat: encode %s: %w", target, err)
		}
		return out, target.ContentType(), nil
	default:
		return nil, "", ErrUnsupportedOutput
	}
}

// encodeArgs builds the ffmpeg argv that reads canonical PCM from stdin
// and writes the target container to stdout.
func encodeArgs(target OutputFormat) []string {
	args := []string{
		"-y", "-loglevel", "error",
		"-f", "s16le",
		"-ar", fmt.Sprint(CanonicalSampleRate),
		"-ac", fmt.Sprint(CanonicalChannels),
		"-i", "pipe:0",
	}
	switch target {
	case OutputMP3:
		args = append(args, "-f", "mp3")
	case OutputOpus:
		// libopus in an Ogg container; content type is audio/ogg.
		args = append(args, "-c:a", "libopus", "-f", "opus")
	case OutputFLAC:
		args = append(args, "-f", "flac")
	}
	return append(args, "pipe:1")
}

// WAVFromCanonicalPCM wraps raw canonical PCM (s16le/16k/mono) in a 44-byte
// WAV header so consumers that decode via container sniffing (e.g. the
// speaker-verification resource, which uses torchaudio/ffmpeg) can read it.
//
// This is UNCONDITIONAL by design: raw PCM has no header and cannot be
// sniffed (see Detect's doc), and a PCM sample can legitimately begin with
// bytes that collide with container magic (e.g. 0xFFFF reads as an MPEG frame
// sync). Callers must therefore pass canonical PCM, never an already-
// containerized blob — which holds for every egress speaker-stage caller.
func WAVFromCanonicalPCM(pcm []byte) []byte { return encodeWAV(pcm) }

// encodeWAV prepends a canonical 44-byte PCM WAV header to raw s16le bytes.
func encodeWAV(pcm []byte) []byte {
	const (
		bitsPerSample = CanonicalBytesPerSample * 8
		byteRate      = CanonicalSampleRate * CanonicalChannels * CanonicalBytesPerSample
		blockAlign    = CanonicalChannels * CanonicalBytesPerSample
	)
	dataLen := protoint.FromIntToUint32(len(pcm))
	out := make([]byte, 0, 44+len(pcm))
	out = append(out, "RIFF"...)
	out = binary.LittleEndian.AppendUint32(out, 36+dataLen) // chunk size
	out = append(out, "WAVE"...)
	out = append(out, "fmt "...)
	out = binary.LittleEndian.AppendUint32(out, 16) // fmt chunk size
	out = binary.LittleEndian.AppendUint16(out, 1)  // PCM
	out = binary.LittleEndian.AppendUint16(out, CanonicalChannels)
	out = binary.LittleEndian.AppendUint32(out, CanonicalSampleRate)
	out = binary.LittleEndian.AppendUint32(out, byteRate)
	out = binary.LittleEndian.AppendUint16(out, blockAlign)
	out = binary.LittleEndian.AppendUint16(out, bitsPerSample)
	out = append(out, "data"...)
	out = binary.LittleEndian.AppendUint32(out, dataLen)
	return append(out, pcm...)
}
