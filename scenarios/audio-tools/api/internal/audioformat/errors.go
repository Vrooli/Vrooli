package audioformat

import "errors"

// ErrUnknownFormat is returned by Detect when the codec was neither
// declared nor sniffable from the leading bytes. The substrate never
// guesses — an undeclared, unrecognized stream is a typed failure so the
// caller can reject it instead of feeding corrupt bytes to a strategy.
var ErrUnknownFormat = errors.New("audio-tools/audioformat: could not determine audio codec (declare input_format)")

// ErrFfmpegRequired is returned when a non-PCM codec must be decoded or a
// non-WAV container must be encoded but ffmpeg is unavailable. Callers map
// this to a capability decision (STT: BufferedFallback; TTS: FailedPrecondition).
var ErrFfmpegRequired = errors.New("audio-tools/audioformat: ffmpeg required for this codec")

// ErrUnsupportedOutput is returned by Encode for an output format outside
// the {mp3,wav,opus,flac} set.
var ErrUnsupportedOutput = errors.New("audio-tools/audioformat: unsupported output format")
