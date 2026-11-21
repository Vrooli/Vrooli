package pipeline

import (
	"strings"
	"unicode"
)

// NormalizeTranscriptText collapses the layout characters a transcription model
// may emit into the single-line, single-spaced form every dictation surface
// expects.
//
// Whisper emits a newline at each of its internal decode boundaries, so a
// segment can arrive as "Call me Ishmael.\n Some years ago". Nothing downstream
// stripped it: the ledger stored it, the wire carried it, and the browser
// inserted it verbatim into the composer, which is how dictating one paragraph
// produced a transcript broken across several lines for no reason the speaker
// could see. A newline is a rendering decision that belongs to the surface, not
// a property of speech, so the transport contract is single-line text.
//
// Word joins are preserved: a run of layout characters becomes exactly one
// space, never zero, so "off the\nspleen" cannot collapse into "off thespleen".
func NormalizeTranscriptText(text string) string {
	if text == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(text))
	pendingSpace := false
	wrote := false
	for _, r := range text {
		if r == '\n' || r == '\r' || r == '\t' || r == '\v' || r == '\f' || unicode.IsSpace(r) {
			// Defer the separator so trailing runs never reach the output.
			if wrote {
				pendingSpace = true
			}
			continue
		}
		if pendingSpace {
			b.WriteRune(' ')
			pendingSpace = false
		}
		b.WriteRune(r)
		wrote = true
	}
	return b.String()
}
