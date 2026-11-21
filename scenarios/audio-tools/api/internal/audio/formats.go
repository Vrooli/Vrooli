package audio

// Canonical AudioFormat short-codes (see handlers/audio/mappers.go) are
// mostly valid ffmpeg muxer/demuxer names, but not all of them. These
// helpers translate the exceptions once, at argv-construction time, so
// every operation and every caller stays consistent.
//
// Verified against ffmpeg 6.1 (`ffmpeg -hide_banner -muxers`):
// there is no "aac" muxer — raw AAC elementary streams are produced by
// the "adts" muxer (content type audio/aac). Likewise there is no
// "opus" demuxer — Opus always travels inside an Ogg container, which
// the "ogg" demuxer reads.

// muxerFor maps an AudioFormat short-code to the ffmpeg muxer that
// emits that format on stdout.
func muxerFor(format string) string {
	if format == "aac" {
		return "adts"
	}
	return format
}

// demuxerFor maps an AudioFormat short-code to the ffmpeg demuxer that
// reads that format from a file or pipe.
func demuxerFor(format string) string {
	if format == "opus" {
		return "ogg"
	}
	return format
}
