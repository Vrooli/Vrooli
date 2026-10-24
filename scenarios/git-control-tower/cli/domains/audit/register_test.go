package audit

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseFlags(t *testing.T) {
	got := parseFlags([]string{
		"--operation=commit",
		"--limit=25",
		"ignored",
	})

	if got.operation != "commit" {
		t.Fatalf("operation = %q, want commit", got.operation)
	}
	if got.limit != "25" {
		t.Fatalf("limit = %q, want 25", got.limit)
	}
}

func TestFormatEntryPrintsFailureDetailsAndTruncatesLongMessage(t *testing.T) {
	longMessage := "this commit message is deliberately longer than fifty characters so it should be shortened"
	entry := &entry{
		Operation:     "commit",
		Success:       false,
		Timestamp:     "2026-05-01T18:30:00Z",
		CommitHash:    "abcdef123456",
		CommitMessage: longMessage,
		Paths:         []string{"api/main.go", "ui/src/App.tsx"},
		Error:         "push rejected",
	}

	output := captureStdout(t, func() {
		formatEntry(entry)
	})

	for _, want := range []string{
		"[commit] FAIL 2026-05-01T18:30:00Z",
		"Commit: abcdef123456",
		"Message: this commit message is deliberately longer than...",
		"Paths: api/main.go, ui/src/App.tsx",
		"Error: push rejected",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("formatEntry output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, longMessage) {
		t.Fatalf("formatEntry should truncate long commit messages:\n%s", output)
	}
}

func TestFormatEntryPrintsMinimalSuccess(t *testing.T) {
	output := captureStdout(t, func() {
		formatEntry(&entry{
			Operation: "stage",
			Success:   true,
			Timestamp: "2026-05-01T18:45:00Z",
		})
	})

	if strings.TrimSpace(output) != "[stage] OK 2026-05-01T18:45:00Z" {
		t.Fatalf("unexpected minimal success output:\n%s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	t.Cleanup(func() {
		os.Stdout = original
	})
	os.Stdout = writer

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("closing stdout pipe writer failed: %v", err)
	}
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("reading stdout pipe failed: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("closing stdout pipe reader failed: %v", err)
	}
	return buf.String()
}
