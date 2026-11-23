package ai

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/disintegration/imaging"
)

func writeFrameImage(t *testing.T, dir, name string, w, h int) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 255 / w), G: uint8(y * 255 / h), B: 90, A: 255})
		}
	}
	path := filepath.Join(dir, name)
	if err := imaging.Save(img, path); err != nil {
		t.Fatal(err)
	}
	return path
}

func frameSize(t *testing.T, path string) (int, int) {
	t.Helper()
	img, err := imaging.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return img.Bounds().Dx(), img.Bounds().Dy()
}

func TestMatchSourceFrame(t *testing.T) {
	cases := []struct {
		name       string
		op         string
		params     map[string]string
		resultW    int
		resultH    int
		wantPolicy string
		wantW      int
		wantH      int
	}{
		{"a model working at its own size is resampled back to the source frame", "edit_instruct", nil, 880, 1168, framePolicyResample, 1152, 1536},
		{"a result already in the frame is left alone", "garment_replace", nil, 1152, 1536, framePolicySource, 1152, 1536},
		{"an upscaled cloud result with the same aspect returns to the source frame", "edit_instruct", nil, 1728, 2304, framePolicyResample, 1152, 1536},
		{"a different aspect is a deliberate crop, never stretched", "edit_instruct", nil, 800, 800, framePolicyModel, 800, 800},
		{"an explicit size is the caller's choice", "edit_instruct", map[string]string{"width": "640", "height": "832"}, 640, 832, framePolicyModel, 640, 832},
		{"an op that changes the frame is not matched", "outpaint", nil, 880, 1168, framePolicyModel, 880, 1168},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := writeFrameImage(t, dir, "source.png", 1152, 1536)
			out := writeFrameImage(t, dir, "out.png", tc.resultW, tc.resultH)
			policy, rewritten, err := matchSourceFrame(tc.op, source, out, tc.params)
			if err != nil {
				t.Fatal(err)
			}
			if policy != tc.wantPolicy {
				t.Fatalf("policy = %q, want %q", policy, tc.wantPolicy)
			}
			if rewritten != (tc.wantPolicy == framePolicyResample) {
				t.Fatalf("rewritten = %v for policy %q", rewritten, policy)
			}
			if w, h := frameSize(t, out); w != tc.wantW || h != tc.wantH {
				t.Fatalf("result is %dx%d, want %dx%d", w, h, tc.wantW, tc.wantH)
			}
		})
	}
}

func TestMatchSourceFrameWithoutASourceOrARaster(t *testing.T) {
	dir := t.TempDir()
	out := writeFrameImage(t, dir, "out.png", 880, 1168)
	if policy, _, err := matchSourceFrame("edit_instruct", "", out, nil); err != nil || policy != framePolicyModel {
		t.Fatalf("no source: policy %q err %v", policy, err)
	}
	garbage := filepath.Join(dir, "garbage.bin")
	if err := os.WriteFile(garbage, []byte("not an image"), 0o600); err != nil {
		t.Fatal(err)
	}
	if policy, rewritten, err := matchSourceFrame("edit_instruct", garbage, out, nil); err != nil || policy != framePolicyModel || rewritten {
		t.Fatalf("undecodable source: policy %q rewritten %v err %v", policy, rewritten, err)
	}
	source := writeFrameImage(t, dir, "source.png", 1152, 1536)
	svg := filepath.Join(dir, "out.svg")
	if err := os.WriteFile(svg, []byte("<svg xmlns='http://www.w3.org/2000/svg'/>"), 0o600); err != nil {
		t.Fatal(err)
	}
	if policy, rewritten, err := matchSourceFrame("edit_instruct", source, svg, nil); err != nil || policy != framePolicyModel || rewritten {
		t.Fatalf("non-raster: policy %q rewritten %v err %v", policy, rewritten, err)
	}
}
