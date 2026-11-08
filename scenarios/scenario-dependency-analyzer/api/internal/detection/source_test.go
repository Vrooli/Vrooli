package detection

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectionSourceRejectsOversizeAndCanceledReads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.go")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(maxDetectionSourceBytes + 1); err != nil {
		t.Fatal(err)
	}
	file.Close()
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if data, err := readDetectionSource(context.Background(), path, entries[0]); err == nil || len(data) != 0 {
		t.Fatalf("oversized source read: bytes=%d err=%v", len(data), err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readDetectionSource(ctx, path, entries[0]); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel ignored: %v", err)
	}
}
