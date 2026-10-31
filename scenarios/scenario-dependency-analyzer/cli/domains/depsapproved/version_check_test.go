package depsapproved

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckModuleVersionsDetectsDrift(t *testing.T) {
	root := t.TempDir()
	writeCheckModule(t, root, "one", "modernc.org/sqlite v1.50.1\n")
	writeCheckModule(t, root, "two", "modernc.org/sqlite v1.51.0\n")
	report, err := checkModuleVersions(root, []string{"modernc.org/sqlite"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Violations) != 1 || len(report.Violations[0].Versions) != 2 {
		t.Fatalf("report = %#v, want one two-version violation", report)
	}
}

func TestCheckModuleVersionsPassesOneVersion(t *testing.T) {
	root := t.TempDir()
	writeCheckModule(t, root, "one", "modernc.org/sqlite v1.50.1\n")
	writeCheckModule(t, root, "two", "modernc.org/sqlite v1.50.1\n")
	report, err := checkModuleVersions(root, []string{"modernc.org/sqlite"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Violations) != 0 {
		t.Fatalf("report = %#v, want no violations", report)
	}
}

func writeCheckModule(t *testing.T, root, name, requirement string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example/"+name+"\n\ngo 1.22\n\nrequire "+requirement), 0o644); err != nil {
		t.Fatal(err)
	}
}
