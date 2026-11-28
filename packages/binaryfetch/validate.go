// Package binaryfetch downloads, verifies, and installs single binaries (or
// binaries packaged inside an archive) fetched from a URL. It is the shared home
// for the artifact-validation logic that was previously trapped scenario-local
// in image-tools (HTML-page rejection, truncated-download rejection,
// magic-byte sniffing) so any module — the platform host-tool handler and the
// image-tools model installer alike — validates downloads identically.
//
// It deliberately depends only on the standard library so both the root module
// and scenario modules can import it across the module boundary.
package binaryfetch

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// SniffLen is how many leading bytes content-type detection and format magic
// consult. http.DetectContentType only reads the first 512 bytes.
const SniffLen = 512

// DefaultSizeFloor is the hard minimum any real binary/weight must exceed. A few
// hundred bytes is always a stub, an error JSON, or a redirect/landing page.
const DefaultSizeFloor int64 = 1 << 10 // 1 KiB

// ErrLooksLikeHTML is returned when a downloaded artifact sniffs as an HTML page
// — the signature of a release/landing page served where a binary was expected.
var ErrLooksLikeHTML = errors.New("binaryfetch: downloaded artifact looks like an HTML page")

// ErrTooSmall is returned when an artifact is below its minimum size — the
// signature of a truncated download or an error page.
var ErrTooSmall = errors.New("binaryfetch: downloaded artifact is smaller than expected")

// ErrChecksumMismatch is returned when the downloaded bytes do not match the
// declared SHA-256.
var ErrChecksumMismatch = errors.New("binaryfetch: sha256 checksum mismatch")

// ReadHead returns up to n leading bytes of the file at path. It returns the
// bytes actually read (which may be fewer than n for short files).
func ReadHead(path string, n int) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // path is caller-constructed under a controlled dir
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, n)
	read, err := f.Read(buf)
	if err != nil && read == 0 {
		return nil, err
	}
	return buf[:read], nil
}

// IsHTML reports whether head sniffs as an HTML document, returning the detected
// content type alongside the verdict.
func IsHTML(head []byte) (string, bool) {
	ct := http.DetectContentType(head)
	return ct, strings.HasPrefix(ct, "text/html")
}

// HasPrefix reports whether b begins with prefix. It is a byte-slice helper for
// magic-byte checks (callers that need a format-specific magic compare against
// the head returned by ReadHead).
func HasPrefix(b, prefix []byte) bool {
	if len(b) < len(prefix) {
		return false
	}
	for i := range prefix {
		if b[i] != prefix[i] {
			return false
		}
	}
	return true
}

// ValidateNotHTMLAndSized is the generic guard applied to every fetched binary:
// the file must be at least minBytes (clamped up to DefaultSizeFloor) and must
// not sniff as HTML. name is used only for error messages.
func ValidateNotHTMLAndSized(path, name string, minBytes int64) error {
	if minBytes < DefaultSizeFloor {
		minBytes = DefaultSizeFloor
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("binaryfetch: stat artifact: %w", err)
	}
	if info.Size() < minBytes {
		return fmt.Errorf("%w: %q is %d bytes, need >= %d", ErrTooSmall, name, info.Size(), minBytes)
	}
	head, err := ReadHead(path, SniffLen)
	if err != nil {
		return fmt.Errorf("binaryfetch: read artifact head: %w", err)
	}
	if ct, ok := IsHTML(head); ok {
		return fmt.Errorf("%w: %q (%s) — its source URL points at a web page, not a downloadable binary", ErrLooksLikeHTML, name, ct)
	}
	return nil
}
