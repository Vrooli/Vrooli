package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureClassDirUnknownClass(t *testing.T) {
	t.Parallel()

	r := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil)})
	_, err := EnsureClassDir(r, Options{ScenarioID: "demo", RootOverride: t.TempDir()}, Class("bad"), 0)
	if err == nil {
		t.Fatalf("expected unknown class error")
	}
}

func TestEnsureAllDirsFailsWhenPathBlockedByFile(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	blocked := filepath.Join(tmp, "config")
	if err := os.WriteFile(blocked, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocked file: %v", err)
	}

	r := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil)})
	_, err := EnsureAllDirs(r, Options{ScenarioID: "demo", RootOverride: tmp}, 0)
	if err == nil {
		t.Fatalf("expected error due to blocked config path")
	}
}

func TestWriteFileAtomicFailsWhenParentCannotBeCreated(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	blocker := filepath.Join(tmp, "blocked")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}

	target := filepath.Join(blocker, "child.txt")
	err := WriteFileAtomic(target, []byte("data"), 0)
	if err == nil {
		t.Fatalf("expected mkdir failure")
	}
}

func TestWriteFileAtomicFailsWhenTargetIsDirectory(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	targetDir := filepath.Join(tmp, "asdir")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}

	err := WriteFileAtomic(targetDir, []byte("data"), 0)
	if err == nil {
		t.Fatalf("expected rename failure when target is directory")
	}
}
