package catalogbuild

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestBuildRequiresExplicitRepositoryRoot(t *testing.T) {
	if _, err := Build("", Options{}); err == nil {
		t.Fatal("expected an empty repository root to be rejected")
	}
}

func TestBuildCheckUsesTheSingleGeneratorEntryPoint(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not locate catalogbuild package")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../"))
	report, err := Build(root, Options{Check: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.Generator != "catalog-build/v1" || report.Status != "ok" || !report.Check {
		t.Fatalf("unexpected generator report: %+v", report)
	}
}
