package resources

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRGBZipRefusesEscapesAndAliases(t *testing.T) {
	for _, name := range []string{"../outside", "OpenRGB.app/../outside", "OtherBundle/file", "OpenRGB.app/C:/outside", "OpenRGB.app/a\\b"} {
		t.Run(name, func(t *testing.T) {
			archive := filepath.Join(t.TempDir(), "archive.zip")
			file, err := os.Create(archive)
			if err != nil {
				t.Fatal(err)
			}
			w := zip.NewWriter(file)
			entry, err := w.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			entry.Write([]byte("untrusted"))
			w.Close()
			file.Close()
			f, err := os.Open(archive)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()
			p, _ := openRGBReleaseProfileFor("darwin", "amd64")
			if err := extractOpenRGBZip(t.Context(), f, filepath.Join(t.TempDir(), "bundle"), p); err == nil {
				t.Fatal("accepted archive path escape")
			}
		})
	}
}

func TestOpenRGBZipNeverTraversesArchiveLinks(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "archive.zip")
	file, _ := os.Create(archive)
	writer := zip.NewWriter(file)
	h := &zip.FileHeader{Name: "OpenRGB.app/link"}
	h.SetMode(os.ModeSymlink | 0o777)
	entry, _ := writer.CreateHeader(h)
	entry.Write([]byte("../outside"))
	entry, _ = writer.Create("OpenRGB.app/link/overwrite")
	entry.Write([]byte("bad"))
	writer.Close()
	file.Close()
	f, _ := os.Open(archive)
	defer f.Close()
	p, _ := openRGBReleaseProfileFor("darwin", "amd64")
	if err := extractOpenRGBZip(t.Context(), f, filepath.Join(t.TempDir(), "bundle"), p); err == nil {
		t.Fatal("accepted escaping archive-provided link")
	}
}

func TestOpenRGBZipNeverCreatesLinksThroughQueuedLinkParents(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(root, "archive.zip")
	file, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for _, link := range [][2]string{{"A", "."}, {"A/B", "../outside"}, {"A/B/C", "."}} {
		h := &zip.FileHeader{Name: "OpenRGB.app/" + link[0]}
		h.SetMode(os.ModeSymlink | 0o777)
		entry, err := writer.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(link[1])); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	p, _ := openRGBReleaseProfileFor("darwin", "amd64")
	if err := extractOpenRGBZip(t.Context(), f, filepath.Join(root, "bundle"), p); err == nil {
		t.Fatal("accepted queued symlink parent topology")
	}
	entries, err := os.ReadDir(outside)
	if err != nil || len(entries) != 0 {
		t.Fatalf("archive created a link outside its bundle: entries=%v err=%v", entries, err)
	}
}
