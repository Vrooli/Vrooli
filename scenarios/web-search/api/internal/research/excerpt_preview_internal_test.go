package research

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

// TestExcerptPreviewRuneSafeTruncation pins the regression behind
// bug-inbox/code-defect/l2-excerpt-byte-truncation-invalid-utf8: the preview
// cap must never split a multi-byte rune, because an invalid-UTF-8 excerpt
// fails the proto marshal of the WHOLE RunL2Response.
func TestExcerptPreviewRuneSafeTruncation(t *testing.T) {
	// "é" is 2 bytes; 799 ASCII bytes + "é" places the rune across the
	// 800-byte cap, exactly the live failure shape.
	text := strings.Repeat("a", excerptPreviewChars-1) + "é" + strings.Repeat("b", 50)

	got := excerptPreview(text)

	require.True(t, utf8.ValidString(got), "truncated excerpt must stay valid UTF-8")
	require.LessOrEqual(t, len(got), excerptPreviewChars)
	require.Equal(t, strings.Repeat("a", excerptPreviewChars-1), got,
		"the rune straddling the cap must be dropped whole, not split")
}

// TestExcerptPreviewScrubsInvalidInputBytes covers the second hazard: fetched
// pages can carry bytes that were never valid UTF-8 (e.g. Latin-1), with no
// truncation involved.
func TestExcerptPreviewScrubsInvalidInputBytes(t *testing.T) {
	got := excerptPreview("caf\xe9 society")

	require.True(t, utf8.ValidString(got))
	require.Equal(t, "caf� society", got, "invalid bytes are replaced, surrounding text preserved")
}

// TestExcerptPreviewShortValidTextUntouched pins the happy path.
func TestExcerptPreviewShortValidTextUntouched(t *testing.T) {
	require.Equal(t, "héllo wörld", excerptPreview("héllo wörld"))
}

// TestExcerptsForResponseAlwaysMarshalable sweeps the projection helper:
// every produced excerpt must be proto-safe regardless of document content.
func TestExcerptsForResponseAlwaysMarshalable(t *testing.T) {
	docs := []Document{
		{URL: "https://a.example", Title: "A", Text: strings.Repeat("猫", 600)}, // 3-byte runes across the cap
		{URL: "https://b.example", Title: "B", Text: "plain ascii"},
		{URL: "https://c.example", Title: "C", Text: "broken \xff\xfe bytes"},
	}

	for _, e := range excerptsForResponse(docs) {
		require.True(t, utf8.ValidString(e.Excerpt), "excerpt for %s must be valid UTF-8", e.URL)
		require.LessOrEqual(t, len(e.Excerpt), excerptPreviewChars)
	}
}
