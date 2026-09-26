package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureAllDirs(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	r := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil)})

	paths, err := EnsureAllDirs(r, Options{ScenarioID: "demo", RootOverride: tmp}, 0)
	if err != nil {
		t.Fatalf("EnsureAllDirs() error = %v", err)
	}

	for _, p := range []string{paths.ConfigDir, paths.DataDir, paths.CacheDir, paths.LogsDir, paths.StateDir} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("Stat(%q) error = %v", p, err)
		}
		if !info.IsDir() {
			t.Fatalf("%q is not a directory", p)
		}
	}
}

func TestEnsureClassDir(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	r := mustResolver(t, ResolverConfig{AppID: "vrooli", EnvGet: mapEnv(nil)})

	p, err := EnsureClassDir(r, Options{ScenarioID: "demo", RootOverride: tmp}, ClassLogs, 0)
	if err != nil {
		t.Fatalf("EnsureClassDir() error = %v", err)
	}

	if p != filepath.Join(tmp, "logs", "vrooli", "demo") {
		t.Fatalf("path = %q", p)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "nested", "state.json")

	if err := WriteFileAtomic(target, []byte("first"), 0); err != nil {
		t.Fatalf("WriteFileAtomic(first) error = %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}
	if string(data) != "first" {
		t.Fatalf("first write content = %q", string(data))
	}

	if err := WriteFileAtomic(target, []byte("second"), 0); err != nil {
		t.Fatalf("WriteFileAtomic(second) error = %v", err)
	}
	data, err = os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if string(data) != "second" {
		t.Fatalf("second write content = %q", string(data))
	}
}

func TestWriteFileAtomicInRootContainsWrites(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if err := WriteFileAtomicInRoot(root, "nested/state.json", []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomicInRoot(root, "nested/state.json", []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	data, err := root.ReadFile("nested/state.json")
	if err != nil || string(data) != "second" {
		t.Fatalf("read: %q %v", data, err)
	}
	if err := WriteFileAtomicInRoot(root, "../escape", []byte("bad"), 0600); err == nil {
		t.Fatal("traversal accepted")
	}
	outside := t.TempDir()
	if err := root.Symlink(outside, "linked"); err != nil {
		t.Skip(err)
	}
	if err := WriteFileAtomicInRoot(root, "linked/escape", []byte("bad"), 0600); err == nil {
		t.Fatal("symlink escape accepted")
	}
	files, err := os.ReadDir(outside)
	if err != nil || len(files) != 0 {
		t.Fatalf("outside changed: %v %v", files, err)
	}
	files, err = rootEntries(root, "nested")
	if err != nil || len(files) != 1 {
		t.Fatalf("staging leaked: %v %v", files, err)
	}
}

func rootEntries(root *os.Root, path string) ([]os.DirEntry, error) {
	file, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return file.ReadDir(-1)
}
