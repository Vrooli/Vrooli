package rewrite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestFSExecutorFileMove(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "a.go"), "package src\n")

	exec := NewFSExecutor()
	if err := exec.Execute(context.Background(), root, FileMove{From: "src/a.go", To: "dst/b.go"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "src", "a.go")); !os.IsNotExist(err) {
		t.Fatalf("from path should be gone; err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "dst", "b.go")); err != nil {
		t.Fatalf("to path missing: %v", err)
	}
}

func TestFSExecutorFileMoveRejectsEscape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	exec := NewFSExecutor()
	err := exec.Execute(context.Background(), root, FileMove{From: "../x", To: "y"})
	if err == nil {
		t.Fatal("expected error for path escape")
	}
}

func TestFSExecutorImportRewrite(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	src := `package m

import (
	"fmt"
	"example.com/old"
)

func F() { fmt.Println(old.X) }
`
	target := filepath.Join(root, "m", "m.go")
	writeFile(t, target, src)
	// File without matching import should be untouched.
	untouched := filepath.Join(root, "m", "other.go")
	writeFile(t, untouched, "package m\n")

	exec := NewFSExecutor()
	if err := exec.Execute(context.Background(), root, ImportRewrite{Old: "example.com/old", New: "example.com/new"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	gotStr := string(got)
	if !strings.Contains(gotStr, `"example.com/new"`) {
		t.Fatalf("new import missing from file:\n%s", gotStr)
	}
	if strings.Contains(gotStr, `"example.com/old"`) {
		t.Fatalf("old import still present:\n%s", gotStr)
	}

	if got2, _ := os.ReadFile(untouched); string(got2) != "package m\n" {
		t.Fatalf("unrelated file mutated: %q", string(got2))
	}
}
