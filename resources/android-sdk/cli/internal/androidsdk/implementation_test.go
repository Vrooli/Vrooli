package androidsdk

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/testkit"
)

func TestInstallDownloadsAndValidatesPlatformTools(t *testing.T) {
	h := testkit.Handlers(t)
	if h.Stdout == nil || h.Stderr == nil {
		t.Fatal("test harness must provide output buffers")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ANDROID_SDK_SKIP_COMPONENTS", "1")
	archivePath := filepath.Join(t.TempDir(), "platform-tools.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create("platform-tools/adb")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte("#!/bin/sh\necho Android Debug Bridge version 1\n")); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(raw)
	}))
	defer server.Close()
	t.Setenv("ANDROID_PLATFORM_TOOLS_URL", server.URL)
	t.Setenv("ANDROID_PLATFORM_TOOLS_SHA256", hex.EncodeToString(digest[:]))
	if err := Install(); err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(os.Getenv("HOME"), ".vrooli", "resources", "android-sdk", "platform-tools", "adb")
	contents, err := os.ReadFile(installed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "Android Debug Bridge") {
		t.Fatalf("installed adb was not retained")
	}
}

func TestInstallRejectsChecksumMismatch(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ANDROID_PLATFORM_TOOLS_URL", "http://127.0.0.1:1/unreachable")
	t.Setenv("ANDROID_PLATFORM_TOOLS_SHA256", strings.Repeat("0", 64))
	if err := Install(); err == nil {
		t.Fatal("expected install failure")
	}
}

func TestInstallArchiveExtractsTarGzToolchain(t *testing.T) {
	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	writer := tar.NewWriter(compressed)
	contents := []byte("#!/bin/sh\necho jdk\n")
	if err := writer.WriteHeader(&tar.Header{Name: "jdk-17/bin/javac", Mode: 0o700, Size: int64(len(contents))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(archive.Bytes())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive.Bytes())
	}))
	defer server.Close()
	destination := filepath.Join(t.TempDir(), "jdk-17")
	if err := InstallArchive(server.URL, hex.EncodeToString(digest[:]), destination, "JDK 17", "", "tar.gz"); err != nil {
		t.Fatal(err)
	}
	installed, err := os.ReadFile(filepath.Join(destination, "bin", "javac"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(contents) {
		t.Fatalf("unexpected extracted toolchain: %q", installed)
	}
}
