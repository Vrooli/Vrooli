package models

import (
	"errors"
	"fmt"

	"github.com/vrooli/binaryfetch"
)

// =============================================================================
// Artifact validation (Phase 1 substrate fix). Before this, `models install`
// fetched whatever a model's source URL returned and pinned that file's hash
// with no check that the bytes were a model — so a GitHub landing PAGE
// downloaded as HTML and was recorded as "installed ✅" (the install-stub bug,
// docs/internal/PROBLEMS.md 2026-06-18). Every downloaded artifact now passes
// validateArtifact before an install is recorded.
//
// The generic guards (HTML-page rejection, truncated-download rejection, head
// sniffing) live in the shared github.com/vrooli/binaryfetch package so the
// platform host-tool fetcher and this model installer validate downloads
// identically. The Kind-specific weight magic below stays here — it is image
// model domain knowledge, not generic binary-fetch logic.
// =============================================================================

// ErrArtifactNotWeight is returned when a downloaded artifact is not a model
// weight (an HTML page, or the wrong format for its declared Kind). The partial
// download is removed by the caller, exactly like a checksum mismatch.
var ErrArtifactNotWeight = errors.New("models: downloaded artifact is not a model weight")

// validateArtifact asserts that the file at path is a plausible model weight of
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
func validateArtifact(path string, a Asset) error {
	if err := binaryfetch.ValidateNotHTMLAndSized(path, a.Filename, a.MinBytes); err != nil {
		return err
	}

	head, err := binaryfetch.ReadHead(path, binaryfetch.SniffLen)
	if err != nil {
		return fmt.Errorf("models: read artifact head: %w", err)
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
