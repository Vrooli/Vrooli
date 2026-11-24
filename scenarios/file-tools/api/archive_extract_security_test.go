package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func extractArchiveForTest(t *testing.T, server *Server, archivePath, destination string) int {
	t.Helper()
	body, err := json.Marshal(map[string]string{"archive_path": archivePath, "destination_path": destination})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/v1/files/extract", bytes.NewReader(body))
	w := httptest.NewRecorder()
	server.handleExtract(w, req)
	return w.Code
}

func writeZipForTest(t *testing.T, archivePath string, entries map[string]string) {
	t.Helper()
	out, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(out)
	for name, contents := range entries {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestHandleExtractRejectsUnsafeZipPaths(t *testing.T) {
	for _, entry := range []string{"../escape.txt", "nested/../../escape.txt", "/absolute.txt", "C:/drive.txt"} {
		t.Run(entry, func(t *testing.T) {
			base := t.TempDir()
			archivePath := filepath.Join(base, "input.zip")
			destination := filepath.Join(base, "out")
			writeZipForTest(t, archivePath, map[string]string{entry: "escaped"})

			server := &Server{}
			if status := extractArchiveForTest(t, server, archivePath, destination); status == 200 {
				t.Fatalf("unsafe entry %q was accepted", entry)
			}
			if _, err := os.Stat(filepath.Join(base, "escape.txt")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("outside file exists or could not be checked: %v", err)
			}
		})
	}
}

func TestExtractArchiveEnforcesEntryAndAggregateBudgets(t *testing.T) {
	t.Run("per-entry", func(t *testing.T) {
		base := t.TempDir()
		archivePath := filepath.Join(base, "input.zip")
		writeZipForTest(t, archivePath, map[string]string{"large.txt": "four"})
		destination := filepath.Join(base, "out")
		_, _, err := extractArchiveWithLimits(archivePath, destination, archiveLimits{
			maxEntries: 10, maxEntryBytes: 3, maxTotalBytes: 10,
		})
		if !errors.Is(err, errArchiveTooLarge) {
			t.Fatalf("extract error = %v, want per-entry limit", err)
		}
		if _, err := os.Stat(filepath.Join(destination, "large.txt")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("oversized entry was created: %v", err)
		}
	})

	t.Run("aggregate", func(t *testing.T) {
		base := t.TempDir()
		archivePath := filepath.Join(base, "input.zip")
		writeZipForTest(t, archivePath, map[string]string{"one.txt": "abc", "two.txt": "def"})
		destination := filepath.Join(base, "out")
		_, _, err := extractArchiveWithLimits(archivePath, destination, archiveLimits{
			maxEntries: 10, maxEntryBytes: 4, maxTotalBytes: 5,
		})
		if !errors.Is(err, errArchiveTooLarge) {
			t.Fatalf("extract error = %v, want aggregate limit", err)
		}
		entries, err := os.ReadDir(destination)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 1 {
			t.Fatalf("extracted entries = %d, want exactly one bounded entry", len(entries))
		}
		info, err := entries[0].Info()
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() != 3 {
			t.Fatalf("first entry size = %d, want 3", info.Size())
		}
	})

	t.Run("entry-count", func(t *testing.T) {
		base := t.TempDir()
		archivePath := filepath.Join(base, "input.zip")
		writeZipForTest(t, archivePath, map[string]string{"one.txt": "", "two.txt": "", "three.txt": ""})
		_, _, err := extractArchiveWithLimits(archivePath, filepath.Join(base, "out"), archiveLimits{
			maxEntries: 2, maxEntryBytes: 3, maxTotalBytes: 10,
		})
		if !errors.Is(err, errArchiveTooLarge) {
			t.Fatalf("extract error = %v, want entry-count limit", err)
		}
	})
}

func TestHandleExtractWritesSafeNestedZipAndTarEntries(t *testing.T) {
	t.Run("zip", func(t *testing.T) {
		base := t.TempDir()
		archivePath := filepath.Join(base, "input.zip")
		writeZipForTest(t, archivePath, map[string]string{"nested/hello.txt": "hello zip"})
		destination := filepath.Join(base, "out")
		if status := extractArchiveForTest(t, &Server{}, archivePath, destination); status != 200 {
			t.Fatalf("extract status = %d, want 200", status)
		}
		got, err := os.ReadFile(filepath.Join(destination, "nested", "hello.txt"))
		if err != nil || string(got) != "hello zip" {
			t.Fatalf("extracted zip contents = %q, %v", got, err)
		}
	})

	t.Run("tar", func(t *testing.T) {
		base := t.TempDir()
		archivePath := filepath.Join(base, "input.tar")
		archive, err := os.Create(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		writer := tar.NewWriter(archive)
		contents := []byte("hello tar")
		if err := writer.WriteHeader(&tar.Header{Name: "nested/hello.txt", Mode: 0o600, Size: int64(len(contents)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(contents); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := archive.Close(); err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(base, "out")
		if status := extractArchiveForTest(t, &Server{}, archivePath, destination); status != 200 {
			t.Fatalf("extract status = %d, want 200", status)
		}
		got, err := os.ReadFile(filepath.Join(destination, "nested", "hello.txt"))
		if err != nil || string(got) != string(contents) {
			t.Fatalf("extracted tar contents = %q, %v", got, err)
		}
	})
}

func TestCopyArchiveEntryStopsAtByteLimit(t *testing.T) {
	var dst bytes.Buffer
	n, err := copyArchiveEntry(&dst, bytes.NewBufferString("abcdef"), 3, 3)
	if !errors.Is(err, errArchiveTooLarge) {
		t.Fatalf("copy error = %v, want archive byte limit", err)
	}
	if n != 3 || dst.String() != "abc" {
		t.Fatalf("copied %d bytes %q, want 3 bytes without writing past the limit", n, dst.String())
	}
}

func TestHandleExtractRejectsTarTraversal(t *testing.T) {
	base := t.TempDir()
	archivePath := filepath.Join(base, "input.tar")
	archive, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := tar.NewWriter(archive)
	if err := writer.WriteHeader(&tar.Header{Name: "../escape.txt", Mode: 0o600, Size: int64(len("escaped")), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("escaped")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}

	server := &Server{}
	if status := extractArchiveForTest(t, server, archivePath, filepath.Join(base, "out")); status == 200 {
		t.Fatal("tar traversal entry was accepted")
	}
	if _, err := os.Stat(filepath.Join(base, "escape.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside file exists or could not be checked: %v", err)
	}
}

func TestHandleExtractDoesNotFollowDestinationSymlinkOutsideRoot(t *testing.T) {
	base := t.TempDir()
	destination := filepath.Join(base, "out")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(destination, "pivot")); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(base, "input.zip")
	writeZipForTest(t, archivePath, map[string]string{"pivot/escape.txt": "escaped"})

	server := &Server{}
	if status := extractArchiveForTest(t, server, archivePath, destination); status == 200 {
		t.Fatal("entry through an escaping destination symlink was accepted")
	}
	if _, err := os.Stat(filepath.Join(outside, "escape.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("symlink target file exists or could not be checked: %v", err)
	}
}

func TestHandleExtractRejectsArchiveSymlinks(t *testing.T) {
	t.Run("zip", func(t *testing.T) {
		base := t.TempDir()
		archivePath := filepath.Join(base, "input.zip")
		out, err := os.Create(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		writer := zip.NewWriter(out)
		header := &zip.FileHeader{Name: "pivot", Method: zip.Deflate}
		header.SetMode(os.ModeSymlink | 0o777)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte("../outside")); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := out.Close(); err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(base, "out")
		if status := extractArchiveForTest(t, &Server{}, archivePath, destination); status == 200 {
			t.Fatal("zip symlink entry was accepted")
		}
		if _, err := os.Lstat(filepath.Join(destination, "pivot")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("zip symlink was materialized or could not be checked: %v", err)
		}
	})

	t.Run("tar", func(t *testing.T) {
		base := t.TempDir()
		archivePath := filepath.Join(base, "input.tar")
		out, err := os.Create(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		writer := tar.NewWriter(out)
		if err := writer.WriteHeader(&tar.Header{Name: "pivot", Linkname: "../outside", Typeflag: tar.TypeSymlink, Mode: 0o777}); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := out.Close(); err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(base, "out")
		if status := extractArchiveForTest(t, &Server{}, archivePath, destination); status == 200 {
			t.Fatal("tar symlink entry was accepted")
		}
		if _, err := os.Lstat(filepath.Join(destination, "pivot")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("tar symlink was materialized or could not be checked: %v", err)
		}
	})
}

func TestGzipTarExtractionUsesSamePathAndByteGuards(t *testing.T) {
	base := t.TempDir()
	archivePath := filepath.Join(base, "input.tgz")
	archive, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(archive)
	writer := tar.NewWriter(gz)
	if err := writer.WriteHeader(&tar.Header{Name: "../escape.txt", Mode: 0o600, Size: 7, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("escaped")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}

	server := &Server{}
	if status := extractArchiveForTest(t, server, archivePath, filepath.Join(base, "out")); status == 200 {
		t.Fatal("gzip tar traversal entry was accepted")
	}
	if _, err := os.Stat(filepath.Join(base, "escape.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside file exists or could not be checked: %v", err)
	}
}
