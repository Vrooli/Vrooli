package dependencies

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNodePackageCheckerFailsMissingNodeModules(t *testing.T) {
	dir := t.TempDir()
	ui := filepath.Join(dir, "ui")
	if err := os.MkdirAll(ui, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(ui, "package.json"), `{"name":"demo"}`)
	writeFile(t, filepath.Join(ui, "pnpm-lock.yaml"), "lockfileVersion: '9'\n")

	result := NewNodePackageChecker(dir, DefaultSettings().NodePackages).Check()
	if result.Success {
		t.Fatal("expected missing node_modules failure")
	}
	if !strings.Contains(result.Error.Error(), "missing node_modules") {
		t.Fatalf("unexpected error: %v", result.Error)
	}
}

func TestNodePackageCheckerFailsMultipleLockfiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"demo"}`)
	writeFile(t, filepath.Join(dir, "pnpm-lock.yaml"), "lockfileVersion: '9'\n")
	writeFile(t, filepath.Join(dir, "package-lock.json"), "{}\n")
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}

	result := NewNodePackageChecker(dir, DefaultSettings().NodePackages).Check()
	if result.Success {
		t.Fatal("expected multiple lockfile failure")
	}
	if !strings.Contains(result.Error.Error(), "multiple lockfiles") {
		t.Fatalf("unexpected error: %v", result.Error)
	}
}

func TestNodePackageCheckerPassesReadyWorkspace(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"name":"demo"}`)
	writeFile(t, filepath.Join(dir, "pnpm-lock.yaml"), "lockfileVersion: '9'\n")
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}

	result := NewNodePackageChecker(dir, DefaultSettings().NodePackages).Check()
	if !result.Success {
		t.Fatalf("expected success, got %v", result.Error)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
