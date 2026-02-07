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
