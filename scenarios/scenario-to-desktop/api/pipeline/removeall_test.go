package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveAllRobust_RemovesNonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "root")
	if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a", "b", "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := removeAllRobust(root); err != nil {
		t.Fatalf("removeAllRobust: %v", err)
	}
	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected root removed; stat err=%v", err)
	}
}

func TestRemoveAllRobust_RefusesRoot(t *testing.T) {
	if err := removeAllRobust("/"); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCleanDesktopOutputsPreservesFrameworkSourceOutsideStaging(t *testing.T) {
	framework := filepath.Join(t.TempDir(), "platforms", "electron")
	for _, relative := range []string{"package.json", "src/main.ts", "bundle/bundle.json", "dist-electron/app.AppImage", "dist/installer", "dist-dev/temp", "dist-dev-electron/temp"} {
		path := filepath.Join(framework, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(relative), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := cleanDesktopOutputs(framework, "proper"); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{"package.json", "src/main.ts"} {
		if _, err := os.Stat(filepath.Join(framework, relative)); err != nil {
			t.Fatalf("source %q was removed: %v", relative, err)
		}
	}
	for _, relative := range []string{"bundle", "dist-electron", "dist", "dist-dev", "dist-dev-electron"} {
		if _, err := os.Stat(filepath.Join(framework, relative)); !os.IsNotExist(err) {
			t.Fatalf("output %q remains: %v", relative, err)
		}
	}
}

func TestCleanDesktopOutputsRemovesEntireStagingFrameworkAndRejectsEmptyPath(t *testing.T) {
	framework := filepath.Join(t.TempDir(), "staging", "electron")
	if err := os.MkdirAll(framework, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(framework, "generated"), []byte("generated"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cleanDesktopOutputs(framework, "staging"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(framework); !os.IsNotExist(err) {
		t.Fatalf("staging framework remains: %v", err)
	}
	if err := cleanDesktopOutputs(" ", "proper"); err == nil {
		t.Fatal("expected empty path rejection")
	}
	if err := forceRemoveTree(filepath.Join(t.TempDir(), "missing")); err != nil {
		t.Fatalf("missing tree removal = %v", err)
	}
}
