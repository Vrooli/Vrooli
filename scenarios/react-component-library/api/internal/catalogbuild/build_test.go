package catalogbuild

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRequiresExplicitRepositoryRoot(t *testing.T) {
	if _, err := Build("", Options{}); err == nil {
		t.Fatal("expected an empty repository root to be rejected")
	}
}

func TestBuildCheckUsesTheSingleGeneratorEntryPoint(t *testing.T) {
	root := repositoryRoot(t, filepath.Join("packages", "react-component-library", "tooling", "catalog-build.mjs"))
	report, err := Build(root, Options{Check: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Generator != "catalog-build/v1" || report.Status != "ok" || !report.Check {
		t.Fatalf("unexpected generator report: %+v", report)
	}
}

func repositoryRoot(t *testing.T, marker string) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for dir := filepath.Clean(cwd); ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return dir
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat repository marker %q: %v", filepath.Join(dir, marker), err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	t.Fatalf("repository marker %q not found above %q", marker, cwd)
	return ""
}
