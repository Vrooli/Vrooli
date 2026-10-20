package composition

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
	"strings"
	"testing"
)

// TestV6PhonePlateFixture is an opt-in replay of the read-only v6 phone sample.
// It writes only to the explicitly supplied renderer-core evidence directory.
func TestV6PhonePlateFixture(t *testing.T) {
	fixtureDir := os.Getenv("DEVICE_STUDIO_V6_PHONE_DIR")
	outputDir := os.Getenv("DEVICE_STUDIO_RENDERER_CORE_DIR")
	if fixtureDir == "" || outputDir == "" {
		t.Skip("set DEVICE_STUDIO_V6_PHONE_DIR and DEVICE_STUDIO_RENDERER_CORE_DIR for the retained v6 replay")
	}
	cleanOutput := filepath.Clean(outputDir)
	if filepath.Base(cleanOutput) != "renderer-core" || !strings.Contains(filepath.ToSlash(cleanOutput), "/goal/evidence/renderer-core") {
		t.Fatalf("output must be the authorized goal/evidence/renderer-core directory, got %q", outputDir)
	}
	read := func(name string) []byte {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(fixtureDir, name))
		if err != nil {
			t.Fatalf("read fixture %s: %v", name, err)
		}
		return b
	}
	decode := func(name string) image.Image {
		t.Helper()
		b := read(name)
		img, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatalf("decode fixture %s: %v", name, err)
		}
		return img
	}
	var calibration struct {
		PlateSHA     string      `json:"plate_sha256"`
		PlateSize    []int       `json:"plate_size"`
		SourceCanvas []int       `json:"source_canvas"`
		Homography   [][]float64 `json:"homography_source_to_plate"`
	}
	if err := json.Unmarshal(read("calibration.json"), &calibration); err != nil {
		t.Fatal(err)
	}
	plateBytes := read("plate.png")
	plateHash := sha256.Sum256(plateBytes)
	if hex.EncodeToString(plateHash[:]) != calibration.PlateSHA {
		t.Fatal("fixture plate digest does not match its calibration")
	}
	if len(calibration.PlateSize) != 2 || len(calibration.SourceCanvas) != 2 || len(calibration.Homography) != 3 || len(calibration.Homography[0]) != 3 {
		t.Fatal("fixture calibration is incomplete")
	}
	plate, err := png.Decode(bytes.NewReader(plateBytes))
	if err != nil {
		t.Fatal(err)
	}
	screen := decode("source-screen.png")
	active := decode("active-mask.png")
	if plate.Bounds().Dx() != calibration.PlateSize[0] || plate.Bounds().Dy() != calibration.PlateSize[1] {
		t.Fatal("fixture plate dimensions mismatch calibration")
	}
	if screen.Bounds().Dx() != calibration.SourceCanvas[0] || screen.Bounds().Dy() != calibration.SourceCanvas[1] {
		t.Fatal("fixture screen dimensions mismatch calibration")
	}
	h := matrix3{}
	for y := range h {
		for x := range h[y] {
			h[y][x] = calibration.Homography[y][x]
		}
	}
	quad := [4]Point{}
	boundary := [4]Point{{0, 0}, {float64(calibration.SourceCanvas[0]), 0}, {float64(calibration.SourceCanvas[0]), float64(calibration.SourceCanvas[1])}, {0, float64(calibration.SourceCanvas[1])}}
	for i, p := range boundary {
		x, y, ok := project(h, p.X-0.5, p.Y-0.5)
		if !ok {
			t.Fatalf("convert calibrated corner %d", i)
		}
		quad[i] = Point{x + 0.5, y + 0.5}
	}
	result, err := Compose(context.Background(), Inputs{Plate: plate, Screen: screen, SurfaceSize: image.Pt(calibration.SourceCanvas[0], calibration.SourceCanvas[1]), Quad: quad, ActiveMask: active, Samples: 2})
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.ProtectedChangedPixelCount != 0 {
		t.Fatalf("protected pixels changed: %+v", result.Receipt)
	}
	if err := os.MkdirAll(cleanOutput, 0o755); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(cleanOutput, "v6-phone-replay.png")
	out, err := os.Create(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(out, result.Image); err != nil {
		_ = out.Close()
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	outHash := sha256.Sum256(encoded)
	receipt := struct {
		Fixture      string  `json:"fixture"`
		PlateSHA256  string  `json:"plate_sha256"`
		ScreenSHA256 string  `json:"screen_sha256"`
		MaskSHA256   string  `json:"active_mask_sha256"`
		OutputSHA256 string  `json:"output_sha256"`
		OutputFile   string  `json:"output_file"`
		Compose      Receipt `json:"compose"`
	}{
		Fixture:      "v6 phone fixed-plate sample; read-only originals",
		PlateSHA256:  calibration.PlateSHA,
		ScreenSHA256: digest(read("source-screen.png")),
		MaskSHA256:   digest(read("active-mask.png")),
		OutputSHA256: hex.EncodeToString(outHash[:]),
		OutputFile:   filepath.Base(outPath),
		Compose:      result.Receipt,
	}
	receiptBytes, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cleanOutput, "v6-phone-replay.json"), append(receiptBytes, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("v6 replay PNG: %s (sha256 %s)", outPath, receipt.OutputSHA256)
}

func digest(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
