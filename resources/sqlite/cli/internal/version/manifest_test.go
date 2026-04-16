package version

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestLoadPrefersInstalledSibling(t *testing.T) {
	dir := t.TempDir()
	installedPath := filepath.Join(dir, "resource-sqlite.manifest.json")
	sourcePath := filepath.Join(dir, "resource.json")

	if err := os.WriteFile(installedPath, []byte("{\"source\":\"installed\"}\n"), 0o644); err != nil {
		t.Fatalf("write installed manifest: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte("{\"source\":\"dev\"}\n"), 0o644); err != nil {
		t.Fatalf("write source manifest: %v", err)
	}

	data, err := (Manifest{InstalledPath: installedPath, SourcePath: sourcePath}).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if string(data) != "{\"source\":\"installed\"}\n" {
		t.Fatalf("manifest data = %q", string(data))
	}
}

func TestManifestLoadFallsBackToExplicitSourceOverride(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "resource.json")
	if err := os.WriteFile(sourcePath, []byte("{\"source\":\"dev\"}\n"), 0o644); err != nil {
		t.Fatalf("write source manifest: %v", err)
	}

	data, err := (Manifest{InstalledPath: filepath.Join(dir, "missing.manifest.json"), SourcePath: sourcePath}).Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if string(data) != "{\"source\":\"dev\"}\n" {
		t.Fatalf("manifest data = %q", string(data))
	}
}
