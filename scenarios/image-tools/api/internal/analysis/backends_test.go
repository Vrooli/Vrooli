package analysis

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/storage"
)

func TestRegisterBackendProviders_PromotesStructuredBackendFamilies(t *testing.T) {
	reg := backends.New()
	if err := RegisterBackendProviders(reg); err != nil {
		t.Fatalf("register providers: %v", err)
	}

	report := reg.DoctorForModels(context.Background(), []models.Model{
		{ID: "laplacian-blur", Backend: models.BackendComputed, Enabled: true, Operations: []string{OpQuality}},
		{ID: "goimagehash", Backend: models.BackendLibraryGo, Enabled: true, Operations: []string{OpDuplicate}},
		{ID: "gozxing", Backend: models.BackendLibraryGo, Enabled: true, Operations: []string{"qr_barcode_read"}},
		{ID: "tesseract", Backend: "library-cgo", Enabled: true, Operations: []string{OpOCR}},
	})
	for _, name := range []string{models.BackendComputed, models.BackendLibraryGo} {
		row, ok := backendRow(report, name)
		if !ok {
			t.Fatalf("missing %s backend row: %+v", name, report.Backends)
		}
		if !row.Available {
			t.Fatalf("%s should be available without host provisioning: %+v", name, row)
		}
		if row.GPUCapable {
			t.Fatalf("%s should be CPU-only: %+v", name, row)
		}
		if row.Provision != "no host provisioning required" {
			t.Fatalf("%s provision = %q", name, row.Provision)
		}
	}
}

func TestDoctorForModelsReportsMissingBackendOperations(t *testing.T) {
	reg := backends.New()
	if err := reg.Register(&tesseractProvider{lookPath: func(string) (string, error) { return "/usr/bin/tesseract", nil }}); err != nil {
		t.Fatalf("register tesseract provider: %v", err)
	}

	report := reg.DoctorForModels(context.Background(), []models.Model{
		{ID: "tesseract", Backend: "library-cgo", Enabled: true, Operations: []string{OpOCR}},
		{ID: "yunet", Backend: "library-cgo", Enabled: true, Operations: []string{"face_detection"}},
	})

	var sawOCR, sawFaceGap bool
	for _, row := range report.Backends {
		if row.Name != "library-cgo" {
			continue
		}
		for _, op := range row.Operations {
			switch op {
			case OpOCR:
				sawOCR = row.Available
			case "face_detection":
				sawFaceGap = !row.Available
			}
		}
	}
	if !sawOCR {
		t.Fatalf("expected available library-cgo OCR row: %+v", report.Backends)
	}
	if !sawFaceGap {
		t.Fatalf("expected unavailable library-cgo face_detection gap row: %+v", report.Backends)
	}
}

func TestTesseractProvider_AvailabilityAndExecute(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "ocr.json")
	provider := &tesseractProvider{
		lookPath: func(string) (string, error) { return "/usr/bin/tesseract", nil },
		run: func(_ context.Context, name string, args []string) ([]byte, error) {
			if name != "tesseract" {
				t.Fatalf("program = %q", name)
			}
			if len(args) != 4 || args[1] != "stdout" || args[2] != "-l" || args[3] != "eng" {
				t.Fatalf("bad tesseract argv: %v", args)
			}
			return []byte("hello from OCR\n"), nil
		},
	}
	if !provider.Available(context.Background()) {
		t.Fatal("provider should be available when tesseract resolves")
	}
	req := backends.Request{
		Operation: OpOCR,
		InputKeys: []string{writePNG(t)},
		Output:    storage.OutputTarget{LocalPath: outPath},
	}
	res, err := provider.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.OutputRef != outPath {
		t.Fatalf("output ref = %q", res.OutputRef)
	}
	var got OCRResult
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if got.FullText != "hello from OCR" || got.Language != "eng" {
		t.Fatalf("bad OCR payload: %+v", got)
	}
}

func TestTesseractProvider_AvailabilityMissingBinary(t *testing.T) {
	provider := &tesseractProvider{lookPath: func(string) (string, error) { return "", errors.New("not found") }}
	a := provider.Availability(context.Background())
	if a.Available {
		t.Fatalf("provider should be unavailable when tesseract is absent")
	}
	if a.Detail == "" || a.Provision == "" {
		t.Fatalf("availability should include detail and provisioning guidance: %+v", a)
	}
}

func TestYuNetProvider_AvailabilityAndExecute(t *testing.T) {
	outPath := filepath.Join(t.TempDir(), "faces.json")
	var gotProgram string
	var gotArgs []string
	provider := &yuNetProvider{
		lookPath: func(name string) (string, error) {
			if name != "python3" {
				t.Fatalf("lookPath name = %q", name)
			}
			return "/usr/bin/python3", nil
		},
		checkPy: func(_ context.Context, python string, modules []string) error {
			if python != "/usr/bin/python3" {
				t.Fatalf("python = %q", python)
			}
			if strings.Join(modules, ",") != "cv2,numpy" {
				t.Fatalf("modules = %v", modules)
			}
			return nil
		},
		run: func(_ context.Context, name string, args []string) ([]byte, error) {
			gotProgram = name
			gotArgs = append([]string(nil), args...)
			return nil, nil
		},
	}
	if !provider.Available(context.Background()) {
		t.Fatal("provider should be available when python and imports are ready")
	}
	req := backends.Request{
		Operation: "face_detection",
		Model: models.Model{
			ID:      "yunet",
			Backend: "library-cgo",
			Source: models.Source{Assets: []models.Asset{{
				Filename: "face_detection_yunet_2023mar.onnx",
				Kind:     models.ArtifactONNX,
			}}},
		},
		ModelDir:  "/models/yunet",
		InputKeys: []string{writePNG(t)},
		Output:    storage.OutputTarget{LocalPath: outPath},
	}
	res, err := provider.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if res.OutputRef != outPath {
		t.Fatalf("output ref = %q", res.OutputRef)
	}
	if gotProgram != "/usr/bin/python3" {
		t.Fatalf("program = %q", gotProgram)
	}
	wantModel := filepath.Join("/models/yunet", "face_detection_yunet_2023mar.onnx")
	wantSubseq := []string{"-m", "image_tools_sidecar.face_detection", "--model", wantModel, "--image", req.InputKeys[0], "--out", outPath}
	if strings.Join(gotArgs, "\x00") != strings.Join(wantSubseq, "\x00") {
		t.Fatalf("bad YuNet argv:\n got: %v\nwant: %v", gotArgs, wantSubseq)
	}
}

func TestYuNetProvider_AvailabilityMissingOpenCV(t *testing.T) {
	provider := &yuNetProvider{
		lookPath: func(string) (string, error) { return "/usr/bin/python3", nil },
		checkPy:  func(context.Context, string, []string) error { return errors.New("no cv2") },
	}
	a := provider.Availability(context.Background())
	if a.Available {
		t.Fatalf("provider should be unavailable when OpenCV imports fail")
	}
	if !strings.Contains(a.Detail, "OpenCV imports failed") || a.Provision == "" {
		t.Fatalf("availability should include OpenCV detail and provisioning guidance: %+v", a)
	}
}

func TestLibraryGoProvider_DuplicateWritesStructuredJSON(t *testing.T) {
	req := backends.Request{
		Operation: OpDuplicate,
		InputKeys: []string{writePNG(t)},
		Output:    storage.OutputTarget{LocalPath: filepath.Join(t.TempDir(), "duplicate.json")},
	}
	if err := runLibraryGoProvider(context.Background(), req); err != nil {
		t.Fatalf("duplicate_detect: %v", err)
	}
	data, err := os.ReadFile(req.Output.LocalPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	var payload struct {
		PhashHex string
		AhashHex string
		HashBits int
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("output is not JSON: %v (%s)", err, data)
	}
	if payload.HashBits != 64 || len(payload.PhashHex) != 16 || len(payload.AhashHex) != 16 {
		t.Fatalf("bad duplicate payload: %+v", payload)
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

func writePNG(t *testing.T) string {
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
	path := filepath.Join(t.TempDir(), "input.png")
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatalf("write input: %v", err)
	}
	return path
}
