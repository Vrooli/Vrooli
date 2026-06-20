package ai

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/storage"
)

func TestRegisterProviders_PromotesInProcessBackendFamilies(t *testing.T) {
	reg := backends.New()
	if err := RegisterProviders(reg, func(string) (string, error) { return "/bin/x", nil }, nil); err != nil {
		t.Fatalf("register providers: %v", err)
	}

	catalog := []models.Model{
		{ID: "normals-from-depth", Backend: models.BackendComputed, Enabled: true, Operations: []string{"normal_map"}},
	}
	report := reg.DoctorForModels(context.Background(), catalog)

	row, ok := backendRow(report, models.BackendComputed)
	if !ok {
		t.Fatalf("missing computed backend row: %+v", report.Backends)
	}
	if !row.Available {
		t.Fatalf("computed should be available without host provisioning: %+v", row)
	}
	if row.GPUCapable {
		t.Fatalf("computed should be CPU-only: %+v", row)
	}
	if row.Provision != "no host provisioning required" {
		t.Fatalf("computed provision = %q", row.Provision)
	}
}

func TestComputedProvider_NormalMapWritesPNG(t *testing.T) {
	req := backends.Request{
		Operation: "normal_map",
		InputKeys: []string{writeDepthPNG(t)},
		Output:    storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "normal.png")},
		Params:    map[string]string{"strength": "4"},
	}
	if err := dispatchComputed(context.Background(), req); err != nil {
		t.Fatalf("normal_map: %v", err)
	}
	f, err := os.Open(req.Output.LocalPath)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	defer func() { _ = f.Close() }()
	out, err := png.Decode(f)
	if err != nil {
		t.Fatalf("output is not PNG: %v", err)
	}
	if out.Bounds().Dx() != 8 || out.Bounds().Dy() != 8 {
		t.Fatalf("normal map dims = %v", out.Bounds())
	}
	center := color.NRGBAModel.Convert(out.At(4, 4)).(color.NRGBA)
	if center.A != 255 || center.B <= center.R || center.B <= center.G {
		t.Fatalf("normal map center should be opaque with dominant Z/blue channel, got %+v", center)
	}
}

func TestComputedProvider_NormalMapGoldenStructuralContract(t *testing.T) {
	req := backends.Request{
		Operation: "normal_map",
		InputKeys: []string{writeDepthPNG(t)},
		Output:    storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "normal.png")},
		Params:    map[string]string{"strength": "4"},
	}
	if err := dispatchComputed(context.Background(), req); err != nil {
		t.Fatalf("normal_map: %v", err)
	}
	f, err := os.Open(req.Output.LocalPath)
	if err != nil {
		t.Fatalf("open output: %v", err)
	}
	defer func() { _ = f.Close() }()
	out, err := png.Decode(f)
	if err != nil {
		t.Fatalf("output is not PNG: %v", err)
	}
	if out.Bounds() != image.Rect(0, 0, 8, 8) {
		t.Fatalf("normal map changed dimensions: %v", out.Bounds())
	}

	left := color.NRGBAModel.Convert(out.At(1, 4)).(color.NRGBA)
	center := color.NRGBAModel.Convert(out.At(4, 4)).(color.NRGBA)
	right := color.NRGBAModel.Convert(out.At(7, 4)).(color.NRGBA)
	for _, px := range []color.NRGBA{left, center, right} {
		if px.A != 255 {
			t.Fatalf("normal map must be fully opaque, got alpha %d in %+v", px.A, px)
		}
		if px.B < 200 {
			t.Fatalf("normal map Z channel should remain dominant/positive, got %+v", px)
		}
	}
	if math.Abs(float64(center.G)-128) > 1 {
		t.Fatalf("horizontal gradient should not tilt the Y/green channel materially, center=%+v", center)
	}
	if left.R >= 90 || center.R >= 55 || right.R >= 90 {
		t.Fatalf("normal map should encode a strong negative X/red component across the gradient: left=%+v center=%+v right=%+v", left, center, right)
	}
	if normalMapChannelVariance(out, 0) <= 1 {
		t.Fatal("normal map red channel is effectively blank; expected gradient-derived X normals")
	}
}

func backendRow(report backends.DoctorReport, name string) (backends.BackendStatus, bool) {
	for _, row := range report.Backends {
		if row.Name == name {
			return row, true
		}
	}
	return backends.BackendStatus{}, false
}

func writeDepthPNG(t *testing.T) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			v := uint8(x * 32)
			img.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	path := filepath.Join(t.TempDir(), "depth.png")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	return path
}

func normalMapChannelVariance(img image.Image, channel int) float64 {
	b := img.Bounds()
	var sum, sumSq float64
	var n float64
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			px := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			vals := [4]uint8{px.R, px.G, px.B, px.A}
			v := float64(vals[channel])
			sum += v
			sumSq += v * v
			n++
		}
	}
	mean := sum / n
	return sumSq/n - mean*mean
}
