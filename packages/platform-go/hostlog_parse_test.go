package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseJournalFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "hostlog", "journal.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	entries := ParseJournalJSON(raw)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.Message == "" || entry.Unit != "systemd-udevd.service" || entry.Cursor != "s=cursor-1" || entry.PID != 42 {
		t.Fatalf("journal entry = %+v", entry)
	}
	if entry.Timestamp.IsZero() || entry.BootID != "boot-123" {
		t.Fatalf("journal timestamp/boot = %+v", entry)
	}
}

func TestParseMacOSFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "hostlog", "macos.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	entries := ParseMacOSNDJSON(raw)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.Message != "WindowServer: display state changed" || entry.Provider != "com.apple.WindowServer" {
		t.Fatalf("macOS entry = %+v", entry)
	}
	if entry.Timestamp.IsZero() || entry.Process == "" {
		t.Fatalf("macOS timestamp/process = %+v", entry)
	}
}

func TestParseWindowsFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "hostlog", "windows.json"))
	if err != nil {
		t.Fatal(err)
	}
	entries := ParseWindowsEventJSON(raw)
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.EventID != "17" || entry.Provider != "Microsoft-Windows-WHEA-Logger" || entry.Hostname != "WIN-HOST" {
		t.Fatalf("Windows entry = %+v", entry)
	}
	if entry.Timestamp.IsZero() || entry.Message == "" {
		t.Fatalf("Windows timestamp/message = %+v", entry)
	}
}
