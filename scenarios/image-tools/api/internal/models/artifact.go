package models

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// =============================================================================
// Artifact validation (Phase 1 substrate fix). Before this, `models install`
// fetched whatever a model's source URL returned and pinned that file's hash
// with no check that the bytes were a model — so a GitHub landing PAGE
// downloaded as HTML and was recorded as "installed ✅" (the install-stub bug,
// docs/internal/PROBLEMS.md 2026-06-18). Every downloaded artifact now passes
// validateArtifact before an install is recorded.
// =============================================================================

// ErrArtifactNotWeight is returned when a downloaded artifact is not a model
// weight (an HTML page, or the wrong format for its declared Kind). The partial
// download is removed by the caller, exactly like a checksum mismatch.
var ErrArtifactNotWeight = errors.New("models: downloaded artifact is not a model weight")

// ErrArtifactTooSmall is returned when an artifact is below its declared (or a
// floor) minimum size — the signature of a truncated download or an error page.
var ErrArtifactTooSmall = errors.New("models: downloaded artifact is smaller than expected")

// artifactSizeFloor is the hard minimum any real weight must exceed. A few
// hundred bytes is always either a stub, an error JSON, or a redirect page.
const artifactSizeFloor int64 = 1 << 10 // 1 KiB

// sniffLen is how many leading bytes we read for content-type detection +
// format magic. http.DetectContentType only consults the first 512 bytes.
const sniffLen = 512

// validateArtifact asserts that the file at path is a plausible model weight of
// the asset's declared Kind. It is the single guard that makes the install-stub
// bug impossible: an HTML page (or any non-weight) is rejected before install.
//
// Checks, in order:
//   - size >= max(MinBytes, artifactSizeFloor)
//   - the leading bytes do NOT sniff as HTML (the page bug)
//   - Kind-specific format magic (e.g. ONNX starts with the protobuf ir_version
//     tag 0x08); generic kind enforces only the HTML + size guards.
//
// The asset's SHA256 (when published) is verified separately by the installer
// (it needs the full-file hash it pins anyway), not here.
func validateArtifact(path string, a Asset) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("models: stat artifact: %w", err)
	}
	minSize := a.MinBytes
	if minSize < artifactSizeFloor {
		minSize = artifactSizeFloor
	}
	if info.Size() < minSize {
		return fmt.Errorf("%w: %q is %d bytes, need >= %d", ErrArtifactTooSmall, a.Filename, info.Size(), minSize)
	}

	head, err := readHead(path, sniffLen)
	if err != nil {
		return fmt.Errorf("models: read artifact head: %w", err)
	}

	// Reject HTML regardless of declared kind — this is the exact stub symptom
	// (a release/repo page served as text/html).
	if ct := http.DetectContentType(head); strings.HasPrefix(ct, "text/html") {
		return fmt.Errorf("%w: %q looks like an HTML page (%s) — its source URL points at a web page, not a downloadable weight", ErrArtifactNotWeight, a.Filename, ct)
	}

	if err := checkKindMagic(a.Kind, head); err != nil {
		return fmt.Errorf("%w: %q (%v)", ErrArtifactNotWeight, a.Filename, err)
	}
	return nil
}

// checkKindMagic applies a lightweight format check for the kinds we can
// recognize cheaply. Unknown/generic kinds pass (the HTML + size guards already
// ran). The goal is to reject obvious non-weights, not to fully parse a model.
func checkKindMagic(kind ArtifactKind, head []byte) error {
	switch kind {
	case ArtifactONNX:
		// ONNX is a serialized ModelProto. Field 1 (ir_version, varint) is
		// effectively always present and first, so the file opens with tag 0x08.
		if len(head) == 0 || head[0] != 0x08 {
			return fmt.Errorf("not an ONNX protobuf (missing ir_version tag)")
		}
	case ArtifactGGUF:
		if !hasPrefix(head, []byte("GGUF")) {
			return fmt.Errorf("missing GGUF magic")
		}
	case ArtifactSafetensors:
		// safetensors opens with an 8-byte little-endian header length followed
		// by a JSON object, so byte 8 is '{'. Cheap, robust enough to reject HTML.
		if len(head) < 9 || head[8] != '{' {
			return fmt.Errorf("missing safetensors JSON header")
		}
	case ArtifactGeneric, ArtifactBinary, ArtifactNCNNParam, ArtifactNCNNBin:
		// No cheap, reliable magic — the HTML + size guards are the protection.
	default:
		// Unknown kind: do not block (forward-compatible), guards already ran.
	}
	return nil
}

func hasPrefix(b, prefix []byte) bool {
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

func readHead(path string, n int) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // path is engine-constructed under Root
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
