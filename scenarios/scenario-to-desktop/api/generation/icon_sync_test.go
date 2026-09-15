package generation

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeTestPNG(t *testing.T, path string, w, h int, c color.NRGBA) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIconSyncWritesExactSizesAndKeepsCorrectAssets(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	// brand-manager-applied electron source.
	srcDir := filepath.Join(root, "scenarios", scenario, "platforms", "electron", "assets")
	writeTestPNG(t, filepath.Join(srcDir, "icon.png"), 512, 512, color.NRGBA{20, 40, 80, 255})

	outputDir := t.TempDir()
	syncer := NewIconSyncer(root)
	if err := syncer.SyncIcons(scenario, outputDir, nil); err != nil {
		t.Fatalf("sync: %v", err)
	}

	assetsDir := filepath.Join(outputDir, "assets")
	for _, target := range iconTargets {
		if !correctSize(filepath.Join(assetsDir, target.Name), target.Size) {
			t.Fatalf("%s is not %dx%d", target.Name, target.Size, target.Size)
		}
	}
	// The old bug wrote one image under every name; a correct set differs by size.
	p512, _ := os.ReadFile(filepath.Join(assetsDir, "icon-512x512.png"))
	p16, _ := os.ReadFile(filepath.Join(assetsDir, "icon-16x16.png"))
	if bytes.Equal(p512, p16) {
		t.Fatal("512 and 16 icons must not be byte-identical")
	}

	// A second sync over correct assets writes nothing (idempotent).
	before, _ := os.ReadFile(filepath.Join(assetsDir, "icon-32x32.png"))
	if err := syncer.SyncIcons(scenario, outputDir, nil); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(filepath.Join(assetsDir, "icon-32x32.png"))
	if !bytes.Equal(before, after) {
		t.Fatal("a correct asset was overwritten on re-sync")
	}
}

func TestIconSyncReplacesPlaceholder(t *testing.T) {
	root := t.TempDir()
	scenario := "demo"
	writeTestPNG(t, filepath.Join(root, "scenarios", scenario, "ui", "public", "public", "icon-512.png"), 512, 512, color.NRGBA{20, 40, 80, 255})

	outputDir := t.TempDir()
	assetsDir := filepath.Join(outputDir, "assets")
	// Seed a 663-byte desktop placeholder under the 16x16 name.
	placeholder := bytes.Repeat([]byte{0x80}, 663)
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "icon-16x16.png"), placeholder, 0o644); err != nil {
		t.Fatal(err)
	}

	syncer := NewIconSyncer(root)
	if err := syncer.SyncIcons(scenario, outputDir, nil); err != nil {
		t.Fatal(err)
	}
	if !correctSize(filepath.Join(assetsDir, "icon-16x16.png"), 16) {
		t.Fatal("placeholder 16x16 was not replaced with an exact 16x16 icon")
	}
	if isPlaceholder(filepath.Join(assetsDir, "icon-16x16.png")) {
		t.Fatal("placeholder bytes survived the sync")
	}
}
