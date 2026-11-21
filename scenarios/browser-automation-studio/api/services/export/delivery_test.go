package export

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

type failingCompletion struct{ err error }

func (r failingCompletion) UpdateExportComplete(context.Context, uuid.UUID, string, int64, database.JSONMap) error {
	return r.err
}

func TestPublishExportPreservesPriorFilesAndRefusesEscapes(t *testing.T) {
	dir := t.TempDir()
	produce := func(path string) error { return os.WriteFile(path, []byte("complete-media"), 0600) }
	for _, name := range []string{"../escape.mp4", "a/b.mp4", `a\b.mp4`, "..", ""} {
		if _, err := PublishFile(t.Context(), dir, name, produce); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	first, err := PublishFile(t.Context(), dir, "film.mp4", produce)
	if err != nil {
		t.Fatal(err)
	}
	if first.Size != 14 || len(first.SHA256) != 64 {
		t.Fatalf("receipt %+v", first)
	}
	if _, err = PublishFile(t.Context(), dir, "film.mp4", produce); err == nil {
		t.Fatal("overwrote existing file")
	}
	bytes, _ := os.ReadFile(first.Path)
	if string(bytes) != "complete-media" {
		t.Fatal("changed prior file")
	}
	if _, err = PublishFile(t.Context(), dir, "partial.mp4", func(path string) error { _ = produce(path); return errors.New("interrupted") }); err == nil {
		t.Fatal("lost producer error")
	}
	if _, err = os.Stat(filepath.Join(dir, "partial.mp4")); !os.IsNotExist(err) {
		t.Fatal("published partial file")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("staging leaked: %v", entries)
	}
}
func TestExportCommitFailureRetainsMediaAndRecovery(t *testing.T) {
	dir := t.TempDir()
	media, err := PublishFile(t.Context(), dir, "film.mp4", func(path string) error { return os.WriteFile(path, []byte("media"), 0600) })
	if err != nil {
		t.Fatal(err)
	}
	row := &database.ExportIndex{ID: uuid.New(), Settings: database.JSONMap{"selected_source": "recorded_video"}}
	if err := CommitExport(t.Context(), failingCompletion{errors.New("database unavailable")}, row, media); err == nil {
		t.Fatal("reported successful finalization")
	}
	if _, err := os.Stat(media.Path); err != nil {
		t.Fatal("lost completed media", err)
	}
	receipt, err := os.ReadFile(media.Path + ".recovery.json")
	if err != nil || len(receipt) == 0 {
		t.Fatal("missing recoverable receipt", err)
	}
}
func TestPublishExportCancellationAndSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("prior"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "film.mp4")); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := PublishFile(ctx, dir, "cancelled.mp4", func(string) error { t.Fatal("producer called after cancel"); return nil }); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := PublishFile(t.Context(), dir, "film.mp4", func(path string) error { return os.WriteFile(path, []byte("replacement"), 0600) }); err == nil {
		t.Fatal("followed symlink")
	}
	b, _ := os.ReadFile(outside)
	if string(b) != "prior" {
		t.Fatal("changed symlink target")
	}
}

func TestPublishExportRefusesProducerSymlinkAndConcurrentCollision(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "source")
	if err := os.WriteFile(outside, []byte("prior"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := PublishFile(t.Context(), dir, "symlink.mp4", func(path string) error {
		if err := os.Remove(path); err != nil {
			return err
		}
		return os.Symlink(outside, path)
	}); err == nil {
		t.Fatal("published producer symlink")
	}
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := PublishFile(t.Context(), dir, "same.mp4", func(path string) error { return os.WriteFile(path, []byte("complete"), 0600) })
			results <- err
		}()
	}
	successes := 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("collision winners=%d", successes)
	}
}
