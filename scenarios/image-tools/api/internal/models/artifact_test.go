package models

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
)

// TestInstall_RejectsHTMLPageDownload is the end-to-end regression for the
// install-stub bug: a model whose source resolves to an HTML page must FAIL the
// install (not record a fake "installed" model). It exercises the real Install
// path with a downloader that returns a page. (Artifact-validation unit coverage
// lives in internal/fetch; this asserts the Installer wires it correctly.)
func TestInstall_RejectsHTMLPageDownload(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	f.payload = append([]byte("<!DOCTYPE html><html><head></head><body>rembg releases</body></html>"), bytes.Repeat([]byte(" "), 4096)...)

	_, err := f.in.Install(ctx, installTestModelID, nil)
	if !errors.Is(err, binaryfetch.ErrLooksLikeHTML) {
		t.Fatalf("an HTML-page download must be rejected, got %v", err)
	}
	if f.in.Installed(ctx, installTestModelID) {
		t.Fatalf("model must NOT be recorded as installed after a rejected page download")
	}
	if dirExists(filepath.Join(f.root, "models", installTestModelID)) {
		t.Fatalf("partial download dir should be removed after rejection")
	}
}

// TestInstall_MultiAssetValidatesEach proves a multi-asset install downloads,
// validates, and lays out every asset under its declared filename.
func TestInstall_MultiAssetValidatesEach(t *testing.T) {
	ctx := context.Background()
	f := newInstallFixture(t)
	// A custom model declaring two generic assets.
	custom := Model{
		ID:         "pair-model",
		Name:       "Pair Model",
		Operations: []string{"upscale"},
		Tier:       TierNiceToHave,
		Backend:    "realesrgan-ncnn-vulkan",
		Hardware:   Hardware{CPUCapable: true},
		Source: Source{Assets: []Asset{
			{URL: "https://example.test/a.param", Filename: "a.param", Kind: ArtifactNCNNParam},
			{URL: "https://example.test/a.bin", Filename: "a.bin", Kind: ArtifactNCNNBin},
		}},
	}
	if err := f.in.AddCustom(ctx, custom); err != nil {
		t.Fatalf("add custom: %v", err)
	}
	if _, err := f.in.Install(ctx, "pair-model", nil); err != nil {
		t.Fatalf("install pair-model: %v", err)
	}
	for _, name := range []string{"a.param", "a.bin"} {
		if !pathExists(filepath.Join(f.root, "models", "pair-model", name)) {
			t.Fatalf("expected asset %s laid out on disk", name)
		}
	}
	if f.downloads != 2 {
		t.Fatalf("expected 2 asset downloads, got %d", f.downloads)
	}
}
