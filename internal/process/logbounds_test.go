package process

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBoundLogsTrimsOversizedLogsAndKeepsTheirTail(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "scenarios", "ui-health")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	big := filepath.Join(dir, "vrooli.develop.ui-health.start-api.log")
	content := append(bytes.Repeat([]byte("old line\n"), 20000), []byte("most recent line\n")...)
	if err := os.WriteFile(big, content, 0o600); err != nil {
		t.Fatal(err)
	}
	small := filepath.Join(dir, "vrooli.develop.ui-health.start-ui.log")
	if err := os.WriteFile(small, []byte("small\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(dir, "state.json")
	if err := os.WriteFile(other, bytes.Repeat([]byte("x"), 200000), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := BoundLogs(root, 64<<10, 1<<10)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Trimmed) != 1 || result.Trimmed[0] != big {
		t.Fatalf("trimmed = %v, want only %s", result.Trimmed, big)
	}
	tail, err := os.ReadFile(big + ".1")
	if err != nil {
		t.Fatal(err)
	}
	if len(tail) != 1<<10 || !strings.HasSuffix(string(tail), "most recent line\n") {
		t.Fatalf("tail = %d bytes, ends %q", len(tail), tail[len(tail)-20:])
	}
	trimmed, err := os.ReadFile(big)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(trimmed), "[vrooli] log exceeded its size bound") {
		t.Fatalf("trimmed log = %q", trimmed)
	}
	if got, _ := os.ReadFile(small); string(got) != "small\n" {
		t.Fatalf("small log changed: %q", got)
	}
	if info, _ := os.Stat(other); info.Size() != 200000 {
		t.Fatalf("non-log file changed size: %d", info.Size())
	}
}

// A live process keeps writing to its log after a trim. With O_APPEND (how the
// lifecycle opens step logs) its next write lands after the marker, not at
// its old offset.
func TestBoundLogsKeepsAnAppendWriterAtTheNewEnd(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "api.log")
	writer, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	if _, err := writer.Write(bytes.Repeat([]byte("noise\n"), 30000)); err != nil {
		t.Fatal(err)
	}
	if _, err := BoundLogs(root, 64<<10, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.WriteString("after trim\n"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "[vrooli] log exceeded its size bound; earlier output was discarded\nafter trim\n"
	if string(got) != want {
		t.Fatalf("log after trim = %q, want %q", got, want)
	}
}

func TestBoundLogsIgnoresAMissingRoot(t *testing.T) {
	result, err := BoundLogs(filepath.Join(t.TempDir(), "absent"), 0, 0)
	if err != nil || result.Scanned != 0 {
		t.Fatalf("result = %+v, err = %v", result, err)
	}
}
