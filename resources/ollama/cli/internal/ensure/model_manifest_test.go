package ensure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstalledManifestDigestReadsExactManagedManifest(t *testing.T) {
	root := filepath.Join(t.TempDir(), "models")
	manifestPath := filepath.Join(root, "manifests", RegistryHost, "library", "model", "8b")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"schemaVersion":2,"layers":[]}`)
	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	digest, err := InstalledManifestDigest(root, "model:8b")
	if err != nil {
		t.Fatal(err)
	}
	if want := "sha256:" + sha256Hex(data); digest != want {
		t.Fatalf("manifest digest = %q, want %q", digest, want)
	}
}

func TestInstalledManifestDigestRejectsMissingAndLinkedManifestPaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "models")
	manifestDir := filepath.Join(root, "manifests", RegistryHost, "library", "model")
	if err := os.MkdirAll(manifestDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := InstalledManifestDigest(root, "model:missing"); err == nil {
		t.Fatal("missing model manifest was accepted")
	}
	outside := filepath.Join(t.TempDir(), "manifest")
	if err := os.WriteFile(outside, []byte(`{"schemaVersion":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(manifestDir, "linked")); err != nil {
		t.Skipf("symlink fixture is unavailable: %v", err)
	}
	if _, err := InstalledManifestDigest(root, "model:linked"); err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("linked manifest error = %v, want non-regular-file refusal", err)
	}
}
