package cliutil

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestEnumerateInputsUsesGitOutputAndDoesNotUseGitForVerdict(t *testing.T) {
	root := t.TempDir()
	var got []string
	files, source, err := EnumerateInputs(root, []string{"api"}, EnumerateDeps{
		Command: func(name string, args ...string) ([]byte, error) {
			got = append([]string{name}, args...)
			return []byte("api/main.go\x00api/go.mod\x00"), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if source != EnumerationGit || !reflect.DeepEqual(files, []string{"api/go.mod", "api/main.go"}) {
		t.Fatalf("enumeration = %v, %v", source, files)
	}
	if got[0] != "git" {
		t.Fatalf("command = %v", got)
	}
}

func TestEnumerateInputsFallsBackToWalkAndGatesExecutableSuffix(t *testing.T) {
	root := t.TempDir()
	for name, content := range map[string]string{
		"api/main.go":  "package main",
		"api/tool.exe": "source",
		"api/run.log":  "output",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, source, err := EnumerateInputs(root, []string{"api"}, EnumerateDeps{
		Command:          func(string, ...string) ([]byte, error) { return nil, os.ErrNotExist },
		ExecutableSuffix: ".exe",
	})
	if err != nil {
		t.Fatal(err)
	}
	if source != EnumerationWalk || !reflect.DeepEqual(files, []string{"api/main.go"}) {
		t.Fatalf("walk enumeration = %v, %v", source, files)
	}
}

// The walk fallback skips the build-output directories the repository
// ignores, while a source package that merely shares a name stays visible.
func TestEnumerateInputsWalkSkipsIgnoredBuildOutput(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"scenarios/app/api/main.go",
		"scenarios/app/api/bin/api",
		"scenarios/app/api/.venv/lib/python/site.py",
		"scenarios/app/api/bundle/packager.go",
		"scenarios/app/platforms/electron/dist-electron/app",
		"scenarios/app/platforms/electron/bundle/runtime/app",
		"scenarios/app/platforms/electron/src/main.ts",
	} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, source, err := EnumerateInputs(root, []string{"."}, EnumerateDeps{
		Command: func(string, ...string) ([]byte, error) { return nil, os.ErrNotExist },
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"scenarios/app/api/bundle/packager.go", "scenarios/app/api/main.go", "scenarios/app/platforms/electron/src/main.ts"}
	if source != EnumerationWalk || !reflect.DeepEqual(files, want) {
		t.Fatalf("walk enumeration = %v %v, want %v", source, files, want)
	}
}

func TestBuildOutputSkipSuffixesFor(t *testing.T) {
	if got := BuildOutputSkipSuffixesFor(""); len(got) != 3 {
		t.Fatalf("unix suffixes = %v", got)
	}
	if got := BuildOutputSkipSuffixesFor(".exe"); len(got) != 4 || got[3] != ".exe" {
		t.Fatalf("windows suffixes = %v", got)
	}
}
