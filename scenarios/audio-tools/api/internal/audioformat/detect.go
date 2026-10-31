package audioformat

import "bytes"

// Detect resolves the codec of an audio stream. A declared codec always
// wins (declare-first contract); when the caller passes CodecUnknown,
// Detect sniffs the leading bytes by container magic number.
//
// Raw PCM cannot be sniffed (it has no header) — it is only ever
// recognized when declared. An undeclared, unrecognized stream returns
// ErrUnknownFormat; the substrate never silently guesses.
func Detect(declared Codec, head []byte) (Codec, error) {
	if declared != CodecUnknown {
		return declared, nil
	}
	if c := sniff(head); c != CodecUnknown {
		return c, nil
	}
	return CodecUnknown, ErrUnknownFormat
}

// sniff inspects container magic bytes. Order matters where prefixes
// overlap (e.g. ID3-tagged MP3 vs. raw frame sync).
func sniff(b []byte) Codec {
	switch {
	case len(b) >= 4 && bytes.Equal(b[:4], []byte{0x1A, 0x45, 0xDF, 0xA3}):
		// EBML header — WebM/Matroska.
		return CodecWebM
	case len(b) >= 4 && bytes.Equal(b[:4], []byte("OggS")):
		return CodecOGG
	case len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WAVE")):
		return CodecWAV
	case len(b) >= 4 && bytes.Equal(b[:4], []byte("fLaC")):
		return CodecFLAC
	case len(b) >= 3 && bytes.Equal(b[:3], []byte("ID3")):
		return CodecMP3
	case len(b) >= 8 && bytes.Equal(b[4:8], []byte("ftyp")):
		// ISO base media — M4A/AAC-in-MP4. Whisper decodes via ffmpeg as AAC.
		return CodecAAC
	case len(b) >= 2 && b[0] == 0xFF && (b[1]&0xF6) == 0xF0:
		// ADTS AAC frame sync (0xFFFx, layer bits 0).
		return CodecAAC
	case len(b) >= 2 && b[0] == 0xFF && (b[1]&0xE0) == 0xE0:
		// MPEG audio frame sync — bare MP3.
		return CodecMP3
	default:
		return CodecUnknown
	}
}
