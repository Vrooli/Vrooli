package audioformat

// PrepareForWhisper converts batch / per-segment audio into a payload the
// Whisper HTTP endpoint accepts, returning the bytes and a filename whose
// extension matches the container (Whisper keys decoding off the upload's
// content, but a correct extension avoids ambiguity).
//
// Whisper decodes every real container (wav/mp3/ogg/webm/m4a/flac/aac)
// with its own bundled ffmpeg, so those pass through untouched — no local
// ffmpeg is needed for the batch path. The one representation Whisper
// cannot read is headerless PCM, so canonical PCM is wrapped in a native
// 44-byte WAV header (ffmpeg-free). An undeclared, unrecognized codec is
// rejected with ErrUnknownFormat rather than sent as corrupt bytes.
func (e *Engine) PrepareForWhisper(codec Codec, in []byte) (payload []byte, filename string, err error) {
	switch codec {
	case CodecPCMS16LE:
		return encodeWAV(in), "recording.wav", nil
	case CodecUnknown:
		return nil, "", ErrUnknownFormat
	default:
		return in, "recording." + containerExt(codec), nil
	}
}

// containerExt returns the conventional file extension for a container
// codec (used only to name the multipart upload field).
func containerExt(c Codec) string {
	switch c {
	case CodecWAV:
		return "wav"
	case CodecMP3:
		return "mp3"
	case CodecFLAC:
		return "flac"
	case CodecOGG, CodecOpus:
		return "ogg"
	case CodecWebM:
		return "webm"
	case CodecAAC:
		return "m4a"
	default:
		return "bin"
	}
}
