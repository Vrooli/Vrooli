package archiveingestion

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/storage"
)

func TestRecordingFileStoreScreenshotCallersPublishThroughOneOwner(t *testing.T) {
	root := t.TempDir()
	executionID := uuid.New()
	store := newRecordingFileStore(root, logrus.New())
	data := []byte("same screenshot payload")
	input := filepath.Join(t.TempDir(), "source.png")
	if err := os.WriteFile(input, data, 0o600); err != nil {
		t.Fatal(err)
	}

	fromBytes, err := store.StoreScreenshot(context.Background(), executionID, "bytes", data, "image/png")
	if err != nil {
		t.Fatal(err)
	}
	fromFile, err := store.StoreScreenshotFromFile(context.Background(), executionID, "file", input)
	if err != nil {
		t.Fatal(err)
	}
	for name, info := range map[string]*storage.ScreenshotInfo{"bytes": fromBytes, "file": fromFile} {
		if info == nil || info.SizeBytes != int64(len(data)) {
			t.Fatalf("%s caller info = %+v", name, info)
		}
		path := filepath.Join(root, executionID.String(), "frames", filepath.Base(info.URL))
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s caller output: %v", name, err)
		}
		if string(got) != string(data) {
			t.Fatalf("%s caller changed payload: %q", name, got)
		}
	}
	entries, err := os.ReadDir(filepath.Join(root, executionID.String(), "frames"))
	if err != nil || len(entries) != 2 {
		t.Fatalf("published entries=%d err=%v, want two complete screenshot files", len(entries), err)
	}
}
