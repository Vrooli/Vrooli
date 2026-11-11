package models

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectInventoryPathCountsHardLinksOnceAcrossRoots(t *testing.T) {
	root := t.TempDir()
	modelRoot := filepath.Join(root, "models", "m1")
	adapterRoot := filepath.Join(root, "adapters", "a1")
	if err := os.MkdirAll(modelRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(adapterRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	modelFile := filepath.Join(modelRoot, "weights.bin")
	if err := os.WriteFile(modelFile, []byte("shared-weights"), 0o600); err != nil {
		t.Fatal(err)
	}
	adapterFile := filepath.Join(adapterRoot, "weights.bin")
	if err := os.Link(modelFile, adapterFile); err != nil {
		t.Skipf("hard links unavailable on this test filesystem: %v", err)
	}
	seen := map[string]inodeInfo{}
	model := inventoryArtifact{ID: "m1", Path: modelRoot, Installed: true}
	if err := inspectInventoryPath(&model, seen); err != nil {
		t.Fatal(err)
	}
	adapter := inventoryArtifact{ID: "a1", Path: adapterRoot, Installed: true}
	if err := inspectInventoryPath(&adapter, seen); err != nil {
		t.Fatal(err)
	}
	if model.ApparentBytes != uint64(len("shared-weights")) || adapter.ApparentBytes != model.ApparentBytes {
		t.Fatalf("apparent bytes model=%d adapter=%d", model.ApparentBytes, adapter.ApparentBytes)
	}
	if adapter.SharedInodeRefs != 1 || len(seen) != 1 {
		t.Fatalf("shared inode accounting refs=%d unique=%d", adapter.SharedInodeRefs, len(seen))
	}
}

func TestInspectInventoryPathFollowsExternalFileSymlinkAndDeduplicatesCacheTarget(t *testing.T) {
	root := t.TempDir()
	modelRoot := filepath.Join(root, "models", "m1")
	cacheRoot := filepath.Join(root, "shared-cache")
	if err := os.MkdirAll(modelRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	cacheFile := filepath.Join(cacheRoot, "weights.bin")
	if err := os.WriteFile(cacheFile, []byte("shared-hugging-face-weight"), 0o600); err != nil {
		t.Fatal(err)
	}
	modelLink := filepath.Join(modelRoot, "weights.bin")
	if err := os.Symlink(cacheFile, modelLink); err != nil {
		t.Skipf("file symlinks unavailable on this test filesystem: %v", err)
	}

	seen := map[string]inodeInfo{}
	model := inventoryArtifact{ID: "m1", Path: modelRoot, Installed: true}
	if err := inspectInventoryPath(&model, seen); err != nil {
		t.Fatal(err)
	}
	if got, want := model.ApparentBytes, uint64(len("shared-hugging-face-weight")); got != want {
		t.Fatalf("model apparent bytes = %d, want %d", got, want)
	}
	if model.SymlinkFiles != 1 || model.ExternalLinkBytes != model.ApparentBytes || len(model.ExternalLinkPaths) != 1 || model.ExternalLinkPaths[0] != cacheFile {
		t.Fatalf("external link accounting = %+v", model)
	}

	cache := inventoryArtifact{ID: "cache", Path: cacheRoot}
	if err := inspectInventoryPath(&cache, seen); err != nil {
		t.Fatal(err)
	}
	if cache.SharedInodeRefs != 1 || len(seen) != 1 || cache.ApparentBytes != model.ApparentBytes {
		t.Fatalf("shared cache was not inode-deduplicated: model=%+v cache=%+v unique=%d", model, cache, len(seen))
	}
}

func TestInspectInventoryPathDoesNotTraverseDirectorySymlinks(t *testing.T) {
	root := t.TempDir()
	modelRoot := filepath.Join(root, "models", "m1")
	external := filepath.Join(root, "external")
	if err := os.MkdirAll(modelRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(external, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(external, "outside.bin"), []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(modelRoot, "linked-directory")); err != nil {
		t.Skipf("directory symlinks unavailable on this test filesystem: %v", err)
	}
	item := inventoryArtifact{ID: "m1", Path: modelRoot}
	if err := inspectInventoryPath(&item, map[string]inodeInfo{}); err != nil {
		t.Fatal(err)
	}
	if item.ApparentBytes != 0 || item.SymlinkFiles != 0 || item.DirectorySymlinks != 1 || item.AllocatedKnown {
		t.Fatalf("directory symlink was followed: %+v", item)
	}
}

func TestInspectInventoryPathMarksBrokenFileSymlinkAccountingUnknown(t *testing.T) {
	root := t.TempDir()
	modelRoot := filepath.Join(root, "models", "m1")
	if err := os.MkdirAll(modelRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "missing.bin"), filepath.Join(modelRoot, "missing.bin")); err != nil {
		t.Skipf("file symlinks unavailable on this test filesystem: %v", err)
	}
	item := inventoryArtifact{ID: "m1", Path: modelRoot}
	if err := inspectInventoryPath(&item, map[string]inodeInfo{}); err != nil {
		t.Fatal(err)
	}
	if item.AllocatedKnown || item.Finding == "" {
		t.Fatalf("broken target did not make inventory incomplete: %+v", item)
	}
}
