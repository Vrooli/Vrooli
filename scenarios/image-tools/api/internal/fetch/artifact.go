package fetch

import (
	"errors"
	"fmt"

	"github.com/vrooli/binaryfetch"
)

// =============================================================================
// Artifact spec + validation. Before the install-stub fix (docs/internal/
// PROBLEMS.md 2026-06-18), `models install` fetched whatever a source URL
// returned and pinned that file's hash with no check that the bytes were a
// model — so a GitHub landing PAGE downloaded as HTML and was recorded as
// "installed ✅". Every downloaded artifact now passes ValidateArtifact before an
// install is recorded.
//
// The generic guards (HTML-page rejection, truncated-download rejection, head
// sniffing) live in the shared github.com/vrooli/binaryfetch package so the
// platform host-tool fetcher and every catalog installer validate downloads
// identically. The Kind-specific weight magic below stays here — it is image
// model-weight domain knowledge shared by both the model and adapter catalogs,
// not generic binary-fetch logic.
// =============================================================================

// ArtifactKind classifies a downloadable weight artifact so the installer can
// validate that the bytes it fetched are actually that kind of file (and not,
// e.g., an HTML landing page — the install-stub bug). Empty kind means "generic
// binary"; only HTML rejection + a size floor are enforced for it.
type ArtifactKind string

const (
	// ArtifactGeneric applies the page/size guards but no format-specific magic.
	ArtifactGeneric ArtifactKind = ""
	// ArtifactONNX is an ONNX model (protobuf; starts with the ir_version tag).
	ArtifactONNX ArtifactKind = "onnx"
	// ArtifactGGUF is a llama.cpp/stable-diffusion.cpp GGUF weight.
	ArtifactGGUF ArtifactKind = "gguf"
	// ArtifactSafetensors is a safetensors weight file.
	ArtifactSafetensors ArtifactKind = "safetensors"
	// ArtifactNCNNParam / ArtifactNCNNBin are the ncnn model pair (realesrgan).
	ArtifactNCNNParam ArtifactKind = "ncnn-param"
	ArtifactNCNNBin   ArtifactKind = "ncnn-bin"
	// ArtifactBinary is a standalone executable artifact.
	ArtifactBinary ArtifactKind = "binary"
)

// Asset is one downloadable artifact a catalog entry needs on disk. An entry may
// require several (e.g. an ncnn .param + .bin pair). URLs MUST be direct,
// resolvable artifact links — never a landing/release/repo PAGE. A page URL
// downloads as HTML and was previously recorded as a "model" (the install-stub
// bug, see docs/internal/PROBLEMS.md 2026-06-18); the installer now validates
// every downloaded asset against Kind + size before an install is recorded.
type Asset struct {
	// URL is the direct, resolvable artifact link (HF resolve/main/<file>, a
	// GitHub release-asset download URL, etc.).
	URL string `json:"url"`
	// Filename is the on-disk name the backend expects under the entry dir.
	Filename string `json:"filename"`
	// Kind drives artifact validation. Empty = generic binary.
	Kind ArtifactKind `json:"kind"`
	// SHA256, when set, is the upstream-published checksum and is enforced. When
	// empty the freshly computed hash is pinned AFTER artifact validation passes.
	SHA256 string `json:"sha256,omitempty"`
	// MinBytes, when >0, is a lower bound used to catch truncated/page downloads.
	MinBytes int64 `json:"min_bytes,omitempty"`
}

// ErrArtifactNotWeight is returned when a downloaded artifact is not a model
// weight (an HTML page, or the wrong format for its declared Kind). The partial
// download is removed by the caller, exactly like a checksum mismatch.
var ErrArtifactNotWeight = errors.New("fetch: downloaded artifact is not a model weight")

// ValidateArtifact asserts that the file at path is a plausible model weight of
// the asset's declared Kind. It is the single guard that makes the install-stub
// bug impossible: an HTML page (or any non-weight) is rejected before install.
//
// Checks, in order:
//   - size >= max(MinBytes, binaryfetch.DefaultSizeFloor) and not HTML (shared)
//   - Kind-specific format magic (e.g. ONNX starts with the protobuf ir_version
//     tag 0x08); generic kind enforces only the HTML + size guards.
//
// The asset's SHA256 (when published) is verified separately by the installer
// (it needs the full-file hash it pins anyway), not here.
func ValidateArtifact(path string, a Asset) error {
	if err := binaryfetch.ValidateNotHTMLAndSized(path, a.Filename, a.MinBytes); err != nil {
		return err
	}

	head, err := binaryfetch.ReadHead(path, binaryfetch.SniffLen)
	if err != nil {
		return fmt.Errorf("fetch: read artifact head: %w", err)
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
		if !binaryfetch.HasPrefix(head, []byte("GGUF")) {
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
