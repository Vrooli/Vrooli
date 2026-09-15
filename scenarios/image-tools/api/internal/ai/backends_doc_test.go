package ai

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// backendsDocPath locates docs/reference/backends.md relative to this test file
// (api/internal/ai → ../../../docs/reference/backends.md).
func backendsDocPath(t *testing.T) string {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "docs", "reference", "backends.md"))
}

func extractGeneratedSection(t *testing.T, doc string) string {
	t.Helper()
	begin := strings.Index(doc, HostToolMatrixBeginMarker)
	end := strings.Index(doc, HostToolMatrixEndMarker)
	if begin < 0 || end < 0 || end < begin {
		t.Fatalf("backends.md is missing the generated host-tool-matrix markers")
	}
	return strings.TrimSpace(doc[begin+len(HostToolMatrixBeginMarker) : end])
}

// TestBackendsDocHostToolMatrixUpToDate fails the build when backends.md drifts
// from providerSpecs(). Regenerate with `make backends-doc` (UPDATE_BACKENDS_DOC=1).
func TestBackendsDocHostToolMatrixUpToDate(t *testing.T) {
	path := backendsDocPath(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read backends.md: %v", err)
	}
	doc := string(data)
	want := strings.TrimSpace(RenderHostToolMatrix())

	if os.Getenv("UPDATE_BACKENDS_DOC") == "1" {
		begin := strings.Index(doc, HostToolMatrixBeginMarker)
		end := strings.Index(doc, HostToolMatrixEndMarker)
		if begin < 0 || end < 0 || end < begin {
			t.Fatalf("backends.md is missing the generated host-tool-matrix markers")
		}
		updated := doc[:begin+len(HostToolMatrixBeginMarker)] + "\n" + want + "\n" + doc[end:]
		if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
			t.Fatalf("write backends.md: %v", err)
		}
		t.Log("regenerated host-tool matrix in backends.md")
		return
	}

	got := extractGeneratedSection(t, doc)
	if got != want {
		t.Fatalf("backends.md host-tool matrix is stale; run `make backends-doc`.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
