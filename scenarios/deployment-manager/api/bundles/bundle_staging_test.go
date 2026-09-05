package bundles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPopulateAssetMetadataAndExpandUIAssets(t *testing.T) {
	root := t.TempDir()
	for _, path := range []string{"assets/readme.txt", "ui/dist/index.html", "ui/dist/assets/app.js"} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(path), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	manifest := &Manifest{Services: []ServiceEntry{
		{Type: "api", Assets: []Asset{{Path: "assets/readme.txt", SHA256: "pending"}}},
		{Type: "ui-bundle"},
	}}
	if err := populateAssetMetadata(manifest, root); err != nil {
		t.Fatalf("populate metadata: %v", err)
	}
	if manifest.Services[0].Assets[0].SHA256 == "pending" || manifest.Services[0].Assets[0].SizeBytes == 0 {
		t.Fatalf("asset metadata not populated: %+v", manifest.Services[0].Assets[0])
	}
	if len(manifest.Services[1].Assets) != 2 {
		t.Fatalf("expected UI assets, got %+v", manifest.Services[1].Assets)
	}
	if err := expandUIAssets(nil, root); err != nil {
		t.Fatal(err)
	}
	if _, _, err := hashFile(filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing file hash should fail")
	}
}

func TestStageBundleArtifactsCopiesFilesDirectoriesAndReportsMissing(t *testing.T) {
	root, out := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "bin", "linux"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "bin", "linux", "app"), []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "assets", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "assets", "nested", "one.txt"), []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := &Manifest{Services: []ServiceEntry{{
		Binaries: map[string]ServiceBinary{"linux": {Path: "bin/linux/app"}},
		Assets:   []Asset{{Path: "assets"}, {Path: "missing.txt"}, {Path: "assets"}},
	}}}
	result, err := stageBundleArtifacts(manifest, root, out)
	if err != nil {
		t.Fatalf("stage: %v", err)
	}
	if len(result.Missing) != 1 || result.Missing[0] != "missing.txt" {
		t.Fatalf("unexpected missing list: %+v", result.Missing)
	}
	for _, path := range []string{"bin/linux/app", "assets/nested/one.txt", "bundle.json"} {
		if _, err := os.Stat(filepath.Join(out, filepath.FromSlash(path))); err != nil {
			t.Fatalf("staged %s: %v", path, err)
		}
	}
	if _, err := stageBundleArtifacts(nil, root, out); err == nil {
		t.Fatal("nil manifest should fail")
	}
	if _, err := stageBundleArtifacts(manifest, root, ""); err == nil {
		t.Fatal("empty output should fail")
	}
	bad := &Manifest{Services: []ServiceEntry{{Binaries: map[string]ServiceBinary{"x": {Path: "../escape"}}}}}
	if _, err := stageBundleArtifacts(bad, root, out); err == nil {
		t.Fatal("escaping path should fail")
	}
}

func TestSanitizeRelPathAndCopyHelpers(t *testing.T) {
	for _, path := range []string{"", ".", "../x", "/absolute"} {
		if _, err := sanitizeRelPath(path); err == nil {
			t.Fatalf("expected invalid path %q", path)
		}
	}
	if got, err := sanitizeRelPath("a/../b"); err != nil || got != "b" {
		t.Fatalf("unexpected normalized path %q: %v", got, err)
	}
	root, dest := t.TempDir(), filepath.Join(t.TempDir(), "copied")
	if err := os.WriteFile(filepath.Join(root, "file"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(root, dest); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dest, "file"))
	if err != nil || string(data) != "data" {
		t.Fatalf("copy failed: %q %v", data, err)
	}
	if err := copyFilePreserveMode(filepath.Join(root, "missing"), filepath.Join(dest, "x")); err == nil {
		t.Fatal("missing copy should fail")
	}
	if resolveScenarioRoot("") != "" {
		t.Fatal("empty scenario should not resolve")
	}
}
