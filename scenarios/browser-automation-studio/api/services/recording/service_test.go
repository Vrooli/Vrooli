package recording

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func writeRecordingArchive(t *testing.T, dir string, manifest any) string {
	t.Helper()

	path := filepath.Join(dir, "recording.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	w, err := zw.Create("manifest.json")
	if err != nil {
		t.Fatalf("create manifest entry: %v", err)
	}
	if _, err := w.Write(data); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	return path
}

func TestImportArchiveRejectsMissingFrames(t *testing.T) {
	dir := t.TempDir()
	archive := writeRecordingArchive(t, dir, map[string]any{
		"runId":  "r1",
		"frames": []any{},
	})

	svc := NewRecordingService(nil, nil, nil, logrus.New(), dir)
	_, err := svc.ImportArchive(context.Background(), archive, RecordingImportOptions{})
	if err == nil || err.Error() == "" {
		t.Fatalf("expected error for missing frames")
	}
	if err != ErrRecordingManifestMissingFrames {
		t.Fatalf("expected ErrRecordingManifestMissingFrames, got %v", err)
	}
}

func TestImportArchiveRejectsTooManyFrames(t *testing.T) {
	dir := t.TempDir()
	frames := make([]any, maxRecordingFrames()+1)
	for i := range frames {
		frames[i] = map[string]any{"index": i, "timestamp": i, "stepType": "navigate", "nodeId": "n"}
	}
	archive := writeRecordingArchive(t, dir, map[string]any{
		"runId":  "r1",
		"frames": frames,
	})

	svc := NewRecordingService(nil, nil, nil, logrus.New(), dir)
	_, err := svc.ImportArchive(context.Background(), archive, RecordingImportOptions{})
	if err == nil {
		t.Fatalf("expected error for too many frames")
	}
	if !strings.Contains(err.Error(), ErrRecordingTooManyFrames.Error()) {
		t.Fatalf("expected error to contain %q, got %v", ErrRecordingTooManyFrames.Error(), err)
	}
}

func TestImportArchiveRejectsTooLargeArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "too-large.zip")

	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Truncate(path, maxRecordingArchiveBytes()+1); err != nil {
		t.Fatalf("truncate file: %v", err)
	}

	svc := NewRecordingService(nil, nil, nil, logrus.New(), dir)
	_, err := svc.ImportArchive(context.Background(), path, RecordingImportOptions{})
	if err != ErrRecordingArchiveTooLarge {
		t.Fatalf("expected ErrRecordingArchiveTooLarge, got %v", err)
	}
}
