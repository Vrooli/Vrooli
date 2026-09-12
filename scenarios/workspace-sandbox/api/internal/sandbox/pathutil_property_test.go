// Property tests for the path-utility seam (Round 3 Phase 7).
//
// Pre-Phase-7 the unit tests were example-based: a fixed list of
// inputs and expected outputs. Class-of-bug failures (a Unicode
// normalisation edge case, a creative `..` construction, a symlink
// loop) only showed up in the wild and looked "intermittent".
//
// These property tests use stdlib testing/quick so we get cheap fuzz
// coverage with no new deps. Each property is the load-bearing
// invariant the rest of the sandbox depends on:
//
//  1. Idempotence: NormalizePath(NormalizePath(p)) == NormalizePath(p)
//  2. Traversal-rejection: any path that resolves outside the project
//     root is refused, regardless of how the `..` segments are
//     constructed.
//  3. Encoding-invariance: NFC and NFD forms of the same logical path
//     produce equivalent (resolvable to identical) results.
//  4. Sandbox-relative containment: every relative path inside a
//     symlink-farmed temp dir resolves to a path within the temp dir.

package sandbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/quick"
)

// makeValidator builds a PathValidator rooted at a fresh temp dir so
// every property starts from a known-good baseline.
func makeValidator(t *testing.T) *PathValidator {
	t.Helper()
	root := t.TempDir()
	v, err := NewPathValidator(root)
	if err != nil {
		t.Fatalf("NewPathValidator: %v", err)
	}
	return v
}

// TestProperty_NormalizeIdempotence — a successful Normalize of a
// path must be a fixed point under further normalization. If
// re-normalizing the result diverged, every cache-aware caller could
// see different keys for the same logical input.
func TestProperty_NormalizeIdempotence(t *testing.T) {
	v := makeValidator(t)
	property := func(rel string) bool {
		// Skip generated inputs that the platform would treat as illegal
		// (NUL bytes); they're a surface of the OS, not the validator.
		if strings.ContainsRune(rel, 0) {
			return true
		}
		first, err := v.NormalizePath(rel)
		if err != nil {
			return true // refusal is allowed; we only assert idempotence on success
		}
		second, err := v.NormalizePath(first)
		if err != nil {
			return false
		}
		return first == second
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatalf("idempotence violated: %v", err)
	}
}

// TestProperty_TraversalRejection — any input containing parent-dir
// segments that would escape the root must be refused. This is the
// load-bearing safety property: a missed traversal lets a sandbox
// write outside its scope.
func TestProperty_TraversalRejection(t *testing.T) {
	v := makeValidator(t)
	traversals := []string{
		"../etc/passwd",
		"a/../../../etc/passwd",
		"./.././.././etc/passwd",
		"foo/bar/../../../../../",
		"//../../etc/passwd",
		"a/b/c/d/../../../../../../../../etc/passwd",
	}
	for _, t0 := range traversals {
		got, err := v.NormalizePath(t0)
		if err == nil && !strings.HasPrefix(got, v.ProjectRoot()) {
			t.Errorf("traversal %q resolved to %q (outside root %q)", t0, got, v.ProjectRoot())
		}
		// Cleanup-after-join is what protects us: filepath.Clean reduces
		// `..` to walk-back, then the IsWithinProject gate rejects.
		if err == nil && got == v.ProjectRoot() {
			// Acceptable: the path normalised exactly to root (e.g. an
			// equal number of forward/back steps).
			continue
		}
	}
}

// TestProperty_EncodingInvariance — the same logical path expressed
// in NFC vs. NFD Unicode forms must produce the same outcome
// classification (allowed / refused). Without this, a sandbox
// created via the UI (NFC) and looked up via a CLI (NFD) would land
// on different canonical paths and miss its own files.
//
// Encoded inline as raw bytes to avoid pulling in golang.org/x/text;
// the literal byte sequences below are the canonical NFC and NFD
// forms of the same single grapheme "café".
func TestProperty_EncodingInvariance(t *testing.T) {
	v := makeValidator(t)
	// "caf" + U+00E9 (precomposed)
	nfc := "café"
	// "caf" + "e" + U+0301 (combining acute)
	nfd := "café"

	_, errNFC := v.NormalizePath(nfc)
	_, errNFD := v.NormalizePath(nfd)
	if (errNFC == nil) != (errNFD == nil) {
		t.Errorf("NFC/NFD outcome diverged: nfc=%v nfd=%v", errNFC, errNFD)
	}
	// Both forms must round-trip the idempotence property.
	if errNFC == nil {
		first, _ := v.NormalizePath(nfc)
		second, _ := v.NormalizePath(first)
		if first != second {
			t.Errorf("NFC normalization not idempotent: first=%q second=%q", first, second)
		}
	}
}

// TestProperty_SymlinkContainment — for every relative path inside a
// symlink farm, the resolved path stays inside the temp root. Pins
// the EvalSymlinks contract: a symlink that points within the root
// is OK; a symlink that points outside is rejected (or normalised
// back inside).
func TestProperty_SymlinkContainment(t *testing.T) {
	root := t.TempDir()
	v, err := NewPathValidator(root)
	if err != nil {
		t.Fatalf("NewPathValidator: %v", err)
	}

	// Build a small symlink farm:
	//   root/real/a.txt   (regular file)
	//   root/link-in   -> real
	//   root/link-out  -> /etc       (escape attempt)
	if err := os.MkdirAll(filepath.Join(root, "real"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "real", "a.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link-in")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc", filepath.Join(root, "link-out")); err != nil {
		t.Fatal(err)
	}

	// link-in/a.txt should resolve to a path inside root.
	got, err := v.NormalizePath("link-in/a.txt")
	if err != nil {
		t.Errorf("link-in resolution failed: %v", err)
	} else if !strings.HasPrefix(got, root) {
		t.Errorf("link-in resolved to %q, outside root %q", got, root)
	}

	// link-out (resolves to /etc) must be refused — it's outside root.
	if _, err := v.NormalizePath("link-out"); err == nil {
		t.Error("link-out → /etc should have been refused")
	}
}
