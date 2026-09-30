package analysis

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// makePNG renders a small solid-color PNG for probe tests.
func makePNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// TestProbe_StructuredResult is the IMG-P0-004 probe unit: the pure-Go probe
// returns structured info with no model or GPU.
func TestProbe_StructuredResult(t *testing.T) {
	src := makePNG(t, 1500, 1000, color.RGBA{R: 200, G: 30, B: 40, A: 255})
	res, err := Probe(src)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if res.Width != 1500 || res.Height != 1000 {
		t.Errorf("dimensions = %dx%d, want 1500x1000", res.Width, res.Height)
	}
	if res.Format != "png" {
		t.Errorf("format = %q, want png", res.Format)
	}
	if res.ColorModel != "rgba" {
		t.Errorf("color model = %q, want rgba", res.ColorModel)
	}
	if res.Megapixels <= 0 {
		t.Errorf("megapixels = %v, want > 0", res.Megapixels)
	}
	if res.SizeBytes != int64(len(src)) {
		t.Errorf("size = %d, want %d", res.SizeBytes, len(src))
	}
	if len(res.DominantColors) == 0 {
		t.Fatal("expected at least one dominant color")
	}
	if res.DominantColors[0].Hex == "" {
		t.Error("dominant color hex is empty")
	}
}

// TestOCR_StructuredResult drives OCR with a fake tesseract returning text.
func TestOCR_StructuredResult(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		LookPath:       func(string) (string, error) { return "/usr/bin/tesseract", nil },
		Run: func(_ context.Context, name string, args []string) ([]byte, error) {
			if name != "tesseract" {
				t.Errorf("ran %q, want tesseract", name)
			}
			return []byte("hello world\n"), nil
		},
	})
	res, err := svc.OCR(context.Background(), []byte("img"))
	if err != nil {
		t.Fatalf("ocr: %v", err)
	}
	if res.FullText != "hello world" {
		t.Errorf("full text = %q, want %q", res.FullText, "hello world")
	}
	if res.Language != "eng" {
		t.Errorf("language = %q, want eng", res.Language)
	}
}

// TestOCR_BackendUnavailable refuses cleanly when tesseract is absent.
func TestOCR_BackendUnavailable(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		LookPath:       func(string) (string, error) { return "", errors.New("not found") },
	})
	_, err := svc.OCR(context.Background(), []byte("img"))
	if !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("err = %v, want ErrBackendUnavailable", err)
	}
}

// TestNSFW_StructuredResult drives NSFW with a fake onnx sidecar returning JSON.
func TestNSFW_StructuredResult(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		NSFWThreshold:  0.5,
		LookPath:       func(string) (string, error) { return "/usr/bin/python3", nil },
		Run: func(context.Context, string, []string) ([]byte, error) {
			return []byte(`{"score":0.92,"categories":[{"label":"nsfw","score":0.92},{"label":"sfw","score":0.08}]}`), nil
		},
	})
	res, err := svc.NSFW(context.Background(), []byte("img"))
	if err != nil {
		t.Fatalf("nsfw: %v", err)
	}
	if !res.NSFW {
		t.Errorf("nsfw = false, want true (score %.2f >= %.2f)", res.Score, res.Threshold)
	}
	if res.Label != "nsfw" {
		t.Errorf("label = %q, want nsfw", res.Label)
	}
	if len(res.Categories) != 2 {
		t.Errorf("categories = %d, want 2", len(res.Categories))
	}
}

// TestNSFW_ThresholdGovernsVerdict proves the operator-tunable threshold.
func TestNSFW_ThresholdGovernsVerdict(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return true },
		NSFWThreshold:  0.95, // strict: 0.92 should read as SFW
		LookPath:       func(string) (string, error) { return "/usr/bin/python3", nil },
		Run: func(context.Context, string, []string) ([]byte, error) {
			return []byte(`{"score":0.92,"categories":[]}`), nil
		},
	})
	res, err := svc.NSFW(context.Background(), []byte("img"))
	if err != nil {
		t.Fatalf("nsfw: %v", err)
	}
	if res.NSFW {
		t.Error("expected SFW at a 0.95 threshold with score 0.92")
	}
}

// TestNSFW_ModelNotInstalled refuses when the model weights are absent.
func TestNSFW_ModelNotInstalled(t *testing.T) {
	svc := mustService(t, Config{
		ModelInstalled: func(string) bool { return false },
		LookPath:       func(string) (string, error) { return "/usr/bin/python3", nil },
	})
	_, err := svc.NSFW(context.Background(), []byte("img"))
	if !errors.Is(err, ErrBackendUnavailable) {
		t.Fatalf("err = %v, want ErrBackendUnavailable", err)
	}
}

func mustService(t *testing.T, cfg Config) *Service {
	t.Helper()
	svc, err := NewService(cfg)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}
