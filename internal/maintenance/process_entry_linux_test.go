//go:build linux

package maintenance

import (
	"os"
	"testing"
)

func TestParseProcStatEntryHandlesParensAndSpacesInComm(t *testing.T) {
	entry, ok := parseProcStatEntry([]byte("4242 (tmux: server (x)) S 1 4242 4242 0 -1 4194560 1 0 0 0\n"))
	if !ok {
		t.Fatal("expected stat line to parse")
	}
	if entry.PID != 4242 || entry.PPID != 1 || entry.PGID != 4242 || entry.SID != 4242 {
		t.Fatalf("entry = %#v", entry)
	}
	if entry.State != "S" {
		t.Fatalf("state = %q, want S", entry.State)
	}
	if entry.Command != "tmux: server (x)" {
		t.Fatalf("command fallback = %q", entry.Command)
	}
}

func TestParseProcStatEntryRejectsMalformedLines(t *testing.T) {
	for _, line := range []string{"", "12345", "12345 (comm", "abc (comm) S 1 2 3", "1 (c) S"} {
		if _, ok := parseProcStatEntry([]byte(line)); ok {
			t.Fatalf("expected %q to be rejected", line)
		}
	}
}

func TestReadProcessEntryReadsSelfWithoutForking(t *testing.T) {
	entry, ok := readProcessEntry(os.Getpid())
	if !ok {
		t.Fatal("expected to read own process entry")
	}
	if entry.PID != os.Getpid() {
		t.Fatalf("pid = %d, want %d", entry.PID, os.Getpid())
	}
	if entry.Command == "" {
		t.Fatal("expected non-empty command")
	}
	if entry.Executable == "" {
		t.Fatal("expected /proc/self/exe to resolve")
	}
}

func TestReadProcessEntryReportsGoneForMissingPID(t *testing.T) {
	if _, ok := readProcessEntry(1 << 22); ok {
		t.Fatal("expected missing PID to report not found")
	}
}
