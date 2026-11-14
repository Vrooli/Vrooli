//go:build linux

package smoketest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvictAndMeasureArtifactPageCacheUsesUnprivilegedFileOperations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "artifact.AppImage")
	if err := os.WriteFile(path, make([]byte, 256*1024), 0o600); err != nil {
		t.Fatal(err)
	}
	residency, err := evictAndMeasureArtifactPageCache(path)
	if err != nil {
		t.Fatal(err)
	}
	if residency < 0 || residency > 100 {
		t.Fatalf("residency=%v, want percentage in [0,100]", residency)
	}
}
