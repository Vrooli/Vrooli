package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCleanupMaxBytesAcceptsHumanAndRawValues(t *testing.T) {
	if got := parseCleanupMaxBytes("1GiB"); got != 1<<30 {
		t.Fatalf("1GiB parsed as %d", got)
	}
	if got := parseCleanupMaxBytes("1073741824"); got != 1<<30 {
		t.Fatalf("raw bytes parsed as %d", got)
	}
	if got := parseCleanupMaxBytes("not-a-size"); got != 0 {
		t.Fatalf("invalid size parsed as %d", got)
	}
}

func TestCaptureCandidatesSelectsExpiredBundlesAndHonorsByteCap(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "old-bundle")
	newer := filepath.Join(root, "newer-bundle")
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newer, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "capture.png"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newer, "capture.png"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldAt := time.Now().Add(-48 * time.Hour)
	newAt := time.Now().Add(-time.Hour)
	if err := os.Chtimes(old, oldAt, oldAt); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, newAt, newAt); err != nil {
		t.Fatal(err)
	}

	service := &ownerCleanupService{capturesRoot: root}
	items, err := service.captureCandidates(context.Background(), 24*60*60, 0, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "capture:old-bundle" || items[0].Bytes != 3 {
		t.Fatalf("items = %#v, want only the expired old bundle", items)
	}
	if err := removeCapture(items[0].Path, root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old capture still exists: %v", err)
	}
	if _, err := os.Stat(newer); err != nil {
		t.Fatalf("new capture was unexpectedly removed: %v", err)
	}
}

func TestRemoveCaptureRejectsPathsOutsideRoot(t *testing.T) {
	if err := removeCapture(filepath.Join(t.TempDir(), "outside"), filepath.Join(t.TempDir(), "captures")); err == nil {
		t.Fatal("removeCapture accepted a path outside the configured root")
	}
}
