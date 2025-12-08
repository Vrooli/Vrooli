package main

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
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

func TestGenerateDerivatives_InvalidImageFailsFast(t *testing.T) {
	tmpDir := t.TempDir()
	logoDir := filepath.Join(tmpDir, "logos")
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		t.Fatalf("setup dir: %v", err)
	}

	srcPath := filepath.Join(logoDir, "logo.png")
	if err := os.WriteFile(srcPath, []byte("not a real image"), 0644); err != nil {
		t.Fatalf("write corrupt source: %v", err)
	}

	svc := &AssetsService{uploadDir: tmpDir}
	derivatives, err := svc.generateDerivatives(srcPath, "logos/logo.png", "image/png", "logo")
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
	if derivatives != nil && len(derivatives) > 0 {
		t.Fatalf("expected no derivatives on failure, got %v", derivatives)
	}

	files, err := os.ReadDir(logoDir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected only source file to remain, found %d files", len(files))
	}
}

func TestAssetsServiceUpload_DisallowedMimeRejectsAndCleansUp(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	t.Setenv("UPLOAD_DIR", tmpDir)

	svc := NewAssetsService(db)
	payloadPath := filepath.Join(tmpDir, "payload.txt")
	if err := os.WriteFile(payloadPath, []byte("not an image"), 0644); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	file, err := os.Open(payloadPath)
	if err != nil {
		t.Fatalf("open payload: %v", err)
	}
	defer file.Close()

	req := &AssetUploadRequest{
		File:     file,
		Header:   &multipart.FileHeader{Filename: "payload.txt", Header: textproto.MIMEHeader{"Content-Type": []string{"text/plain"}}, Size: 12},
		Category: "logo",
	}

	asset, err := svc.Upload(req)
	if err == nil {
		t.Fatalf("expected invalid mime type error, got asset %+v", asset)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets WHERE original_filename = 'payload.txt'`).Scan(&count); err != nil {
		t.Fatalf("count assets: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no asset records to be created, found %d", count)
	}

	entries, err := os.ReadDir(filepath.Join(tmpDir, "logos"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("read logos dir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != "" {
			t.Fatalf("expected no uploaded files, found %s", entry.Name())
		}
	}
}

func TestAssetsServiceUpload_RespectsSizeLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	t.Setenv("UPLOAD_DIR", tmpDir)

	svc := NewAssetsService(db)
	svc.maxSize = 16

	payloadPath := filepath.Join(tmpDir, "small.png")
	if err := os.WriteFile(payloadPath, []byte{0, 1, 2, 3}, 0644); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	file, err := os.Open(payloadPath)
	if err != nil {
		t.Fatalf("open payload: %v", err)
	}
	defer file.Close()

	req := &AssetUploadRequest{
		File: file,
		Header: &multipart.FileHeader{
			Filename: "small.png",
			Header:   textproto.MIMEHeader{"Content-Type": []string{"image/png"}},
			Size:     32,
		},
		Category: "favicon",
	}

	if _, err := svc.Upload(req); err == nil {
		t.Fatal("expected size limit enforcement error, got nil")
	}
}

func TestAssetsServiceUpload_PersistsBaseFileWhenDerivativesFail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	uploadDir := t.TempDir()
	t.Setenv("UPLOAD_DIR", uploadDir)

	svc := NewAssetsService(db)

	sourceDir := t.TempDir()
	payloadPath := filepath.Join(sourceDir, "corrupt.png")
	if err := os.WriteFile(payloadPath, []byte("not a real png"), 0644); err != nil {
		t.Fatalf("write corrupt payload: %v", err)
	}
	file, err := os.Open(payloadPath)
	if err != nil {
		t.Fatalf("open corrupt payload: %v", err)
	}
	defer file.Close()

	req := &AssetUploadRequest{
		File: file,
		Header: &multipart.FileHeader{
			Filename: "corrupt.png",
			Header:   textproto.MIMEHeader{"Content-Type": []string{"image/png"}},
			Size:     14,
		},
		Category: "logo",
		AltText:  "Corrupt asset",
	}

	asset, err := svc.Upload(req)
	if err != nil {
		t.Fatalf("upload should tolerate derivative failure: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM assets WHERE id = $1`, asset.ID)
	})

	if asset == nil {
		t.Fatal("expected asset returned even when derivatives fail")
	}
	if asset.Derivatives != nil {
		t.Fatalf("expected derivatives to be empty on rasterization failure, got %v", asset.Derivatives)
	}
	if asset.ThumbnailPath != nil {
		t.Fatalf("expected thumbnail unset when derivatives fail, got %v", asset.ThumbnailPath)
	}

	storedPath := filepath.Join(uploadDir, asset.StoragePath)
	if _, err := os.Stat(storedPath); err != nil {
		t.Fatalf("stored asset missing at %s: %v", storedPath, err)
	}

	entries, err := os.ReadDir(filepath.Dir(storedPath))
	if err != nil {
		t.Fatalf("read stored dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected only base asset stored when derivatives fail, found %d entries", len(entries))
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets WHERE id = $1`, asset.ID).Scan(&count); err != nil {
		t.Fatalf("count assets: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected asset row persisted despite derivative failure, got %d rows", count)
	}
}

func TestAssetsServiceUpload_GeneratesDerivativesAndThumbnail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	t.Setenv("UPLOAD_DIR", tmpDir)

	svc := NewAssetsService(db)

	srcImg := image.NewRGBA(image.Rect(0, 0, 400, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 400; x++ {
			srcImg.Set(x, y, color.RGBA{R: 80, G: 20, B: 200, A: 255})
		}
	}

	logoDir := filepath.Join(tmpDir, "logos")
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		t.Fatalf("mkdir logos: %v", err)
	}
	srcPath := filepath.Join(logoDir, "upload-logo.png")
	out, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("create logo: %v", err)
	}
	if err := png.Encode(out, srcImg); err != nil {
		t.Fatalf("encode logo: %v", err)
	}
	info, _ := out.Stat()
	out.Close()

	file, err := os.Open(srcPath)
	if err != nil {
		t.Fatalf("open logo: %v", err)
	}
	defer file.Close()

	req := &AssetUploadRequest{
		File: file,
		Header: &multipart.FileHeader{
			Filename: "upload-logo.png",
			Header:   textproto.MIMEHeader{"Content-Type": []string{"image/png"}},
			Size:     info.Size(),
		},
		Category: "logo",
		AltText:  "Brand mark",
	}

	asset, err := svc.Upload(req)
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}
	if asset.ThumbnailPath == nil || *asset.ThumbnailPath == "" {
		t.Fatalf("expected thumbnail path to be set from derivatives, got %+v", asset.ThumbnailPath)
	}
	for _, key := range []string{"logo_512", "logo_256", "logo_icon", "favicon_32"} {
		if _, ok := asset.Derivatives[key]; !ok {
			t.Fatalf("expected derivative %s to be present", key)
		}
	}

	fullPath := filepath.Join(tmpDir, asset.StoragePath)
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("expected stored asset at %s: %v", fullPath, err)
	}

	for key, rel := range asset.Derivatives {
		derivativePath := filepath.Join(tmpDir, rel)
		info, err := os.Stat(derivativePath)
		if err != nil {
			t.Fatalf("derivative %s missing at %s: %v", key, derivativePath, err)
		}
		if info.Size() == 0 {
			t.Fatalf("derivative %s is empty at %s", key, derivativePath)
		}
	}
}

func TestAssetsServiceUpload_DetectsMimeAndStoresGeneralAssets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()
	t.Setenv("UPLOAD_DIR", tmpDir)

	srcImg := image.NewRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			srcImg.Set(x, y, color.RGBA{R: 30, G: 120, B: 200, A: 255})
		}
	}

	srcPath := filepath.Join(tmpDir, "general.png")
	f, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("create source: %v", err)
	}
	if err := png.Encode(f, srcImg); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	info, _ := f.Stat()
	f.Close()

	file, err := os.Open(srcPath)
	if err != nil {
		t.Fatalf("open source: %v", err)
	}
	defer file.Close()

	svc := NewAssetsService(db)
	asset, err := svc.Upload(&AssetUploadRequest{
		File: file,
		Header: &multipart.FileHeader{
			Filename: "general.png",
			Header:   textproto.MIMEHeader{}, // force detection based on extension
			Size:     info.Size(),
		},
		AltText: "General graphic",
	})
	if err != nil {
		t.Fatalf("upload failed: %v", err)
	}

	if asset.Category != "general" {
		t.Fatalf("expected default general category, got %s", asset.Category)
	}
	if asset.MimeType != "image/png" {
		t.Fatalf("expected mime type detected from extension, got %s", asset.MimeType)
	}
	if !strings.HasPrefix(asset.StoragePath, "general/") {
		t.Fatalf("expected storage under general/, got %s", asset.StoragePath)
	}
	if asset.AltText == nil || *asset.AltText != "General graphic" {
		t.Fatalf("expected alt text to persist, got %+v", asset.AltText)
	}
	if len(asset.Derivatives) != 0 {
		t.Fatalf("expected no derivatives for general assets, got %v", asset.Derivatives)
	}

	full := filepath.Join(tmpDir, asset.StoragePath)
	if _, err := os.Stat(full); err != nil {
		t.Fatalf("stored asset missing at %s: %v", full, err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM assets WHERE id = $1`, asset.ID).Scan(&count); err != nil {
		t.Fatalf("count assets: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected asset row persisted, got %d rows", count)
	}
}
