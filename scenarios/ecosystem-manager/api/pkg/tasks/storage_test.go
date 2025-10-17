package tasks

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveQueueItemRemovesDuplicates(t *testing.T) {
	tmp := t.TempDir()
	for _, status := range queueStatuses {
		if err := os.MkdirAll(filepath.Join(tmp, status), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	storage := NewStorage(tmp)

	// Seed an existing copy in the failed queue.
	duplicatePath := filepath.Join(tmp, "failed", "task-123.yaml")
	if err := os.WriteFile(duplicatePath, []byte("id: task-123\nstatus: failed\n"), 0o644); err != nil {
		t.Fatalf("seed duplicate: %v", err)
	}

	item := TaskItem{ID: "task-123", Status: "pending"}
	if err := storage.SaveQueueItem(item, "pending"); err != nil {
		t.Fatalf("SaveQueueItem: %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmp, "pending", "task-123.yaml")); err != nil {
		t.Fatalf("expected pending copy: %v", err)
	}

	if _, err := os.Stat(duplicatePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected failed copy removed, got err=%v", err)
	}
}

func TestCleanupDuplicatesKeepsNewestCopy(t *testing.T) {
	tmp := t.TempDir()
	for _, status := range queueStatuses {
		if err := os.MkdirAll(filepath.Join(tmp, status), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	storage := NewStorage(tmp)

	newerPath := filepath.Join(tmp, "pending", "task-abc.yaml")
	olderPath := filepath.Join(tmp, "failed", "task-abc.yaml")

	if err := os.WriteFile(olderPath, []byte("id: task-abc\nstatus: failed\n"), 0o644); err != nil {
		t.Fatalf("write older: %v", err)
	}
	// Set older timestamp.
	olderTime := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(olderPath, olderTime, olderTime); err != nil {
		t.Fatalf("chtimes older: %v", err)
	}

	if err := os.WriteFile(newerPath, []byte("id: task-abc\nstatus: pending\n"), 0o644); err != nil {
		t.Fatalf("write newer: %v", err)
	}

	if err := storage.CleanupDuplicates(); err != nil {
		t.Fatalf("CleanupDuplicates: %v", err)
	}

	if _, err := os.Stat(newerPath); err != nil {
		t.Fatalf("expected newer copy retained: %v", err)
	}

	if _, err := os.Stat(olderPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected older copy removed, err=%v", err)
	}
}
