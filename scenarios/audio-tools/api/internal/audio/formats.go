package audio

// Canonical AudioFormat short-codes (see handlers/audio/mappers.go) mostly
// match ffmpeg muxer/demuxer names, but not all of them. These helpers
// translate the exceptions once, at argv-construction time, so every
// operation and every caller stays consistent.
//
// Verified against ffmpeg 6.1 (`ffmpeg -hide_banner -muxers`):
// there is no "aac" muxer — raw AAC elementary streams are produced by
// the "adts" muxer (content type audio/aac). Likewise there is no
// "opus" demuxer — Opus always travels inside an Ogg container, which
// the "ogg" demuxer reads.

// muxerFor maps an AudioFormat short-code to the ffmpeg muxer that
// emits that format on stdout.
func muxerFor(format string) string {
	switch format {
	case "aac":
		return "adts"
	case "pcm_s16le":
		return "s16le"
	default:
		return format
	}
}

// demuxerFor maps an AudioFormat short-code to the ffmpeg demuxer that
// reads that format from a file or pipe.
func demuxerFor(format string) string {
	switch format {
	case "opus":
		return "ogg"
	case "pcm_s16le":
		return "s16le"
	default:
		return format
	}
}
