package binaryfetch

import (
	"archive/zip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeZipWithSymlink(t *testing.T, linkname string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "bundle.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	file, err := w.Create("App.app/Contents/Resources/libllama.0.0.1.dylib")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("dylib")); err != nil {
		t.Fatal(err)
	}
	hdr := &zip.FileHeader{Name: "App.app/Contents/Resources/libllama.0.dylib", Method: zip.Store}
	hdr.SetMode(os.ModeSymlink | 0o777)
	link, err := w.CreateHeader(hdr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := link.Write([]byte(linkname)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// macOS application bundles ship versioned dylibs as zip symlinks (Ollama.app);
// they extract as symlinks inside the destination.
func TestZipSymlinkInsideTheTreeIsExtracted(t *testing.T) {
	dest := t.TempDir()
	summary, err := ExtractArchiveBounded(writeZipWithSymlink(t, "libllama.0.0.1.dylib"), "zip", dest, ExtractOptions{})
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if summary.Symlinks != 1 {
		t.Fatalf("symlinks = %d, want 1", summary.Symlinks)
	}
	got, err := os.Readlink(filepath.Join(dest, "App.app/Contents/Resources/libllama.0.dylib"))
	if err != nil || got != "libllama.0.0.1.dylib" {
		t.Fatalf("symlink target = %q (%v)", got, err)
	}
}

func TestZipSymlinkEscapingTheTreeIsRefused(t *testing.T) {
	_, err := ExtractArchiveBounded(writeZipWithSymlink(t, "../../../../../etc/passwd"), "zip", t.TempDir(), ExtractOptions{})
	var archiveErr *ArchiveError
	if !errors.As(err, &archiveErr) {
		t.Fatalf("err = %v, want an ArchiveError", err)
	}
}
