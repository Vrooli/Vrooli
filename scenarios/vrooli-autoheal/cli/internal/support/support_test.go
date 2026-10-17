package support

import "testing"

func TestBuildQueryTrimsAndSkipsEmptyValues(t *testing.T) {
	query := BuildQuery(map[string]string{
		"checkId": "  disk  ",
		"empty":   "   ",
	})

	if got := query.Get("checkId"); got != "disk" {
		t.Fatalf("BuildQuery() checkId = %q, want disk", got)
	}
	if got := query.Get("empty"); got != "" {
		t.Fatalf("BuildQuery() empty = %q, want empty string", got)
	}
}

func TestStatusGlyphAndBoolWord(t *testing.T) {
	if got := StatusGlyph("healthy"); got != "OK" {
		t.Fatalf("StatusGlyph(healthy) = %q, want OK", got)
	}
	if got := StatusGlyph("warning"); got != "WARN" {
		t.Fatalf("StatusGlyph(warning) = %q, want WARN", got)
	}
	if got := BoolWord(true); got != "yes" {
		t.Fatalf("BoolWord(true) = %q, want yes", got)
	}
	if got := BoolWord(false); got != "no" {
		t.Fatalf("BoolWord(false) = %q, want no", got)
	}
}

func TestTruncateLines(t *testing.T) {
	got := TruncateLines("one\ntwo\nthree", 2)
	if got != "one\ntwo\n..." {
		t.Fatalf("TruncateLines() = %q, want %q", got, "one\ntwo\n...")
	}
}
