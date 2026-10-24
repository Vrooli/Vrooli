package presentations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/draw"
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
	lightPhoneFill := RGB{R: 246, G: 247, B: 250}
	phoneLightConfig := phoneConfig
	phoneLightConfig.Phone = &PhoneConfig{
		ShowStatus: true, ShowHome: true, Clock: "9:41", Carrier: "Vrooli", BatteryPercent: 38,
		ShowBatteryPercent: true, Cellular: true, WiFi: true, Appearance: AppearanceLight,
		TopFill: &lightPhoneFill, BottomFill: &lightPhoneFill,
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
	assertPinnedV2Golden(t, evidenceDir, phoneHash, "phone", phoneResult, phoneConfig)
	assertPinnedV2Golden(t, evidenceDir, phoneHash, "phone_light", phoneLightResult, phoneLightConfig)
	assertPinnedV2Golden(t, evidenceDir, desktopHash, "desktop", desktopResult, browserConfig)
	phoneV3Config := Config{Target: image.Pt(1206, 2622), Phone: &PhoneConfig{
		ProfileVersion: PhoneProfileVersionAuto, ShowStatus: true, ShowHome: true,
		Clock: "9:41", BatteryPercent: 84, ShowBatteryPercent: true,
		Cellular: true, WiFi: true, Appearance: AppearanceAuto,
	}}
	phoneV3Result, err := Compose(context.Background(), phone, phoneV3Config)
	if err != nil {
		t.Fatal(err)
	}
	_, phoneV3OutputHash := encodeEvidencePNG(t, evidenceDir, "phone-v3-auto-1206x2622.png", phoneV3Result.Raster)
	receipt := map[string]any{
		"schema_version": 1,
		"provenance":     "illustrative phone presentation v3 raster over an immutable raw app capture; this is not photo projection or a product route",
		"source": map[string]string{
			"role": "raw phone app capture (app content only)", "sha256": phoneHash,
		},
		"font_sha256": fontDigest(),
		"profile": map[string]string{
			"semantic_version": "device-studio-presentation-v3",
			"profile_id":       PhoneProfileID, "profile_version": PhoneProfileVersionAuto,
			"rasterizer": RasterizerID, "fill_policy": FillPolicyID,
		},
		"output": map[string]any{
			"name": "phone-v3-auto-1206x2622.png", "sha256": phoneV3OutputHash,
			"config": phoneV3Config, "layout": phoneV3Result.Layout,
		},
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(evidenceDir, "phone-v3-auto-replay.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertPinnedV2Golden(t *testing.T, dir, sourceSHA, key string, result Result, config Config) {
	t.Helper()
	var receipt struct {
		Phone struct {
			Config Config `json:"config"`
			Layout Layout `json:"layout"`
			Output string `json:"output"`
			SHA    string `json:"output_sha256"`
			Source string `json:"source_sha256"`
		} `json:"phone"`
		PhoneLight struct {
			Config Config `json:"config"`
			Layout Layout `json:"layout"`
			Output string `json:"output"`
			SHA    string `json:"output_sha256"`
			Source string `json:"source_sha256"`
		} `json:"phone_light"`
		Desktop struct {
			Config Config `json:"config"`
			Layout Layout `json:"layout"`
			Output string `json:"output"`
			SHA    string `json:"output_sha256"`
			Source string `json:"source_sha256"`
		} `json:"desktop"`
	}
	data, err := os.ReadFile(filepath.Join(dir, "raw-capture-replay.json"))
	if err != nil {
		t.Fatalf("read retained v2 receipt: %v", err)
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatalf("decode retained v2 receipt: %v", err)
	}
	var oldConfig Config
	var oldLayout Layout
	var outputName, outputSHA, oldSource string
	switch key {
	case "phone":
		oldConfig, oldLayout, outputName, outputSHA, oldSource = receipt.Phone.Config, receipt.Phone.Layout, receipt.Phone.Output, receipt.Phone.SHA, receipt.Phone.Source
	case "phone_light":
		oldConfig, oldLayout, outputName, outputSHA, oldSource = receipt.PhoneLight.Config, receipt.PhoneLight.Layout, receipt.PhoneLight.Output, receipt.PhoneLight.SHA, receipt.PhoneLight.Source
	case "desktop":
		oldConfig, oldLayout, outputName, outputSHA, oldSource = receipt.Desktop.Config, receipt.Desktop.Layout, receipt.Desktop.Output, receipt.Desktop.SHA, receipt.Desktop.Source
	default:
		t.Fatalf("unknown v2 golden key %q", key)
	}
	if oldSource != sourceSHA || outputName == "" || oldLayout.ProfileVersion != PhoneProfileVersion && key != "desktop" {
		t.Fatalf("retained %s v2 receipt identity changed: source=%s layout=%s output=%s", key, oldSource, oldLayout.ProfileVersion, outputName)
	}
	oldConfigBytes, _ := json.Marshal(oldConfig)
	configBytes, _ := json.Marshal(config)
	oldLayoutBytes, _ := json.Marshal(oldLayout)
	layoutBytes, _ := json.Marshal(result.Layout)
	if !bytes.Equal(oldConfigBytes, configBytes) || !bytes.Equal(oldLayoutBytes, layoutBytes) {
		t.Fatalf("retained %s v2 config/layout changed", key)
	}
	outputBytes, err := os.ReadFile(filepath.Join(dir, outputName))
	if err != nil {
		t.Fatalf("read retained %s v2 output: %v", key, err)
	}
	sum := sha256.Sum256(outputBytes)
	if hex.EncodeToString(sum[:]) != outputSHA {
		t.Fatalf("retained %s v2 output hash differs from receipt", key)
	}
	oldRaster, err := png.Decode(bytes.NewReader(outputBytes))
	if err != nil {
		t.Fatalf("decode retained %s v2 output: %v", key, err)
	}
	oldNRGBA := image.NewNRGBA(oldRaster.Bounds())
	draw.Draw(oldNRGBA, oldNRGBA.Bounds(), oldRaster, oldRaster.Bounds().Min, draw.Src)
	if !equalRaster(oldNRGBA, result.Raster) {
		t.Fatalf("retained %s v2 raster pixels changed", key)
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
