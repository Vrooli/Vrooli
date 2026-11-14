package doclogs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"knowledge-observatory/internal/doccontract"
)

func TestAppendDatedMarkdownSection(t *testing.T) {
	path := writeLogDoc(t, "# Problems\n\n## Entries\n\n_None yet._\n\n## Architecture Drift\n\nKeep me.\n")
	op := doccontract.AppendLogOperation{
		Enabled:       true,
		TargetHeading: "Entries",
		Format:        "dated-markdown-section",
		EmptyMarker:   "_None yet._",
		Fields:        []string{"title", "body"},
		Retention:     doccontract.AppendLogRetention{SupportsReset: true, DateSource: "heading"},
	}

	result, err := appendWithClock(path, op, Entry{Title: "Known issue", Body: "Details."}, time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if !strings.Contains(result.EntryAdded, "### 2026-05-10 - Known issue") {
		t.Fatalf("unexpected entry: %s", result.EntryAdded)
	}
	content := readLogDoc(t, path)
	if strings.Contains(content, "_None yet._") {
		t.Fatalf("empty marker was not removed:\n%s", content)
	}
	if !strings.Contains(content, "## Architecture Drift\n\nKeep me.") {
		t.Fatalf("content after append region was not preserved:\n%s", content)
	}
}

func TestResetDatedMarkdownSectionPreservesNonLogSections(t *testing.T) {
	path := writeLogDoc(t, "# Problems\n\n## Entries\n\n### 2024-01-01 - Old\n\nOld body.\n\n### 2026-05-10 - New\n\nNew body.\n\n## Architecture Drift\n\nKeep me.\n")
	op := doccontract.AppendLogOperation{
		Enabled:       true,
		TargetHeading: "Entries",
		Format:        "dated-markdown-section",
		EmptyMarker:   "_None yet._",
		Retention:     doccontract.AppendLogRetention{SupportsReset: true, DateSource: "heading"},
	}

	result, err := resetWithClock(path, op, ResetConfig{MaxAgeDays: 30, KeepMinEntries: 1}, time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if result.RemovedCount != 1 || result.KeptCount != 1 {
		t.Fatalf("unexpected counts: removed=%d kept=%d", result.RemovedCount, result.KeptCount)
	}
	content := readLogDoc(t, path)
	if strings.Contains(content, "2024-01-01") || !strings.Contains(content, "2026-05-10") {
		t.Fatalf("unexpected retained entries:\n%s", content)
	}
	if !strings.Contains(content, "## Architecture Drift\n\nKeep me.") {
		t.Fatalf("non-log section was not preserved:\n%s", content)
	}
}

func TestAppendAndResetMarkdownTable(t *testing.T) {
	path := writeLogDoc(t, "# Progress\n\n## Progress Log\n\n| Date | Author | Status | Notes |\n|---|---|---|---|\n| _No progress entries yet._ |  |  |  |\n")
	op := doccontract.AppendLogOperation{
		Enabled:       true,
		TargetHeading: "Progress Log",
		Format:        "markdown-table",
		EmptyMarker:   "_No progress entries yet._",
		Fields:        []string{"date", "author", "status", "notes"},
		Retention:     doccontract.AppendLogRetention{SupportsReset: true, DateSource: "first-column"},
	}

	if _, err := appendWithClock(path, op, Entry{Author: "tester", Status: "done", Body: "Completed work"}, time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Append: %v", err)
	}
	content := readLogDoc(t, path)
	if strings.Contains(content, "_No progress entries yet._") || !strings.Contains(content, "| 2026-05-10 | tester | done | Completed work |") {
		t.Fatalf("unexpected table append:\n%s", content)
	}

	result, err := resetWithClock(path, op, ResetConfig{MaxAgeDays: 1}, time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if result.RemovedCount != 1 || !strings.Contains(result.NewContent, "| _No progress entries yet._ |  |  |  |") {
		t.Fatalf("expected reset to restore empty marker, got removed=%d:\n%s", result.RemovedCount, result.NewContent)
	}
}

func writeLogDoc(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "LOG.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func readLogDoc(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(data)
}
