//go:build linux

package finiteinstallation

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestExecutingImageHashUsesRetainedFDDespitePathReplacement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image")
	if e := os.WriteFile(path, []byte("original image"), 0600); e != nil {
		t.Fatal(e)
	}
	f, e := os.Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if e = os.Remove(path); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, []byte("replacement image"), 0600); e != nil {
		t.Fatal(e)
	}
	expected := sha256.Sum256([]byte("original image"))
	got, e := hashImageFD(f)
	if e != nil || got != hex.EncodeToString(expected[:]) {
		t.Fatalf("retained executing image hash: %v", e)
	}
}
