package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupOldTranscripts_Disabled(t *testing.T) {
	root := t.TempDir()
	config := TranscriptCleanupConfig{Enabled: false}

	if err := CleanupOldTranscripts(root, config); err != nil {
		t.Fatalf("cleanup with disabled config should not error: %v", err)
	}
}

func TestCleanupOldTranscripts_EmptyDirectory(t *testing.T) {
	root := t.TempDir()
	config := DefaultTranscriptCleanupConfig()

	if err := CleanupOldTranscripts(root, config); err != nil {
		t.Fatalf("cleanup with empty directory should not error: %v", err)
	}
}

func TestCleanupOldTranscripts_MaxAge(t *testing.T) {
	root := t.TempDir()
	transcriptDir := filepath.Join(root, "tmp", "codex")
	if err := os.MkdirAll(transcriptDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create old file (8 days old)
	oldFile := filepath.Join(transcriptDir, "old-transcript.jsonl")
	if err := os.WriteFile(oldFile, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(oldFile, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}

	// Create recent file (1 day old)
	recentFile := filepath.Join(transcriptDir, "recent-transcript.jsonl")
	if err := os.WriteFile(recentFile, []byte("recent"), 0o644); err != nil {
		t.Fatal(err)
	}
	recentTime := time.Now().Add(-24 * time.Hour)
	if err := os.Chtimes(recentFile, recentTime, recentTime); err != nil {
		t.Fatal(err)
	}

	config := TranscriptCleanupConfig{
		MaxAge:   7 * 24 * time.Hour,
		MaxCount: 1000,
		Enabled:  true,
	}

	if err := CleanupOldTranscripts(root, config); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Old file should be deleted
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Error("old file should have been deleted")
	}

	// Recent file should be preserved
	if _, err := os.Stat(recentFile); err != nil {
		t.Error("recent file should have been preserved")
	}
}

func TestCleanupOldTranscripts_MaxCount(t *testing.T) {
	root := t.TempDir()
	transcriptDir := filepath.Join(root, "tmp", "codex")
	if err := os.MkdirAll(transcriptDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create 5 files with different ages
	now := time.Now()
	for i := 0; i < 5; i++ {
		filename := filepath.Join(transcriptDir, "transcript-"+string(rune('a'+i))+".jsonl")
		if err := os.WriteFile(filename, []byte("data"), 0o644); err != nil {
			t.Fatal(err)
		}
		fileTime := now.Add(-time.Duration(i) * time.Hour)
		if err := os.Chtimes(filename, fileTime, fileTime); err != nil {
			t.Fatal(err)
		}
	}

	config := TranscriptCleanupConfig{
		MaxAge:   365 * 24 * time.Hour, // Don't delete by age
		MaxCount: 3,                    // Keep only 3 newest
		Enabled:  true,
	}

	if err := CleanupOldTranscripts(root, config); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// Count remaining files
	entries, err := os.ReadDir(transcriptDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 files to remain, got %d", len(entries))
	}
}

func TestDefaultTranscriptCleanupConfig(t *testing.T) {
	config := DefaultTranscriptCleanupConfig()

	if !config.Enabled {
		t.Error("default config should be enabled")
	}
	if config.MaxAge != 3*24*time.Hour {
		t.Errorf("expected MaxAge of 3 days, got %v", config.MaxAge)
	}
	if config.MaxCount != 50 {
		t.Errorf("expected MaxCount of 50, got %d", config.MaxCount)
	}
}
