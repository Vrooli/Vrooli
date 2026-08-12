package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallDownloadsAndValidatesPlatformTools(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
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
	if err := install(); err != nil {
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
	if err := install(); err == nil {
		t.Fatal("expected install failure")
	}
}
