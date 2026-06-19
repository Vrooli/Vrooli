package analysis

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func solidPNGBytes(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

// checkerPNG renders a high-frequency checkerboard — maximally "sharp".
func checkerPNG(t *testing.T, w, h, cell int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(0)
			if ((x/cell)+(y/cell))%2 == 0 {
				v = 255
			}
			img.SetNRGBA(x, y, color.NRGBA{R: v, G: v, B: v, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestQualityAssess_FlatImageIsBlurryAndLowContrast(t *testing.T) {
	// A solid mid-gray image has zero sharpness and zero contrast.
	res, err := QualityAssess(solidPNGBytes(t, 64, 64, color.NRGBA{R: 128, G: 128, B: 128, A: 255}))
	if err != nil {
		t.Fatalf("QualityAssess: %v", err)
	}
	if !res.Blurry {
		t.Errorf("a flat image should be judged blurry; sharpness=%v", res.Sharpness)
	}
	if res.Contrast != 0 {
		t.Errorf("flat image contrast = %v, want 0", res.Contrast)
	}
	if res.Exposure != "well-exposed" {
		t.Errorf("mid-gray exposure = %q, want well-exposed", res.Exposure)
	}
	if len(res.Notes) == 0 {
		t.Error("expected a low-contrast note")
	}
}

func TestQualityAssess_SharpImageScoresHigher(t *testing.T) {
	flat, err := QualityAssess(solidPNGBytes(t, 64, 64, color.NRGBA{R: 128, G: 128, B: 128, A: 255}))
	if err != nil {
		t.Fatal(err)
	}
	sharp, err := QualityAssess(checkerPNG(t, 64, 64, 4))
	if err != nil {
		t.Fatal(err)
	}
	if sharp.Sharpness <= flat.Sharpness {
		t.Errorf("checker sharpness %v should exceed flat %v", sharp.Sharpness, flat.Sharpness)
	}
	if sharp.Blurry {
		t.Error("a high-frequency checker should not be judged blurry")
	}
	if sharp.OverallScore <= flat.OverallScore {
		t.Errorf("checker score %v should exceed flat %v", sharp.OverallScore, flat.OverallScore)
	}
}

func TestQualityAssess_Exposure(t *testing.T) {
	dark, _ := QualityAssess(solidPNGBytes(t, 32, 32, color.NRGBA{R: 10, G: 10, B: 10, A: 255}))
	if dark.Exposure != "underexposed" {
		t.Errorf("dark exposure = %q, want underexposed", dark.Exposure)
	}
	bright, _ := QualityAssess(solidPNGBytes(t, 32, 32, color.NRGBA{R: 250, G: 250, B: 250, A: 255}))
	if bright.Exposure != "overexposed" {
		t.Errorf("bright exposure = %q, want overexposed", bright.Exposure)
	}
}

func TestQualityAssess_DecodeError(t *testing.T) {
	if _, err := QualityAssess([]byte("not an image")); err == nil {
		t.Error("expected a decode error")
	}
}

func TestDuplicateDetect_StableAndDistinct(t *testing.T) {
	a := checkerPNG(t, 96, 96, 8)
	res1, err := DuplicateDetect(a)
	if err != nil {
		t.Fatalf("DuplicateDetect: %v", err)
	}
	if res1.HashBits != 64 {
		t.Errorf("hash bits = %d, want 64", res1.HashBits)
	}
	if len(res1.PhashHex) != 16 || len(res1.AhashHex) != 16 {
		t.Errorf("hashes should be 16 hex chars: phash=%q ahash=%q", res1.PhashHex, res1.AhashHex)
	}
	// Deterministic: the same image hashes identically.
	res2, _ := DuplicateDetect(a)
	if res1.PhashHex != res2.PhashHex || res1.AhashHex != res2.AhashHex {
		t.Error("duplicate fingerprints must be deterministic for the same image")
	}
	// A visibly different image should produce a different pHash.
	other := checkerPNG(t, 96, 96, 3)
	resOther, _ := DuplicateDetect(other)
	if resOther.PhashHex == res1.PhashHex {
		t.Error("distinct images should produce distinct pHashes")
	}
}

func TestDuplicateDetect_DecodeError(t *testing.T) {
	if _, err := DuplicateDetect([]byte("nope")); err == nil {
		t.Error("expected a decode error")
	}
}
