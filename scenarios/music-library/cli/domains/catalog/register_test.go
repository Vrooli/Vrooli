package catalog

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifySource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "track.wav")
	data := []byte("source bytes")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if err := verifySource(path, fmt.Sprintf("%x", sum)); err != nil {
		t.Fatal(err)
	}
	if err := verifySource(path, fmt.Sprintf("%064d", 0)); err == nil {
		t.Fatal("accepted incorrect digest")
	}
	if err := verifySource("relative.wav", fmt.Sprintf("%x", sum)); err == nil {
		t.Fatal("accepted relative path")
	}
	if err := verifySource(filepath.Join(t.TempDir(), "missing"), fmt.Sprintf("%x", sum)); err == nil {
		t.Fatal("accepted missing file")
	}
}
