package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateLogoDerivatives(t *testing.T) {
	tmpDir := t.TempDir()

	// Prepare a source image
	srcImg := image.NewRGBA(image.Rect(0, 0, 800, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 800; x++ {
			srcImg.Set(x, y, color.RGBA{R: 200, G: 20, B: 20, A: 255})
		}
	}

	logoDir := filepath.Join(tmpDir, "logos")
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		t.Fatalf("setup dir: %v", err)
	}

	srcPath := filepath.Join(logoDir, "logo.png")
	out, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := png.Encode(out, srcImg); err != nil {
		t.Fatalf("encode source: %v", err)
	}
	out.Close()

	svc := &AssetsService{uploadDir: tmpDir}
	derivatives, err := svc.generateDerivatives(srcPath, "logos/logo.png", "image/png", "logo")
	if err != nil {
		t.Fatalf("generate derivatives: %v", err)
	}

	expectedKeys := []string{"logo_512", "logo_256", "logo_128", "logo_icon", "favicon_32", "apple_touch_180"}
	for _, key := range expectedKeys {
		path, ok := derivatives[key]
		if !ok {
			t.Fatalf("missing derivative key %s", key)
		}
		full := filepath.Join(tmpDir, path)
		info, err := os.Stat(full)
		if err != nil || info.Size() == 0 {
			t.Fatalf("derivative %s not written: %v", full, err)
		}
		if key == "logo_512" {
			f, _ := os.Open(full)
			img, err := png.Decode(f)
			f.Close()
			if err != nil {
				t.Fatalf("decode derivative: %v", err)
			}
			b := img.Bounds()
			if b.Dx() != 512 || b.Dy() != 512 {
				t.Fatalf("expected 512x512, got %dx%d", b.Dx(), b.Dy())
			}
		}
	}
}

func TestGenerateLogoDerivativesJpeg(t *testing.T) {
	tmpDir := t.TempDir()

	srcImg := image.NewRGBA(image.Rect(0, 0, 600, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 600; x++ {
			srcImg.Set(x, y, color.RGBA{R: 120, G: 80, B: 10, A: 255})
		}
	}

	logoDir := filepath.Join(tmpDir, "logos")
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		t.Fatalf("setup dir: %v", err)
	}

	srcPath := filepath.Join(logoDir, "logo.jpg")
	f, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := jpeg.Encode(f, srcImg, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("encode jpg: %v", err)
	}
	f.Close()

	svc := &AssetsService{uploadDir: tmpDir}
	derivatives, err := svc.generateDerivatives(srcPath, "logos/logo.jpg", "image/jpeg", "logo")
	if err != nil {
		t.Fatalf("generate derivatives: %v", err)
	}
	for _, key := range []string{"logo_256", "favicon_32"} {
		if _, ok := derivatives[key]; !ok {
			t.Fatalf("missing derivative %s for jpeg", key)
		}
	}
	if alias, ok := derivatives["logo_icon"]; !ok || alias == "" {
		t.Fatalf("logo_icon alias missing for jpeg")
	}
}

func TestGenerateDerivativesSvgFallback(t *testing.T) {
	tmpDir := t.TempDir()
	logoDir := filepath.Join(tmpDir, "logos")
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		t.Fatalf("setup dir: %v", err)
	}
	srcPath := filepath.Join(logoDir, "logo.svg")
	if err := os.WriteFile(srcPath, []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100"></svg>`), 0644); err != nil {
		t.Fatalf("write svg: %v", err)
	}

	svc := &AssetsService{uploadDir: tmpDir}
	derivatives, err := svc.generateDerivatives(srcPath, "logos/logo.svg", "image/svg+xml", "logo")
	if err != nil {
		t.Fatalf("svg fallback returned error: %v", err)
	}

	for _, key := range []string{"favicon_32", "apple_touch_180", "logo_icon"} {
		if derivatives[key] != "logos/logo.svg" {
			t.Fatalf("expected fallback to original for %s", key)
		}
	}
}

func TestGenerateFaviconDerivatives(t *testing.T) {
	tmpDir := t.TempDir()
	favDir := filepath.Join(tmpDir, "favicons")
	if err := os.MkdirAll(favDir, 0755); err != nil {
		t.Fatalf("setup dir: %v", err)
	}

	// square source
	srcImg := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			srcImg.Set(x, y, color.RGBA{R: 10, G: 200, B: 10, A: 255})
		}
	}
	srcPath := filepath.Join(favDir, "favicon.png")
	f, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := png.Encode(f, srcImg); err != nil {
		t.Fatalf("encode: %v", err)
	}
	f.Close()

	svc := &AssetsService{uploadDir: tmpDir}
	derivatives, err := svc.generateDerivatives(srcPath, "favicons/favicon.png", "image/png", "favicon")
	if err != nil {
		t.Fatalf("generate derivatives: %v", err)
	}

	for _, key := range []string{"favicon_64", "favicon_32", "favicon_16", "apple_touch_180"} {
		if _, ok := derivatives[key]; !ok {
			t.Fatalf("missing derivative %s", key)
		}
	}
}

func TestGenerateOgDerivatives(t *testing.T) {
	tmpDir := t.TempDir()
	ogDir := filepath.Join(tmpDir, "og-images")
	if err := os.MkdirAll(ogDir, 0755); err != nil {
		t.Fatalf("setup dir: %v", err)
	}

	srcImg := image.NewRGBA(image.Rect(0, 0, 800, 800))
	for y := 0; y < 800; y++ {
		for x := 0; x < 800; x++ {
			srcImg.Set(x, y, color.RGBA{R: 50, G: 50, B: 220, A: 255})
		}
	}
	srcPath := filepath.Join(ogDir, "og.png")
	f, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := png.Encode(f, srcImg); err != nil {
		t.Fatalf("encode: %v", err)
	}
	f.Close()

	svc := &AssetsService{uploadDir: tmpDir}
	derivatives, err := svc.generateDerivatives(srcPath, "og-images/og.png", "image/png", "og_image")
	if err != nil {
		t.Fatalf("generate derivatives: %v", err)
	}

	path, ok := derivatives["og_image_1200x630"]
	if !ok {
		t.Fatalf("missing og_image_1200x630")
	}
	full := filepath.Join(tmpDir, path)
	info, err := os.Stat(full)
	if err != nil || info.Size() == 0 {
		t.Fatalf("og derivative not written: %v", err)
	}
	fp, _ := os.Open(full)
	img, err := png.Decode(fp)
	fp.Close()
	if err != nil {
		t.Fatalf("decode og derivative: %v", err)
	}
	if img.Bounds().Dx() != 1200 || img.Bounds().Dy() != 630 {
		t.Fatalf("og derivative has wrong size %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}
