// Package filepreview is the reusable file-preview subsystem for web-console.
// It resolves a raw path string (from a message link or inline code chip) into
// rich preview metadata, classifies it into a preview Kind, and issues opaque
// short-lived preview ids the REST blob endpoint serves bytes against.
//
// The package is transport-neutral: it has no knowledge of Connect, mux, or
// the session store. Package main adapts the live session cwd into Resolve and
// wires the Connect handler + blob route around Store. This keeps path
// resolution, classification, and the id store independently unit-testable and
// reusable by future web-console surfaces (records, BAS runs, plan drill-downs)
// without dragging conversation semantics along.
package filepreview

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Kind classifies how a resolved file should be previewed in the UI. Every
// resolved, previewable file maps to exactly one Kind; the UI renderer registry
// is keyed by these values.
type Kind string

const (
	KindMarkdown    Kind = "markdown"
	KindCode        Kind = "code"
	KindText        Kind = "text"
	KindSVG         Kind = "svg"
	KindImage       Kind = "image"
	KindPDF         Kind = "pdf"
	KindAudio       Kind = "audio"
	KindVideo       Kind = "video"
	KindCSV         Kind = "csv"
	KindDiff        Kind = "diff"
	KindUnsupported Kind = "unsupported"
)

// textKinds render from bounded UTF-8 content fetched over Connect-RPC.
var textKinds = map[Kind]bool{
	KindMarkdown: true,
	KindCode:     true,
	KindText:     true,
	KindCSV:      true,
	KindDiff:     true,
}

// blobKinds render from the HTTP blob endpoint (opaque bytes, range support).
// SVG is served as a blob (image/svg+xml) and rendered via a blob URL rather
// than injected into the DOM, keeping it XSS-safe.
var blobKinds = map[Kind]bool{
	KindSVG:   true,
	KindImage: true,
	KindPDF:   true,
	KindAudio: true,
	KindVideo: true,
}

// TextContentAvailable reports whether GetTextContent can serve this kind.
func (k Kind) TextContentAvailable() bool { return textKinds[k] }

// UsesBlob reports whether the kind renders from the HTTP blob endpoint.
func (k Kind) UsesBlob() bool { return blobKinds[k] }

// CanPreview reports whether the kind has a dedicated renderer (anything but
// KindUnsupported).
func (k Kind) CanPreview() bool { return k != KindUnsupported }

// extKind maps a lower-case extension (including the leading dot) to a Kind.
// .ogg is intentionally treated as audio (the common case for agent-generated
// clips); video uses the unambiguous .ogv.
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
	".diff": KindDiff, ".patch": KindDiff,
	".go": KindCode, ".ts": KindCode, ".tsx": KindCode, ".js": KindCode, ".jsx": KindCode,
	".json": KindCode, ".yml": KindCode, ".yaml": KindCode, ".sh": KindCode, ".bash": KindCode,
	".zsh": KindCode, ".sql": KindCode, ".css": KindCode, ".scss": KindCode, ".html": KindCode,
	".txt": KindText, ".proto": KindCode, ".toml": KindCode, ".ini": KindCode, ".env": KindCode,
	".py": KindCode, ".rs": KindCode, ".rb": KindCode, ".java": KindCode, ".c": KindCode,
	".h": KindCode, ".cpp": KindCode, ".hpp": KindCode, ".xml": KindCode, ".mod": KindCode,
}

// mimeByKind gives the canonical Content-Type for text-rendered kinds whose
// extension has no registered MIME or whose registered MIME we want to override
// (markdown, csv) so the blob/Connect layers agree on charset.
var mimeByKind = map[Kind]string{
	KindMarkdown: "text/markdown; charset=utf-8",
	KindCode:     "text/plain; charset=utf-8",
	KindText:     "text/plain; charset=utf-8",
	KindCSV:      "text/csv; charset=utf-8",
	KindDiff:     "text/x-diff; charset=utf-8",
	KindSVG:      "image/svg+xml",
}

// classification is the result of inspecting a file's extension and (for
// unknown extensions) a content sniff.
type classification struct {
	kind     Kind
	mimeType string
}

// classify determines the preview Kind and MIME type for a resolved file.
// sniff supplies up to 512 leading bytes for content detection; it is only
// called for extensions with no Kind mapping (so well-known agent artifacts
// never pay a read).
func classify(path string, sniff func() ([]byte, error)) classification {
	ext := strings.ToLower(filepath.Ext(path))
	if k, ok := extKind[ext]; ok {
		return classification{kind: k, mimeType: mimeForKind(k, ext)}
	}

	// Unknown extension: sniff content to separate UTF-8 text from binary.
	data, err := sniff()
	if err != nil || len(data) == 0 {
		// Unreadable or empty — treat empty as text, unreadable as unsupported.
		if err != nil {
			return classification{kind: KindUnsupported, mimeType: "application/octet-stream"}
		}
		return classification{kind: KindText, mimeType: "text/plain; charset=utf-8"}
	}
	if isProbablyText(data) {
		return classification{kind: KindText, mimeType: "text/plain; charset=utf-8"}
	}
	detected := http.DetectContentType(data)
	if k := kindFromMIME(detected); k != KindUnsupported {
		return classification{kind: k, mimeType: cleanMIME(detected)}
	}
	return classification{kind: KindUnsupported, mimeType: cleanMIME(detected)}
}

// mimeForKind resolves the Content-Type for a kind+extension. Kinds with a
// canonical override use it; otherwise the registered extension MIME wins, with
// an octet-stream fallback for blob kinds and text/plain for text kinds.
func mimeForKind(k Kind, ext string) string {
	if m, ok := mimeByKind[k]; ok {
		return m
	}
	if m := mime.TypeByExtension(ext); m != "" {
		return m
	}
	if k.UsesBlob() {
		return "application/octet-stream"
	}
	return "text/plain; charset=utf-8"
}

// kindFromMIME maps a sniffed MIME type onto a blob Kind for unknown
// extensions (e.g. an extensionless PNG).
func kindFromMIME(m string) Kind {
	base := cleanMIME(m)
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

// cleanMIME strips parameters (e.g. "; charset=utf-8") for comparison.
func cleanMIME(m string) string {
	if i := strings.IndexByte(m, ';'); i >= 0 {
		return strings.TrimSpace(m[:i])
	}
	return strings.TrimSpace(m)
}

// isProbablyText reports whether the sample is valid UTF-8 with no NUL bytes.
func isProbablyText(data []byte) bool {
	if !utf8.Valid(data) {
		return false
	}
	for _, b := range data {
		if b == 0 {
			return false // NUL byte → binary
		}
	}
	return true
}
