package cliresolve

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutableReturnsAbsoluteInstalledPath(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".vrooli", "bin", "system-monitor")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Executable(home, "system-monitor")
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(path)
	if got != want {
		t.Fatalf("Executable = %q, want %q", got, want)
	}
}

func TestExecutableReturnsTypedErrorWithSearchDirectories(t *testing.T) {
	home := filepath.Join(t.TempDir(), "missing-home")
	_, err := Executable(home, "missing-cli")
	if err == nil {
		t.Fatal("Executable unexpectedly succeeded")
	}
	var notFound *NotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("error type = %T, want *NotFoundError: %v", err, err)
	}
	if notFound.Name != "missing-cli" || len(notFound.Searched) != 1 {
		t.Fatalf("not-found details = %#v", notFound)
	}
	if !strings.Contains(err.Error(), notFound.Searched[0]) {
		t.Fatalf("error %q does not name searched directory %q", err, notFound.Searched[0])
	}
}

func TestExecutableReportsEmptyHomeWithoutPanicking(t *testing.T) {
	r := &Resolver{}
	_, err := r.Executable("missing-cli")
	if err == nil {
		t.Fatal("Executable unexpectedly succeeded")
	}
	if !IsNotFound(err) && !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("unexpected empty-home error: %v", err)
	}
}
