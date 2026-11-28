package filekind_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/vrooli/api-core/filekind"
)

func noSniff() ([]byte, error) { return nil, nil }

func TestClassifyByExtension(t *testing.T) {
	cases := []struct {
		name string
		want filekind.Kind
		mime string
	}{
		{"README.md", filekind.KindMarkdown, "text/markdown; charset=utf-8"},
		{"a.MDX", filekind.KindMarkdown, "text/markdown; charset=utf-8"},
		{"diagram.svg", filekind.KindSVG, "image/svg+xml"},
		{"shot.PNG", filekind.KindImage, "image/png"},
		{"photo.jpeg", filekind.KindImage, "image/jpeg"},
		{"doc.pdf", filekind.KindPDF, "application/pdf"},
		{"clip.mp4", filekind.KindVideo, "video/mp4"},
		{"clip.ogv", filekind.KindVideo, ""},
		{"song.ogg", filekind.KindAudio, ""},
		{"song.mp3", filekind.KindAudio, "audio/mpeg"},
		{"rows.csv", filekind.KindCSV, "text/csv; charset=utf-8"},
		{"rows.tsv", filekind.KindCSV, "text/csv; charset=utf-8"},
		{"letter.docx", filekind.KindDocument, "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"deck.pptx", filekind.KindPresentation, "application/vnd.openxmlformats-officedocument.presentationml.presentation"},
		{"deck.ppt", filekind.KindPresentation, "application/vnd.ms-powerpoint"},
		{"deck.ppsx", filekind.KindPresentation, "application/vnd.openxmlformats-officedocument.presentationml.slideshow"},
		{"deck.potx", filekind.KindPresentation, "application/vnd.openxmlformats-officedocument.presentationml.template"},
		{"book.epub", filekind.KindEbook, "application/epub+zip"},
		{"data.sqlite3", filekind.KindDatabase, "application/vnd.sqlite3"},
		{"face.woff2", filekind.KindFont, ""},
		{"part.glb", filekind.KindModel3D, ""},
		{"part.stl", filekind.KindModel3D, ""},
		{"bundle.tar", filekind.KindArchive, ""},
		{"bundle.zst", filekind.KindArchive, ""},
		{"sheet.xlsx", filekind.KindSpreadsheet, ""},
		{"change.patch", filekind.KindDiff, "text/x-diff; charset=utf-8"},
		{"main.go", filekind.KindCode, "text/plain; charset=utf-8"},
		{"page.html", filekind.KindCode, "text/plain; charset=utf-8"},
		{"notes.txt", filekind.KindText, "text/plain; charset=utf-8"},
		{"sub/dir/main.py", filekind.KindCode, "text/plain; charset=utf-8"},
	}
	for _, c := range cases {
		got := filekind.Classify(c.name, noSniff)
		if got.Kind != c.want {
			t.Errorf("%s: kind = %q, want %q", c.name, got.Kind, c.want)
		}
		if got.MIME == "" {
			t.Errorf("%s: empty MIME", c.name)
		}
		if c.mime != "" && got.MIME != c.mime {
			t.Errorf("%s: MIME = %q, want %q", c.name, got.MIME, c.mime)
		}
	}
}

func TestClassifyKnownExtensionNeverSniffs(t *testing.T) {
	called := false
	filekind.Classify("a.png", func() ([]byte, error) { called = true; return nil, nil })
	if called {
		t.Fatal("sniff was called for a known extension")
	}
}

func TestClassifyUnknownExtensionSniffs(t *testing.T) {
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0, 0, 0, 0}
	cases := []struct {
		name  string
		data  []byte
		err   error
		want  filekind.Kind
		mime  string
		sniff bool
	}{
		{"utf-8 text", []byte("hello world"), nil, filekind.KindText, "text/plain; charset=utf-8", true},
		{"empty is text", []byte{}, nil, filekind.KindText, "text/plain; charset=utf-8", true},
		{"unreadable", nil, errors.New("read failed"), filekind.KindUnsupported, "application/octet-stream", true},
		{"nul bytes are binary", []byte{0, 1, 2, 3}, nil, filekind.KindUnsupported, "application/octet-stream", true},
		{"invalid utf-8 is binary", []byte{0x80, 0x01, 0x81}, nil, filekind.KindUnsupported, "application/octet-stream", true},
		{"png magic", png, nil, filekind.KindImage, "image/png", true},
		{"pdf magic", []byte("%PDF-1.7\x00\x01\x02\xff"), nil, filekind.KindPDF, "application/pdf", true},
		{"sqlite magic", append([]byte("SQLite format 3\x00"), 0, 0, 0xff), nil, filekind.KindDatabase, "application/vnd.sqlite3", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := filekind.Classify("noext", func() ([]byte, error) { return c.data, c.err })
			if got.Kind != c.want || got.MIME != c.mime {
				t.Fatalf("got %+v, want kind %q mime %q", got, c.want, c.mime)
			}
		})
	}
}

func TestClassifyTextCutMidCharacterBySniffLimitIsStillText(t *testing.T) {
	// A sample that filled the sniff limit is a prefix of a longer file, so a
	// character cut by its end must not make valid text fall through to net/http's
	// signature matching, which would call this one a PDF.
	for _, tail := range [][]byte{{0xc3}, {0xe2, 0x82}} {
		sample := append([]byte("%PDF-"), []byte(strings.Repeat("a", filekind.SniffBytes-len("%PDF-")-len(tail)))...)
		sample = append(sample, tail...)
		got := filekind.Classify("noext", func() ([]byte, error) { return sample, nil })
		if got.Kind != filekind.KindText || got.MIME != "text/plain; charset=utf-8" {
			t.Fatalf("sample cut after %d bytes of a character: got %+v, want text/plain", len(tail), got)
		}
	}
}

func TestClassifySniffedHTMLIsServedAsPlainText(t *testing.T) {
	// HTML in a legacy encoding is not valid UTF-8, and net/http would call it
	// text/html. Only a known extension may make a file HTML.
	html := []byte("<!DOCTYPE html><html><body>caf\xe9 <script>alert(1)</script></body></html>")
	got := filekind.Classify("noext", func() ([]byte, error) { return html, nil })
	if got.Kind != filekind.KindText || got.MIME != "text/plain; charset=utf-8" {
		t.Fatalf("got %+v, want text served as text/plain", got)
	}
}

func TestClassifyNilSniffIsUnsupported(t *testing.T) {
	got := filekind.Classify("noext", nil)
	if got.Kind != filekind.KindUnsupported {
		t.Fatalf("kind = %q", got.Kind)
	}
}

func TestByExtension(t *testing.T) {
	if k, ok := filekind.ByExtension("a/b/c.MD"); !ok || k != filekind.KindMarkdown {
		t.Fatalf("got %q, %v", k, ok)
	}
	for _, name := range []string{"Makefile", "x.unknownext", "dir.png/file"} {
		if k, ok := filekind.ByExtension(name); ok {
			t.Errorf("%s: unexpectedly mapped to %q", name, k)
		}
	}
}

func TestIsMedia(t *testing.T) {
	for name, want := range map[string]bool{"a.mp4": true, "a.WAV": true, "a.png": false, "a.txt": false, "a": false} {
		if got := filekind.IsMedia(name); got != want {
			t.Errorf("IsMedia(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestMIMEForFallbacks(t *testing.T) {
	if got := filekind.MIMEFor(filekind.KindArchive, ".unregistered-archive-test"); got != "application/octet-stream" {
		t.Errorf("archive fallback = %q", got)
	}
	if got := filekind.MIMEFor(filekind.KindModel3D, ".unregistered-model-test"); got != "application/octet-stream" {
		t.Errorf("blob fallback = %q", got)
	}
	if got := filekind.MIMEFor(filekind.KindText, ".unregistered"); got != "text/plain; charset=utf-8" {
		t.Errorf("text fallback = %q", got)
	}
}

func TestKindCapabilities(t *testing.T) {
	cases := []struct {
		kind                     filekind.Kind
		text, blob, listing, can bool
	}{
		{filekind.KindMarkdown, true, false, false, true},
		{filekind.KindCSV, true, false, false, true},
		{filekind.KindImage, false, true, false, true},
		{filekind.KindVideo, false, true, false, true},
		{filekind.KindDirectory, false, false, true, true},
		{filekind.KindArchive, false, false, false, true},
		{filekind.KindUnsupported, false, false, false, false},
	}
	for _, c := range cases {
		if c.kind.TextContentAvailable() != c.text || c.kind.UsesBlob() != c.blob ||
			c.kind.ListingAvailable() != c.listing || c.kind.CanPreview() != c.can {
			t.Errorf("%s: capabilities do not match", c.kind)
		}
	}
}
