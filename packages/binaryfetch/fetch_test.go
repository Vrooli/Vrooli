package binaryfetch

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// bigBinary returns a payload comfortably above the size floor that does not
// sniff as HTML (leading ELF-ish bytes).
func bigBinary() []byte {
	body := append([]byte{0x7f, 'E', 'L', 'F'}, bytes.Repeat([]byte{0x42}, 4096)...)
	return body
}

func serve(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchRawHappyPath(t *testing.T) {
	body := bigBinary()
	srv := serve(t, body)
	dest := t.TempDir()

	var stages []Stage
	path, err := Fetch(context.Background(), Target{
		Name:   "mytool",
		URL:    srv.URL,
		SHA256: sha256hex(body),
	}, dest, func(p Progress) { stages = append(stages, p.Stage) })
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if path != filepath.Join(dest, "mytool") {
		t.Fatalf("path = %q", path)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read installed: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Fatalf("installed bytes differ")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("installed binary is not executable: %v", info.Mode())
	}
	// Progress should report downloading then installing.
	if len(stages) == 0 || stages[0] != StageDownloading {
		t.Fatalf("expected downloading first, got %v", stages)
	}
}

func TestFetchChecksumMismatch(t *testing.T) {
	body := bigBinary()
	srv := serve(t, body)
	_, err := Fetch(context.Background(), Target{
		Name:   "mytool",
		URL:    srv.URL,
		SHA256: sha256hex([]byte("different")),
	}, t.TempDir(), nil)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("err = %v, want ErrChecksumMismatch", err)
	}
}

func TestFetchRejectsHTML(t *testing.T) {
	body := []byte("<!DOCTYPE html><html><head><title>Release</title></head><body>page</body></html>" + strings.Repeat(" ", 2000))
	srv := serve(t, body)
	_, err := Fetch(context.Background(), Target{
		Name:   "mytool",
		URL:    srv.URL,
		SHA256: sha256hex(body),
	}, t.TempDir(), nil)
	if !errors.Is(err, ErrLooksLikeHTML) {
		t.Fatalf("err = %v, want ErrLooksLikeHTML", err)
	}
}

func TestFetchRejectsTooSmall(t *testing.T) {
	body := []byte{0x7f, 'E', 'L', 'F', 0x01}
	srv := serve(t, body)
	_, err := Fetch(context.Background(), Target{
		Name:   "mytool",
		URL:    srv.URL,
		SHA256: sha256hex(body),
	}, t.TempDir(), nil)
	if !errors.Is(err, ErrTooSmall) {
		t.Fatalf("err = %v, want ErrTooSmall", err)
	}
}

func TestFetchTarGzExtract(t *testing.T) {
	bin := bigBinary()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: "build/bin/sd", Mode: 0o755, Size: int64(len(bin)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	tw.Write(bin)
	tw.Close()
	gz.Close()
	archive := buf.Bytes()

	srv := serve(t, archive)
	dest := t.TempDir()
	path, err := Fetch(context.Background(), Target{
		Name:    "sd",
		URL:     srv.URL,
		SHA256:  sha256hex(archive),
		Archive: "tar.gz",
		BinPath: "build/bin/sd",
	}, dest, nil)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, bin) {
		t.Fatalf("extracted bytes differ")
	}
}

func TestFetchZipExtractSingleFile(t *testing.T) {
	bin := bigBinary()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("realesrgan")
	w.Write(bin)
	zw.Close()
	archive := buf.Bytes()

	srv := serve(t, archive)
	dest := t.TempDir()
	// binPath omitted: single regular file should be selected automatically.
	path, err := Fetch(context.Background(), Target{
		Name:    "realesrgan-ncnn-vulkan",
		URL:     srv.URL,
		SHA256:  sha256hex(archive),
		Archive: "zip",
	}, dest, nil)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, bin) {
		t.Fatalf("extracted bytes differ")
	}
}

func TestFetchArchiveMissingBinPath(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, name := range []string{"a", "b"} {
		tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: 4, Typeflag: tar.TypeReg})
		tw.Write([]byte("data"))
	}
	tw.Close()
	gz.Close()
	archive := buf.Bytes()

	srv := serve(t, archive)
	_, err := Fetch(context.Background(), Target{
		Name:    "x",
		URL:     srv.URL,
		SHA256:  sha256hex(archive),
		Archive: "tar.gz",
		BinPath: "does-not-exist",
	}, t.TempDir(), nil)
	if !errors.Is(err, ErrNoBinaryInArchive) {
		t.Fatalf("err = %v, want ErrNoBinaryInArchive", err)
	}
}

// zipTree builds a zip whose entries are the given name->bytes map.
func zipTree(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFetchDirHappyPath(t *testing.T) {
	bin := bigBinary()
	lib := bigBinary()
	archive := zipTree(t, map[string][]byte{
		"sd-cli":                 bin,
		"libstable-diffusion.so": lib,
		"README.txt":             bytes.Repeat([]byte("x"), 64),
	})
	srv := serve(t, archive)
	optDir := filepath.Join(t.TempDir(), "opt", "sd")
	// Seed a stale file to prove optDir is cleaned/replaced on (re)install.
	if err := os.MkdirAll(optDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(optDir, "stale"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}

	entry, err := FetchDir(context.Background(), Target{
		Name:    "sd",
		URL:     srv.URL,
		SHA256:  sha256hex(archive),
		Archive: "zip",
		Layout:  "dir",
		BinPath: "sd-cli",
	}, optDir, nil)
	if err != nil {
		t.Fatalf("FetchDir: %v", err)
	}
	if entry != filepath.Join(optDir, "sd-cli") {
		t.Fatalf("entry = %q", entry)
	}
	// All archive members present; stale file gone.
	for _, name := range []string{"sd-cli", "libstable-diffusion.so", "README.txt"} {
		if _, statErr := os.Stat(filepath.Join(optDir, name)); statErr != nil {
			t.Fatalf("expected %q extracted: %v", name, statErr)
		}
	}
	if _, statErr := os.Stat(filepath.Join(optDir, "stale")); statErr == nil {
		t.Fatalf("stale file should have been removed on reinstall")
	}
	info, _ := os.Stat(entry)
	if info.Mode().Perm()&0o100 == 0 {
		t.Fatalf("entry binary not executable: %v", info.Mode())
	}
}

func TestFetchDirRejectsTraversal(t *testing.T) {
	// A zip entry escaping the extract dir via ../ must be refused.
	archive := zipTree(t, map[string][]byte{
		"sd-cli":        bigBinary(),
		"../escape.txt": []byte("pwned"),
	})
	srv := serve(t, archive)
	optDir := filepath.Join(t.TempDir(), "opt", "sd")
	_, err := FetchDir(context.Background(), Target{
		Name:    "sd",
		URL:     srv.URL,
		SHA256:  sha256hex(archive),
		Archive: "zip",
		Layout:  "dir",
		BinPath: "sd-cli",
	}, optDir, nil)
	if err == nil || !strings.Contains(err.Error(), "escapes extract dir") {
		t.Fatalf("err = %v, want traversal rejection", err)
	}
}

func TestFetchDirChecksumMismatch(t *testing.T) {
	archive := zipTree(t, map[string][]byte{"sd-cli": bigBinary()})
	srv := serve(t, archive)
	_, err := FetchDir(context.Background(), Target{
		Name:    "sd",
		URL:     srv.URL,
		SHA256:  sha256hex([]byte("different")),
		Archive: "zip",
		Layout:  "dir",
		BinPath: "sd-cli",
	}, filepath.Join(t.TempDir(), "sd"), nil)
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Fatalf("err = %v, want ErrChecksumMismatch", err)
	}
}

func TestFetchDirMissingBinPath(t *testing.T) {
	archive := zipTree(t, map[string][]byte{"other-bin": bigBinary()})
	srv := serve(t, archive)
	_, err := FetchDir(context.Background(), Target{
		Name:    "sd",
		URL:     srv.URL,
		SHA256:  sha256hex(archive),
		Archive: "zip",
		Layout:  "dir",
		BinPath: "sd-cli",
	}, filepath.Join(t.TempDir(), "sd"), nil)
	if !errors.Is(err, ErrNoBinaryInArchive) {
		t.Fatalf("err = %v, want ErrNoBinaryInArchive", err)
	}
}

func TestFetchDirTarGzPreservesTree(t *testing.T) {
	bin := bigBinary()
	model := bytes.Repeat([]byte("m"), 2048)
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range []struct {
		name string
		body []byte
		typ  byte
	}{
		{"realesrgan-ncnn-vulkan", bin, tar.TypeReg},
		{"models/", nil, tar.TypeDir},
		{"models/realesr.bin", model, tar.TypeReg},
	} {
		hdr := &tar.Header{Name: e.name, Mode: 0o755, Size: int64(len(e.body)), Typeflag: e.typ}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if len(e.body) > 0 {
			tw.Write(e.body)
		}
	}
	tw.Close()
	gz.Close()
	archive := buf.Bytes()

	srv := serve(t, archive)
	optDir := filepath.Join(t.TempDir(), "opt", "realesrgan")
	entry, err := FetchDir(context.Background(), Target{
		Name:    "realesrgan-ncnn-vulkan",
		URL:     srv.URL,
		SHA256:  sha256hex(archive),
		Archive: "tar.gz",
		Layout:  "dir",
		BinPath: "realesrgan-ncnn-vulkan",
	}, optDir, nil)
	if err != nil {
		t.Fatalf("FetchDir: %v", err)
	}
	if _, statErr := os.Stat(entry); statErr != nil {
		t.Fatalf("entry missing: %v", statErr)
	}
	got, err := os.ReadFile(filepath.Join(optDir, "models", "realesr.bin"))
	if err != nil {
		t.Fatalf("nested model file missing: %v", err)
	}
	if !bytes.Equal(got, model) {
		t.Fatalf("nested model bytes differ")
	}
}

func TestFetchCancellation(t *testing.T) {
	// A server that blocks lets us prove the context cancels the download.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := Fetch(ctx, Target{Name: "x", URL: srv.URL, SHA256: sha256hex(bigBinary())}, t.TempDir(), nil)
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
}

func TestFetchRequiresChecksum(t *testing.T) {
	_, err := Fetch(context.Background(), Target{Name: "x", URL: "https://e/x"}, t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "SHA256 is required") {
		t.Fatalf("err = %v, want SHA256 required", err)
	}
}
