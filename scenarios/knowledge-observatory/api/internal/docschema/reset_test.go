package docschema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestResetDocumentProblems(t *testing.T) {
	now := time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC)
	content := strings.Join([]string{
		"# Problems",
		"",
		"## 2026-01-20: Recent issue",
		"Details.",
		"",
		"## 2025-11-01: Old issue",
		"Details.",
		"",
		"## 2025-10-01: Older issue",
		"Details.",
		"",
	}, "\n")

	filePath := filepath.Join(t.TempDir(), "PROBLEMS.md")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	config := ResetConfig{
		DocType:        DocTypeProblems,
		MaxAgeDays:     60,
		KeepMinEntries: 1,
		PreviewMode:    true,
	}
	result, err := resetDocumentWithClock(filePath, config, now)
	if err != nil {
		t.Fatalf("resetDocumentWithClock: %v", err)
	}
	if result.RemovedCount != 2 {
		t.Fatalf("expected 2 removed entries, got %d", result.RemovedCount)
	}
	if result.KeptCount != 1 {
		t.Fatalf("expected 1 kept entry, got %d", result.KeptCount)
	}
	if !strings.Contains(result.NewContent, "2026-01-20") {
		t.Fatalf("expected recent entry to remain")
	}
	if strings.Contains(result.NewContent, "2025-11-01") {
		t.Fatalf("expected old entry to be removed")
	}

	updated, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(updated) != content {
		t.Fatalf("expected preview mode to leave file unchanged")
	}
}

func TestResetDocumentProgress(t *testing.T) {
	now := time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC)
	content := strings.Join([]string{
		"| Date | Author | Status Snapshot | Notes |",
		"|------|--------|-----------------|-------|",
		"| 2026-01-10 | Dev | Recent | Note |",
		"| 2025-12-01 | Dev | Older | Note |",
		"| 2025-10-01 | Dev | Oldest | Note |",
		"",
	}, "\n")

	filePath := filepath.Join(t.TempDir(), "PROGRESS.md")
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	config := ResetConfig{
		DocType:        DocTypeProgress,
		MaxAgeDays:     30,
		KeepMinEntries: 2,
		PreviewMode:    false,
	}
	result, err := resetDocumentWithClock(filePath, config, now)
	if err != nil {
		t.Fatalf("resetDocumentWithClock: %v", err)
	}
	if result.RemovedCount != 1 {
		t.Fatalf("expected 1 removed entry, got %d", result.RemovedCount)
	}
	if result.KeptCount != 2 {
		t.Fatalf("expected 2 kept entries, got %d", result.KeptCount)
	}
	if !strings.Contains(result.NewContent, "2026-01-10") {
		t.Fatalf("expected recent entry to remain")
	}
	if !strings.Contains(result.NewContent, "2025-12-01") {
		t.Fatalf("expected keep-min entry to remain")
	}
	if strings.Contains(result.NewContent, "2025-10-01") {
		t.Fatalf("expected oldest entry to be removed")
	}

	updated, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(updated) != result.NewContent {
		t.Fatalf("expected file to be updated when preview disabled")
	}
}
