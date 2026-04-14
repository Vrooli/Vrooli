package scenariohandlers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScenarioLogHelperReaders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "alpha.log")
	if err := os.WriteFile(path, []byte("one\ntwo\nthree\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}

	tail, err := ReadLastLogLines(path, 2)
	if err != nil {
		t.Fatalf("ReadLastLogLines() error = %v", err)
	}
	if string(tail) != "two\nthree\n" {
		t.Fatalf("tail = %q", string(tail))
	}

	delta, nextOffset, err := ReadScenarioLogDelta(path, int64(len("one\n")))
	if err != nil {
		t.Fatalf("ReadScenarioLogDelta() error = %v", err)
	}
	if string(delta) != "two\nthree\n" {
		t.Fatalf("delta = %q", string(delta))
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	if _, err := file.WriteString("four\n"); err != nil {
		_ = file.Close()
		t.Fatalf("append log: %v", err)
	}
	_ = file.Close()

	delta, _, err = ReadScenarioLogDelta(path, nextOffset)
	if err != nil {
		t.Fatalf("ReadScenarioLogDelta() appended error = %v", err)
	}
	if string(delta) != "four\n" {
		t.Fatalf("appended delta = %q", string(delta))
	}
}
