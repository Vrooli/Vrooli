// Package audioformat is the single owner of audio-format handling in
// audio-tools. Every audio entry/exit point routes through it:
//
//   - INGRESS (STT): declare-or-detect the inbound codec, then normalize
//     to the canonical STT representation (16-bit LE PCM, mono, 16 kHz) —
//     one-shot for whole-file batch, streamingly (one long-lived ffmpeg
//     per session) for live transports. A PCM fast-path skips ffmpeg.
//   - EGRESS (TTS): encode canonical engine PCM to the requested output
//     container (mp3/wav/opus/flac), guaranteed.
//
// The package is transport- and provider-agnostic: it knows nothing
// about WebSockets, Connect, Whisper, or kokoro. It owns the ffmpeg
// argv and the codec sniffing so no other package re-implements either.
//
// seam: the streaming decode subprocess is the audioformat.ProcessRunner
// seam (SEAMS.md). Production wires execProcessRunner; tests wire
// internal/audioformat/mocks.FakeProcessRunner. The one-shot ffmpeg runs
// through the existing audio.Runner seam.
package audioformat

import (
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
)

// Canonical STT internal representation. This is FIXED, not caller-tunable:
// the VAD RMS math and Whisper both depend on 16-bit signed little-endian
// PCM at 16 kHz mono. An inbound sample-rate hint describes the bytes the
// client sends; it never changes this target.
const (
	CanonicalSampleRate     = 16000
	CanonicalChannels       = 1
	CanonicalBytesPerSample = 2 // s16le
)

// Codec identifies a container/codec the substrate can detect, decode, or
// (for the canonical PCM value) pass through untouched.
type Codec int

const (
	// CodecUnknown is the zero value: codec not declared and not sniffable.
	CodecUnknown Codec = iota
	// CodecPCMS16LE is raw 16-bit LE PCM with no container — the canonical
	// STT representation. Declaring it takes the ffmpeg-free fast-path.
	CodecPCMS16LE
	CodecWAV
	CodecMP3
	CodecFLAC
	CodecOGG
	CodecWebM
	CodecOpus
	CodecAAC
)

// AllInputCodecs returns every input codec this build understands, in
// stable Codec order. It is the format vocabulary the STT API accepts:
// batch transcription always decodes these (Whisper has its own decoder),
// and live streaming decodes them when ffmpeg is present. Engine.Accepts
// reports the narrower ffmpeg-gated subset the substrate can normalize
// itself for the live path.
func AllInputCodecs() []Codec {
	return []Codec{CodecPCMS16LE, CodecWAV, CodecMP3, CodecFLAC, CodecOGG, CodecWebM, CodecOpus, CodecAAC}
}

// IsCanonicalPCM reports whether c is the raw-PCM fast-path codec (no
// container, no decode required).
func (c Codec) IsCanonicalPCM() bool { return c == CodecPCMS16LE }

// String returns the lower-case wire vocabulary used across the internal
// chains (sttchain.Request.Format, the WS `format` query param, etc.).
func (c Codec) String() string {
	switch c {
	case CodecPCMS16LE:
		return "pcm_s16le"
	case CodecWAV:
		return "wav"
	case CodecMP3:
		return "mp3"
	case CodecFLAC:
		return "flac"
	case CodecOGG:
		return "ogg"
	case CodecWebM:
		return "webm"
	case CodecOpus:
		return "opus"
	case CodecAAC:
		return "aac"
	default:
		return ""
	}
}

// CodecFromString maps the internal wire vocabulary to a Codec. An empty
// or unrecognized string maps to CodecUnknown so callers fall back to
// sniffing rather than guessing.
func CodecFromString(s string) Codec {
	switch s {
	case "pcm_s16le", "pcm":
		return CodecPCMS16LE
	case "wav":
		return CodecWAV
	case "mp3":
		return CodecMP3
	case "flac":
		return CodecFLAC
	case "ogg":
		return CodecOGG
	case "webm":
		return CodecWebM
	case "opus":
		return CodecOpus
	case "aac":
		return CodecAAC
	default:
		return CodecUnknown
	}
}

// FromProto maps a common.AudioFormat enum value to a Codec. The bool is
// false for AUDIO_FORMAT_UNSPECIFIED (the caller should sniff instead).
func FromProto(f commonv1.AudioFormat) (Codec, bool) {
	switch f {
	case commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE:
		return CodecPCMS16LE, true
	case commonv1.AudioFormat_AUDIO_FORMAT_WAV:
		return CodecWAV, true
	case commonv1.AudioFormat_AUDIO_FORMAT_MP3:
		return CodecMP3, true
	case commonv1.AudioFormat_AUDIO_FORMAT_FLAC:
		return CodecFLAC, true
	case commonv1.AudioFormat_AUDIO_FORMAT_OGG:
		return CodecOGG, true
	case commonv1.AudioFormat_AUDIO_FORMAT_WEBM:
		return CodecWebM, true
	case commonv1.AudioFormat_AUDIO_FORMAT_OPUS:
		return CodecOpus, true
	case commonv1.AudioFormat_AUDIO_FORMAT_AAC:
		return CodecAAC, true
	default:
		return CodecUnknown, false
	}
}

// ToProto maps a Codec back to its common.AudioFormat enum value.
func ToProto(c Codec) commonv1.AudioFormat {
	switch c {
	case CodecPCMS16LE:
		return commonv1.AudioFormat_AUDIO_FORMAT_PCM_S16LE
	case CodecWAV:
		return commonv1.AudioFormat_AUDIO_FORMAT_WAV
	case CodecMP3:
		return commonv1.AudioFormat_AUDIO_FORMAT_MP3
	case CodecFLAC:
		return commonv1.AudioFormat_AUDIO_FORMAT_FLAC
	case CodecOGG:
		return commonv1.AudioFormat_AUDIO_FORMAT_OGG
	case CodecWebM:
		return commonv1.AudioFormat_AUDIO_FORMAT_WEBM
	case CodecOpus:
		return commonv1.AudioFormat_AUDIO_FORMAT_OPUS
	case CodecAAC:
		return commonv1.AudioFormat_AUDIO_FORMAT_AAC
	default:
		return commonv1.AudioFormat_AUDIO_FORMAT_UNSPECIFIED
	}
}
