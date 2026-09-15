package pipeline

import "testing"

func TestNormalizeTranscriptTextStripsModelLineBreaks(t *testing.T) {
	// The exact shape Whisper delivered for one spoken paragraph. Every "\n"
	// here reached the browser composer verbatim before normalization existed.
	const whisperSegment = "Call me Ishmael.\n Some years ago, never mind how long precisely, having\n little or no money in my purse and\n nothing particular to interest me on shore"
	const want = "Call me Ishmael. Some years ago, never mind how long precisely, having little or no money in my purse and nothing particular to interest me on shore"
	if got := NormalizeTranscriptText(whisperSegment); got != want {
		t.Fatalf("NormalizeTranscriptText() =\n%q\nwant\n%q", got, want)
	}
}

func TestNormalizeTranscriptTextNeverJoinsWords(t *testing.T) {
	// A newline between two words is a word separator. Deleting it rather than
	// replacing it would silently corrupt the transcript instead of tidying it.
	if got := NormalizeTranscriptText("off the\nspleen"); got != "off the spleen" {
		t.Fatalf("NormalizeTranscriptText() = %q, want %q", got, "off the spleen")
	}
}

func TestNormalizeTranscriptTextTable(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"empty", "", ""},
		{"already clean", "Call me Ishmael.", "Call me Ishmael."},
		{"crlf", "one\r\ntwo", "one two"},
		{"tabs", "one\t\ttwo", "one two"},
		{"collapses runs", "one   \n\n  two", "one two"},
		{"trims both ends", "\n  Call me Ishmael. \n", "Call me Ishmael."},
		{"whitespace only", " \n\t ", ""},
		{"nbsp is whitespace", "one two", "one two"},
		// Punctuation attachment is the rendering surface's job (see
		// transcriptBuffer.transcriptSeparator); this layer only guarantees that a
		// run of layout characters becomes exactly one space.
		{"punctuation spacing is left to the surface", "wait\n, then go", "wait , then go"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeTranscriptText(tc.in); got != tc.want {
				t.Fatalf("NormalizeTranscriptText(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
