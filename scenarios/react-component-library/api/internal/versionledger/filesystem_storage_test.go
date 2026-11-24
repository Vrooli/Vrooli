package versionledger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestUpdatesAreAtomicAndPreservePermissions(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "components", "Button", "component.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"draft":"1.0.1","deprecatedVersions":[]}`), 0o640); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(nil, root)
	if err := repo.clearDraftManifest("components/Button/component.json"); err != nil {
		t.Fatal(err)
	}
	if err := repo.updateManifestArray("components/Button/component.json", "evictedVersions", "1.0.0"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("manifest mode = %04o, want 0640", got)
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got == `{"draft":"1.0.1","deprecatedVersions":[]}` {
		t.Fatal("manifest update was not published")
	}
}

func TestManifestUpdateRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "component.json")
	original := []byte(`{"draft":"1.0.1"}`)
	if err := os.WriteFile(outside, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "component.json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := NewRepository(nil, root).clearDraftManifest("component.json"); err == nil {
		t.Fatal("expected symlink escape to be rejected")
	}
	data, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Fatalf("external target changed: %s", data)
	}
}

func TestReleaseHashLedgerUpdatePreservesPermissions(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "released-version-hashes.json")
	content := []byte(`{"schemaVersion":1,"entries":[{"path":"components/Button/versions/1.0.0/Button.tsx","sha256":"abc"},{"path":"components/Input/versions/1.0.0/Input.tsx","sha256":"def"}]}`)
	if err := os.WriteFile(path, content, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := NewRepository(nil, root).removeReleaseLedgerEntries("components/Button/versions/1.0.0/Button.tsx"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("release ledger mode = %04o, want 0640", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got == string(content) {
		t.Fatal("release ledger entry was not removed")
	}
}

func TestSourceRelativePathRejectsEscape(t *testing.T) {
	for _, path := range []string{"../outside", "/tmp/outside", "components/../../outside"} {
		if _, err := sourceRelativePath(path); err == nil {
			t.Errorf("sourceRelativePath(%q) unexpectedly succeeded", path)
		}
	}
}
