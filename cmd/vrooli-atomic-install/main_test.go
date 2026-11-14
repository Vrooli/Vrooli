package main

import (
	"bytes"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"os"
	"path/filepath"
	"testing"
)

func TestInvalidCompiledAuthorityRefusesBeforeAtomicMutation(t *testing.T) {
	old := buildinfo.RuntimeHomeSnapshotBase64
	buildinfo.RuntimeHomeSnapshotBase64 = "invalid"
	t.Cleanup(func() { buildinfo.RuntimeHomeSnapshotBase64 = old })
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "candidate"), filepath.Join(dir, "installed")
	if err := os.WriteFile(src, []byte("candidate"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("prior"), 0700); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if code := run([]string{"installer", src, dst}, &stderr); code != 1 {
		t.Fatalf("invalid stamped authority accepted: code%d", code)
	}
	got, _ := os.ReadFile(dst)
	if string(got) != "prior" {
		t.Fatal("invalid authority mutated installed image")
	}
}
func TestUsagePreservesExitAndDoesNotInitializeAuthority(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"installer"}, &stderr); code != usageExitCode {
		t.Fatal(code)
	}
}
