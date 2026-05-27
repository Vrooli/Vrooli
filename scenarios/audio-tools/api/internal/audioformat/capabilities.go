package audioformat

// Accepts returns the codecs the engine can normalize to canonical PCM
// for ingress. CodecPCMS16LE is always accepted (fast-path, no ffmpeg);
// every container codec requires ffmpeg, so without it only PCM is
// accepted. The list is sorted by Codec value for stable output.
func (e *Engine) Accepts() []Codec {
	out := []Codec{CodecPCMS16LE}
	if e.hasFfmpeg() {
		out = append(out, CodecWAV, CodecMP3, CodecFLAC, CodecOGG, CodecWebM, CodecOpus, CodecAAC)
	}
	return out
}

// Emits returns the TTS output formats the engine can produce for egress.
// WAV is always emittable (native header, no ffmpeg); mp3/opus/flac
// require ffmpeg.
func (e *Engine) Emits() []OutputFormat {
	if e.hasFfmpeg() {
		return []OutputFormat{OutputMP3, OutputWAV, OutputOpus, OutputFLAC}
	}
	return []OutputFormat{OutputWAV}
}

// CanEmit reports whether the engine can currently produce f.
func (e *Engine) CanEmit(f OutputFormat) bool {
	for _, c := range e.Emits() {
		if c == f {
			return true
		}
	}
	return false
}
