package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotatingFileWriterBoundsAndKeepsBackup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api.log")
	writer := &rotatingFileWriter{path: path, maxBytes: 4, maxBackups: 1}
	if err := writer.openFile(); err != nil {
		t.Fatalf("openFile() error = %v", err)
	}
	defer writer.file.Close()

	if _, err := writer.Write([]byte("12345")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	backups, err := filepath.Glob(path + ".*")
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backup count = %d, want 1", len(backups))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("active log missing after rotation: %v", err)
	}
}
