// Package filekind decides how a file should be previewed: which viewer kind it
// belongs to and which Content-Type it should be served with. It is the one
// classifier shared by every app that previews files (Web Console, Notification
// Hub), so the same file opens in the same viewer everywhere.
//
// The package is pure standard library and does no I/O of its own. Callers that
// can read content pass a sniff function; Classify only calls it for names whose
// extension is not in the registry, so well-known artifacts never cost a read.
package filekind

import (
	"mime"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"
)

// Kind classifies how a file is previewed. Every previewable file maps to
// exactly one Kind; viewer registries are keyed by these values.
type Kind string

// The preview kinds. The string values are a wire contract (UI renderer
// registries and API fields use them), so they never change.
const (
	KindMarkdown     Kind = "markdown"
	KindCode         Kind = "code"
	KindText         Kind = "text"
	KindSVG          Kind = "svg"
	KindImage        Kind = "image"
	KindPDF          Kind = "pdf"
	KindAudio        Kind = "audio"
	KindVideo        Kind = "video"
	KindCSV          Kind = "csv"
	KindSpreadsheet  Kind = "spreadsheet"
	KindDocument     Kind = "document"
	KindPresentation Kind = "presentation"
	KindEbook        Kind = "ebook"
	KindModel3D      Kind = "model3d"
	KindFont         Kind = "font"
	KindDatabase     Kind = "database"
	KindArchive      Kind = "archive"
	KindDiff         Kind = "diff"
	KindDirectory    Kind = "directory"
	KindUnsupported  Kind = "unsupported"
)

// MIMEDirectory is the conventional media type reported for a directory.
// Directories have no bytes; it only labels the target.
const MIMEDirectory = "inode/directory"

// SniffBytes is how many leading bytes a sniff function should return. Content
// detection never looks further than this.
const SniffBytes = 512

const (
	mimeOctetStream = "application/octet-stream"
	mimeSQLite      = "application/vnd.sqlite3"
	mimePlainText   = "text/plain; charset=utf-8"
)

// textKinds render from bounded UTF-8 content.
var textKinds = map[Kind]bool{
	KindMarkdown: true,
	KindCode:     true,
	KindText:     true,
	KindCSV:      true,
	KindDiff:     true,
}

// blobKinds render from an HTTP byte stream with range support. SVG is a blob
// (image/svg+xml) so viewers can render it from a blob URL instead of injecting
// it into the DOM, which keeps it XSS-safe.
var blobKinds = map[Kind]bool{
	KindSVG:          true,
	KindImage:        true,
	KindPDF:          true,
	KindSpreadsheet:  true,
	KindDocument:     true,
	KindPresentation: true,
	KindEbook:        true,
	KindModel3D:      true,
	KindFont:         true,
	KindDatabase:     true,
	KindAudio:        true,
	KindVideo:        true,
}

// listingKinds render from a bounded, paginated listing rather than from text or
// a byte stream.
var listingKinds = map[Kind]bool{
	KindDirectory: true,
}

// TextContentAvailable reports whether the kind renders from bounded UTF-8 text.
func (k Kind) TextContentAvailable() bool { return textKinds[k] }

// UsesBlob reports whether the kind renders from a byte stream.
func (k Kind) UsesBlob() bool { return blobKinds[k] }

// ListingAvailable reports whether the kind renders from a directory listing.
func (k Kind) ListingAvailable() bool { return listingKinds[k] }

// CanPreview reports whether the kind has a dedicated viewer (anything but
// KindUnsupported).
func (k Kind) CanPreview() bool { return k != KindUnsupported }

// extKind maps a lower-case extension (with its leading dot) to a Kind. .ogg is
// deliberately audio (the common case for generated clips); video uses the
// unambiguous .ogv.
var extKind = map[string]Kind{
	".md": KindMarkdown, ".mdx": KindMarkdown, ".markdown": KindMarkdown,
	".svg": KindSVG,
	".png": KindImage, ".jpg": KindImage, ".jpeg": KindImage, ".gif": KindImage,
	".webp": KindImage, ".bmp": KindImage, ".ico": KindImage, ".avif": KindImage,
	".tiff": KindImage, ".tif": KindImage,
	".pdf": KindPDF,
	".mp4": KindVideo, ".webm": KindVideo, ".ogv": KindVideo, ".mov": KindVideo, ".m4v": KindVideo,
	".mp3": KindAudio, ".wav": KindAudio, ".ogg": KindAudio, ".oga": KindAudio,
	".m4a": KindAudio, ".aac": KindAudio, ".flac": KindAudio, ".opus": KindAudio,
	".csv": KindCSV, ".tsv": KindCSV,
	".docx": KindDocument,
	".pptx": KindPresentation, ".ppsx": KindPresentation, ".potx": KindPresentation, ".ppt": KindPresentation,
	".epub": KindEbook,
	".db":   KindDatabase, ".sqlite": KindDatabase, ".sqlite3": KindDatabase,
	".ttf": KindFont, ".otf": KindFont, ".woff": KindFont, ".woff2": KindFont,
	".glb": KindModel3D, ".gltf": KindModel3D, ".obj": KindModel3D, ".stl": KindModel3D,
	".zip": KindArchive, ".tar": KindArchive, ".tgz": KindArchive, ".gz": KindArchive, ".7z": KindArchive, ".rar": KindArchive, ".xz": KindArchive, ".zst": KindArchive,
	".xlsx": KindSpreadsheet, ".xls": KindSpreadsheet, ".xlsm": KindSpreadsheet, ".xlsb": KindSpreadsheet,
	".diff": KindDiff, ".patch": KindDiff,
	".go": KindCode, ".ts": KindCode, ".tsx": KindCode, ".js": KindCode, ".jsx": KindCode,
	".json": KindCode, ".yml": KindCode, ".yaml": KindCode, ".sh": KindCode, ".bash": KindCode,
	".zsh": KindCode, ".sql": KindCode, ".css": KindCode, ".scss": KindCode, ".html": KindCode,
	".txt": KindText, ".proto": KindCode, ".toml": KindCode, ".ini": KindCode, ".env": KindCode,
	".py": KindCode, ".rs": KindCode, ".rb": KindCode, ".java": KindCode, ".c": KindCode,
	".h": KindCode, ".cpp": KindCode, ".hpp": KindCode, ".xml": KindCode, ".mod": KindCode,
}

// mimeByKind is the canonical Content-Type for kinds whose extension has no
// registered MIME, or whose registered MIME is overridden so every layer agrees
// on charset.
var mimeByKind = map[Kind]string{
	KindDatabase:     mimeSQLite,
	KindMarkdown:     "text/markdown; charset=utf-8",
	KindCode:         mimePlainText,
	KindText:         mimePlainText,
	KindCSV:          "text/csv; charset=utf-8",
	KindDiff:         "text/x-diff; charset=utf-8",
	KindSVG:          "image/svg+xml",
	KindDocument:     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	KindPresentation: "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	KindEbook:        "application/epub+zip",
}

// Result is the outcome of classifying one file.
type Result struct {
	Kind Kind
	// MIME is the Content-Type to serve the file with.
	MIME string
}

// Classify determines the Kind and MIME type of a file from its name and, for
// names whose extension is not in the registry, a content sniff. sniff returns
// up to SniffBytes leading bytes; it is called at most once and never for a
// known extension. A nil sniff is treated as an unreadable file.
//
// An unreadable unknown file is KindUnsupported, an empty one is KindText, and
// valid UTF-8 without NUL bytes is KindText. Binary content is recognized by
// its magic bytes (SQLite, then net/http content detection).
func Classify(name string, sniff func() ([]byte, error)) Result {
	ext := extension(name)
	if k, ok := extKind[ext]; ok {
		return Result{Kind: k, MIME: MIMEFor(k, ext)}
	}
	if sniff == nil {
		return Result{Kind: KindUnsupported, MIME: mimeOctetStream}
	}
	data, err := sniff()
	if err != nil {
		return Result{Kind: KindUnsupported, MIME: mimeOctetStream}
	}
	if len(data) == 0 {
		return Result{Kind: KindText, MIME: mimePlainText}
	}
	if isProbablyText(data) {
		return Result{Kind: KindText, MIME: mimePlainText}
	}
	if len(data) >= 16 && string(data[:16]) == "SQLite format 3\x00" {
		return Result{Kind: KindDatabase, MIME: mimeSQLite}
	}
	detected := http.DetectContentType(data)
	kind := kindFromMIME(detected)
	if kind == KindText {
		// A text sniff (text/html included, for HTML in a legacy encoding) is shown
		// as text and served as text; only a known extension makes a file HTML.
		return Result{Kind: KindText, MIME: mimePlainText}
	}
	return Result{Kind: kind, MIME: baseMIME(detected)}
}

// ByExtension returns the Kind implied by a name's extension alone, with
// ok=false when the extension is unmapped. Listings use it so a page of entries
// costs no file reads; callers should present an unmapped extension as "kind
// determined on open" rather than guess.
func ByExtension(name string) (Kind, bool) {
	k, ok := extKind[extension(name)]
	return k, ok
}

// IsMedia reports whether the name has an audio or video extension.
func IsMedia(name string) bool {
	k, ok := ByExtension(name)
	return ok && (k == KindAudio || k == KindVideo)
}

// MIMEFor resolves the Content-Type for a kind and a lower-case extension with
// its leading dot. A kind with a canonical override uses it; otherwise the
// registered extension MIME wins, falling back to application/octet-stream for
// byte-stream kinds and archives, and text/plain for text kinds.
func MIMEFor(k Kind, ext string) string {
	if k == KindPresentation {
		switch ext {
		case ".ppt":
			return "application/vnd.ms-powerpoint"
		case ".ppsx":
			return "application/vnd.openxmlformats-officedocument.presentationml.slideshow"
		case ".potx":
			return "application/vnd.openxmlformats-officedocument.presentationml.template"
		}
	}
	if m, ok := mimeByKind[k]; ok {
		return m
	}
	if m := mime.TypeByExtension(ext); m != "" {
		return m
	}
	if k.UsesBlob() || k == KindArchive {
		return mimeOctetStream
	}
	return mimePlainText
}

// extension returns the lower-case extension of name's final slash-separated
// element, with its leading dot, or "".
func extension(name string) string {
	return strings.ToLower(path.Ext(name))
}

// kindFromMIME maps a sniffed MIME type onto a Kind for unknown extensions
// (for example an extensionless PNG).
func kindFromMIME(m string) Kind {
	base := baseMIME(m)
	switch {
	case base == "application/pdf":
		return KindPDF
	case base == "image/svg+xml":
		return KindSVG
	case strings.HasPrefix(base, "image/"):
		return KindImage
	case strings.HasPrefix(base, "audio/"):
		return KindAudio
	case strings.HasPrefix(base, "video/"):
		return KindVideo
	case strings.HasPrefix(base, "text/"):
		return KindText
	}
	return KindUnsupported
}

// baseMIME strips parameters (for example "; charset=utf-8").
func baseMIME(m string) string {
	if i := strings.IndexByte(m, ';'); i >= 0 {
		return strings.TrimSpace(m[:i])
	}
	return strings.TrimSpace(m)
}

// isProbablyText reports whether the sample is valid UTF-8 with no NUL bytes.
// A sample that filled SniffBytes is a prefix of a longer file, so a multi-byte
// character cut by its end is not evidence of binary content.
func isProbablyText(data []byte) bool {
	if len(data) >= SniffBytes {
		data = trimPartialRune(data)
	}
	if !utf8.Valid(data) {
		return false
	}
	for _, b := range data {
		if b == 0 {
			return false
		}
	}
	return true
}

// trimPartialRune drops a trailing, incomplete UTF-8 sequence (at most three
// bytes) so a sample cut in the middle of a character can still be validated.
func trimPartialRune(data []byte) []byte {
	for back := 1; back < utf8.UTFMax && back <= len(data); back++ {
		b := data[len(data)-back]
		if b < utf8.RuneSelf {
			return data
		}
		if utf8.RuneStart(b) {
			if !utf8.FullRune(data[len(data)-back:]) {
				return data[:len(data)-back]
			}
			return data
		}
	}
	return data
}
