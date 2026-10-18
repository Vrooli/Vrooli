package buildinfo

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/vrooli/repo-contract-go/repocontracttest"
)

func TestComputeFingerprint_DetectsChanges(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "cli")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	writeFile(t, root, "main.go", []byte("package main\n"))
	writeFile(t, root, "format.go", []byte("package main\n"))

	original, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint: %v", err)
	}

	writeFile(t, root, "format.go", []byte("package main\n// updated\n"))
	updated, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint after update: %v", err)
	}

	if original == updated {
		t.Fatalf("expected fingerprint to change after modifying files")
	}

	// Ensure deterministic ordering by rerunning without modifications
	secondRun, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint second run: %v", err)
	}
	if updated != secondRun {
		t.Fatalf("expected fingerprint to be deterministic; got %s and %s", updated, secondRun)
	}
}

func TestComputeFingerprint_UsesScopedCoverageOutputRoots(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name string, content []byte) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		writeFile(t, root, name, content)
	}
	write(".vrooli/repo-contract.json", []byte("{}\n"))
	write(".gitignore", []byte("/coverage/\npackages/*/coverage/\nscenarios/*/ui/coverage/\ntemplates/scenarios/*/ui/coverage/\n"))
	write("api/internal/checks/coverage/check.go", []byte("package coverage\nconst Value = 1\n"))
	write("api/internal/ui/coverage/check.go", []byte("package coverage\nconst UIValue = 1\n"))
	write("coverage/index.html", []byte("root report"))
	write("scenarios/demo/ui/coverage/index.html", []byte("scenario UI report"))
	write("scenarios/demo/coverage/authored.go", []byte("package coverage\n"))

	first, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("ComputeFingerprint: %v", err)
	}
	write("coverage/index.html", []byte("updated root report"))
	write("scenarios/demo/ui/coverage/index.html", []byte("updated scenario UI report"))
	outputsChanged, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("ComputeFingerprint after output churn: %v", err)
	}
	if outputsChanged != first {
		t.Fatal("declared coverage report churn changed the source fingerprint")
	}
	write("api/internal/checks/coverage/check.go", []byte("package coverage\nconst Value = 2\n"))
	sourceChanged, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("ComputeFingerprint after source edit: %v", err)
	}
	if sourceChanged == first {
		t.Fatal("nested authored Go source did not change the fingerprint")
	}
}

func TestComputeFingerprint_IgnoresModtimeAndSkippedDirs(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "cli")
	if err := os.MkdirAll(filepath.Join(root, "dist"), 0o755); err != nil {
		t.Fatalf("setup dist: %v", err)
	}

	writeFile(t, root, "main.go", []byte("package main\n"))

	original, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint: %v", err)
	}

	// Change modtime only
	if err := os.Chtimes(filepath.Join(root, "main.go"), time.Now().Add(time.Hour), time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("touch file: %v", err)
	}

	// Add file in skipped directory
	writeFile(t, root, "dist/bundle.js", []byte("console.log('skip')"))

	updated, err := ComputeFingerprint(root)
	if err != nil {
		t.Fatalf("compute fingerprint updated: %v", err)
	}

	if original != updated {
		t.Fatalf("expected fingerprint unchanged when only modtime or skipped dirs differ")
	}
}

func TestComputeFingerprint_ErrorsOnUnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		repocontracttest.SkipPlatform(t, "file permission semantics differ on Windows")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "cli")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	target := filepath.Join(root, "secret.txt")
	if err := os.WriteFile(target, []byte("data"), 0o200); err != nil { // write-only
		t.Fatalf("write: %v", err)
	}
	if _, err := ComputeFingerprint(root); err == nil {
		t.Fatalf("expected error for unreadable file")
	}
}

func TestSkipDirNormalizesSeparators(t *testing.T) {
	if !skipDir(t.TempDir(), "node_modules\\pkg") {
		t.Fatalf("expected windows-style node_modules path to be skipped")
	}
	if !skipFile("custom\\cache\\index.json", []string{"custom/cache"}) {
		t.Fatalf("expected windows-style custom/cache path to be skipped via extra list")
	}
}

func TestIsCoverageOutputDirRequiresDeclaredRepositoryPath(t *testing.T) {
	root := t.TempDir()
	if IsCoverageOutputDir(root, "coverage") {
		t.Fatal("an arbitrary input root must not turn its coverage package into an output")
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("/coverage/\npackages/*/coverage/\nscenarios/*/ui/coverage/\ntemplates/scenarios/*/ui/coverage/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"coverage", "packages/demo/coverage", "scenarios/demo/ui/coverage", "templates/scenarios/demo/ui/coverage"} {
		if !IsCoverageOutputDir(root, path) {
			t.Errorf("expected repo-declared coverage output %q", path)
		}
	}
	for _, path := range []string{"scenarios/demo/coverage", "api/internal/ui/coverage", "scenarios/demo/api/internal/ui/coverage"} {
		if IsCoverageOutputDir(root, path) {
			t.Errorf("unlisted path %q was treated as a coverage output", path)
		}
	}
}

func writeFile(t *testing.T, root, name string, contents []byte) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
