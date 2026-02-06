package docschema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppendProblemEntry(t *testing.T) {
	now := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)
	filePath := filepath.Join(t.TempDir(), "PROBLEMS.md")

	result, err := appendEntryWithClock(filePath, AppendConfig{
		DocType: DocTypeProblems,
		Title:   "Test issue",
		Body:    "Something went wrong.",
	}, now)
	if err != nil {
		t.Fatalf("appendEntryWithClock: %v", err)
	}
	if result.FilePath != filePath {
		t.Fatalf("unexpected file path: %s", result.FilePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "## 2026-02-05: Test issue") {
		t.Fatalf("expected problem heading in content:\n%s", text)
	}
	if !strings.Contains(text, "### Problem\nSomething went wrong.") {
		t.Fatalf("expected problem body in content:\n%s", text)
	}

	// Round-trip: parseProblemEntries should find the entry
	lines := strings.Split(text, "\n")
	_, entries := parseProblemEntries(lines)
	if len(entries) != 1 {
		t.Fatalf("expected 1 parsed entry, got %d", len(entries))
	}
	if !entries[0].dateValid {
		t.Fatalf("expected valid date")
	}
	if entries[0].date != time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC) {
		t.Fatalf("unexpected date: %v", entries[0].date)
	}
}

func TestAppendProblemEntryNoBody(t *testing.T) {
	now := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)
	filePath := filepath.Join(t.TempDir(), "PROBLEMS.md")

	_, err := appendEntryWithClock(filePath, AppendConfig{
		DocType: DocTypeProblems,
		Title:   "Minimal entry",
	}, now)
	if err != nil {
		t.Fatalf("appendEntryWithClock: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "## 2026-02-05: Minimal entry") {
		t.Fatalf("expected heading:\n%s", text)
	}
	if strings.Contains(text, "### Problem") {
		t.Fatalf("expected no problem body section when body is empty")
	}
}

func TestAppendProgressEntry(t *testing.T) {
	now := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)
	filePath := filepath.Join(t.TempDir(), "PROGRESS.md")

	result, err := appendEntryWithClock(filePath, AppendConfig{
		DocType: DocTypeProgress,
		Title:   "Added feature X",
		Author:  "Claude Code",
		Status:  "done",
	}, now)
	if err != nil {
		t.Fatalf("appendEntryWithClock: %v", err)
	}
	if result.FilePath != filePath {
		t.Fatalf("unexpected file path: %s", result.FilePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "| 2026-02-05 | Claude Code | done | Added feature X |") {
		t.Fatalf("expected progress row in content:\n%s", text)
	}

	// Round-trip: parseProgressEntries should find the entry
	lines := strings.Split(text, "\n")
	_, entries := parseProgressEntries(lines)
	if len(entries) != 1 {
		t.Fatalf("expected 1 parsed entry, got %d", len(entries))
	}
	if !entries[0].dateValid {
		t.Fatalf("expected valid date")
	}
}

func TestAppendToExistingFile(t *testing.T) {
	now := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)
	existing := "# Problems\n\n## 2026-01-01: Old issue\nDetails.\n\n---\n"
	filePath := filepath.Join(t.TempDir(), "PROBLEMS.md")
	if err := os.WriteFile(filePath, []byte(existing), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := appendEntryWithClock(filePath, AppendConfig{
		DocType: DocTypeProblems,
		Title:   "New issue",
		Body:    "New details.",
	}, now)
	if err != nil {
		t.Fatalf("appendEntryWithClock: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(content)

	lines := strings.Split(text, "\n")
	_, entries := parseProblemEntries(lines)
	if len(entries) != 2 {
		t.Fatalf("expected 2 parsed entries, got %d", len(entries))
	}
}

func TestAppendRequiresTitle(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "PROBLEMS.md")
	_, err := appendEntryWithClock(filePath, AppendConfig{
		DocType: DocTypeProblems,
		Title:   "",
	}, time.Now())
	if err == nil {
		t.Fatalf("expected error for empty title")
	}
}

func TestAppendRejectsUnsupportedDocType(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "SEAMS.md")
	_, err := appendEntryWithClock(filePath, AppendConfig{
		DocType: DocTypeSeams,
		Title:   "Test",
	}, time.Now())
	if err == nil {
		t.Fatalf("expected error for unsupported doc type")
	}
}

func TestAppendProgressDefaultAuthorStatus(t *testing.T) {
	now := time.Date(2026, 2, 5, 12, 0, 0, 0, time.UTC)
	filePath := filepath.Join(t.TempDir(), "PROGRESS.md")

	_, err := appendEntryWithClock(filePath, AppendConfig{
		DocType: DocTypeProgress,
		Title:   "Something happened",
	}, now)
	if err != nil {
		t.Fatalf("appendEntryWithClock: %v", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "| 2026-02-05 | system | done | Something happened |") {
		t.Fatalf("expected default author/status in row:\n%s", text)
	}
}
