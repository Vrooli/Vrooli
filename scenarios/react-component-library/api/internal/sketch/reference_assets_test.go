package sketch

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveReferenceAssetUsesContentIdentityAndRejectsNonImages(t *testing.T) {
	root := t.TempDir()
	id, name, size, err := SaveReferenceAsset(root, "demo", "home", "reference.png", "image/png", bytes.NewReader([]byte("png-bytes")))
	if err != nil {
		t.Fatal(err)
	}
	if id == "" || name != id+".png" || size != 9 {
		t.Fatalf("unexpected asset metadata: id=%q name=%q size=%d", id, name, size)
	}
	path, err := ReferenceAssetPath(root, "demo", "home", name)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, []byte("png-bytes")) {
		t.Fatalf("stored content = %q", stored)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("asset mode = %04o, want 0600", fileInfo.Mode().Perm())
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("asset directory mode = %04o, want 0700", dirInfo.Mode().Perm())
	}
	if _, _, _, err := SaveReferenceAsset(root, "demo", "home", "reference.txt", "text/plain", bytes.NewReader([]byte("nope"))); err == nil {
		t.Fatal("expected non-image asset to be rejected")
	}
}

func TestReferenceAssetPathRejectsTraversal(t *testing.T) {
	if _, err := ReferenceAssetPath(filepath.FromSlash("/tmp/root"), "demo", "home", "../secret.png"); err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}
