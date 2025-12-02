package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyPathPreservesExecutablesAndDirectories(t *testing.T) {
	root := t.TempDir()

	srcDir := filepath.Join(root, "src", "resources", "playwright")
	nested := filepath.Join(srcDir, "chromium")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	// Executable asset inside the directory tree.
	chromeSrc := filepath.Join(nested, "chrome")
	if err := os.WriteFile(chromeSrc, []byte("fake-chrome"), 0o755); err != nil {
		t.Fatalf("write source chrome: %v", err)
	}

	dstDir := filepath.Join(root, "bundle", "resources", "playwright")
	if err := copyPath(srcDir, dstDir); err != nil {
		t.Fatalf("copyPath: %v", err)
	}

	chromeDst := filepath.Join(dstDir, "chromium", "chrome")
	info, err := os.Stat(chromeDst)
	if err != nil {
		t.Fatalf("stat copied chrome: %v", err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("expected executable bit preserved, got mode %v", info.Mode())
	}

	// Direct file copy still works and preserves mode.
	fileDst := filepath.Join(root, "bundle", "bin", "shim")
	if err := copyPath(chromeSrc, fileDst); err != nil {
		t.Fatalf("copyPath file: %v", err)
	}
	fileInfo, err := os.Stat(fileDst)
	if err != nil {
		t.Fatalf("stat copied file: %v", err)
	}
	if fileInfo.Mode()&0o111 == 0 {
		t.Fatalf("expected executable bit preserved on file copy, got mode %v", fileInfo.Mode())
	}
}

func TestEnsureBundleExtraResourcesAddsEntry(t *testing.T) {
	appDir := t.TempDir()
	pkgPath := filepath.Join(appDir, "package.json")

	original := map[string]any{
		"build": map[string]any{
			"extraResources": []any{
				map[string]any{"from": "app", "to": "app"},
			},
		},
	}
	data, _ := json.Marshal(original)
	if err := os.WriteFile(pkgPath, data, 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}

	if err := ensureBundleExtraResources(appDir); err != nil {
		t.Fatalf("ensureBundleExtraResources: %v", err)
	}

	updatedRaw, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatalf("read updated package.json: %v", err)
	}

	var updated map[string]any
	if err := json.Unmarshal(updatedRaw, &updated); err != nil {
		t.Fatalf("parse updated package.json: %v", err)
	}

	build, _ := updated["build"].(map[string]any)
	extras, _ := build["extraResources"].([]any)

	foundBundle := false
	for _, entry := range extras {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		if m["from"] == "bundle" || m["to"] == "bundle" {
			foundBundle = true
			break
		}
	}
	if !foundBundle {
		t.Fatalf("bundle extraResources entry not added: %+v", extras)
	}
}
