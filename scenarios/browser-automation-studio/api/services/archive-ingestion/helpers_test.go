package archiveingestion

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/storage"
)

func TestRecordingScreenshotUsesSharedStorageContentTypeOwner(t *testing.T) {
	payload := []byte("<!doctype html><html><body>captured</body></html>")
	archivePath := filepath.Join(t.TempDir(), "recording.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	entry, err := writer.Create("screenshots/frame.bin")
	if err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if _, err := entry.Write(payload); err != nil {
		_ = writer.Close()
		_ = file.Close()
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = archive.Close() })
	completed := time.Now().UTC()
	adapter := newRecordingAdapter(uuid.New(), uuid.New(), &recordingManifest{
		Viewport: recordingViewport{Width: 800, Height: 600},
	}, map[string]*zip.File{"screenshots/frame.bin": archive.File[0]}, nil)
	outcome := &autocontracts.StepOutcome{CompletedAt: &completed}

	attached, err := adapter.attachScreenshot(outcome, recordingFrame{Screenshot: "screenshots/frame.bin"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !attached || outcome.Screenshot == nil {
		t.Fatal("recorded screenshot was not attached")
	}
	wantType := storage.SniffContentType(bytes.NewReader(payload))
	if got := outcome.Screenshot.MediaType; got != wantType {
		t.Fatalf("screenshot media type=%q, shared storage owner reports %q", got, wantType)
	}
	if !bytes.Equal(outcome.Screenshot.Data, payload) {
		t.Fatal("screenshot payload changed while reading its content type")
	}
}
