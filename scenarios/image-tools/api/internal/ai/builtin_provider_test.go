package ai

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
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
