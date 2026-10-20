package presentations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// This opt-in visual proof reads the pinned raw source captures and writes
// only derived rasters and a receipt under the Device Studio evidence tree.
func TestRawCapturePresentationEvidence(t *testing.T) {
	if os.Getenv("DEVICE_STUDIO_PRESENTATION_EVIDENCE") != "1" {
		t.Skip("set DEVICE_STUDIO_PRESENTATION_EVIDENCE=1 to refresh local visual proof")
	}
	evidenceDir := filepath.Clean("../../../docs/internal/goal/evidence/E2/presentations")
	if err := os.MkdirAll(evidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	phone, phoneHash := readPinnedPNG(t,
		"/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v4/phone/source-frame84.png",
		"bd6bd9f303fe60771928c6fc7d1e2a6bbb776a0f7f45fee2926a2e098f0c697d", image.Pt(430, 880))
	desktop, desktopHash := readPinnedPNG(t,
		"/home/matthalloran8/.vrooli/evidence/device-presentation-2026-10-18/v3/imported-laptop/source/app-capture.png",
		"70ecddd6cfd3db8e32438a09f1c58d249fe83735bf28b99b7aa6e187461e389d", image.Pt(1440, 800))

	phoneConfig := Config{Target: image.Pt(1206, 2622), Phone: &PhoneConfig{
		ShowStatus: true, ShowHome: true, Clock: "9:41", BatteryPercent: 84,
		ShowBatteryPercent: true, Cellular: true, WiFi: true, Appearance: AppearanceDark,
	}}
	phoneResult, err := Compose(context.Background(), phone, phoneConfig)
	if err != nil {
		t.Fatal(err)
	}
	phoneLightConfig := phoneConfig
	phoneLightConfig.Phone = &PhoneConfig{
		ShowStatus: true, ShowHome: true, Clock: "9:41", Carrier: "Vrooli", BatteryPercent: 38,
		ShowBatteryPercent: true, Cellular: true, WiFi: true, Appearance: AppearanceLight,
	}
	phoneLightResult, err := Compose(context.Background(), phone, phoneLightConfig)
	if err != nil {
		t.Fatal(err)
	}
	browserConfig := Config{Target: image.Pt(2560, 1440), Browser: &BrowserConfig{
		Window: BasisRect{X: 650, Y: 500, Width: 8700, Height: 8500},
		Title:  "Meridian — product preview", Address: "studio.local/projects/meridian",
		Appearance: AppearanceDark, ShowToolbar: true, ShowWindowControls: true,
		ShowMenuBar: true, ShowDock: true,
	}}
	desktopResult, err := Compose(context.Background(), desktop, browserConfig)
	if err != nil {
		t.Fatal(err)
	}
	phoneOut, phoneOutputHash := encodeEvidencePNG(t, evidenceDir, "phone-dark-1206x2622.png", phoneResult.Raster)
	phoneLightOut, phoneLightOutputHash := encodeEvidencePNG(t, evidenceDir, "phone-light-1206x2622.png", phoneLightResult.Raster)
	desktopOut, desktopOutputHash := encodeEvidencePNG(t, evidenceDir, "desktop-2560x1440.png", desktopResult.Raster)
	_ = phoneOut
	_ = phoneLightOut
	_ = desktopOut
	receipt := map[string]any{
		"schema_version": 1,
		"provenance":     "illustrative pure-package raster proof from immutable raw app captures; no photo projection or product route",
		"font_sha256":    fontDigest(),
		"phone": map[string]any{
			"source_role": "raw phone app capture (app content only)", "source_sha256": phoneHash,
			"output": "phone-dark-1206x2622.png", "output_sha256": phoneOutputHash,
			"profile_id": phoneResult.Layout.ProfileID, "profile_version": phoneResult.Layout.ProfileVersion,
			"rasterizer": phoneResult.Layout.Rasterizer, "config_sha256": phoneResult.Layout.ConfigSHA256,
			"display_size": phoneResult.Layout.DisplaySize, "app_viewport": phoneResult.Layout.AppViewport,
		},
		"phone_light": map[string]any{
			"source_role": "raw phone app capture (app content only)", "source_sha256": phoneHash,
			"output": "phone-light-1206x2622.png", "output_sha256": phoneLightOutputHash,
			"profile_id": phoneLightResult.Layout.ProfileID, "profile_version": phoneLightResult.Layout.ProfileVersion,
			"rasterizer": phoneLightResult.Layout.Rasterizer, "config_sha256": phoneLightResult.Layout.ConfigSHA256,
			"display_size": phoneLightResult.Layout.DisplaySize, "app_viewport": phoneLightResult.Layout.AppViewport,
		},
		"desktop": map[string]any{
			"source_role": "raw laptop app capture (1440x800); not the derived bare-app reference", "source_sha256": desktopHash,
			"output": "desktop-2560x1440.png", "output_sha256": desktopOutputHash,
			"profile_id": desktopResult.Layout.ProfileID, "profile_version": desktopResult.Layout.ProfileVersion,
			"rasterizer": desktopResult.Layout.Rasterizer, "config_sha256": desktopResult.Layout.ConfigSHA256,
			"display_size": desktopResult.Layout.DisplaySize, "app_viewport": desktopResult.Layout.AppViewport,
		},
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(evidenceDir, "raw-capture-replay.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readPinnedPNG(t *testing.T, path, expectedHash string, expectedSize image.Point) (image.Image, string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != expectedHash {
		t.Fatalf("fixture hash drift for %s: got %s", filepath.Base(path), got)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Size() != expectedSize {
		t.Fatalf("fixture dimensions changed: got %v want %v", img.Bounds().Size(), expectedSize)
	}
	return img, got
}

func encodeEvidencePNG(t *testing.T, dir, name string, raster *image.NRGBA) (string, string) {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, raster); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return path, hex.EncodeToString(sum[:])
}
